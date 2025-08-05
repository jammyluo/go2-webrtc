package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"go2-webrtc/gpio"

	"github.com/gorilla/mux"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type RobotConn interface {
	Connect(ip string, token string) error
	Close() error
	SendCommand(command string, data interface{})
}

// WebRTCProxy WebRTC代理服务器
type WebRTCProxy struct {
	connections map[string]RobotConn
	clients     map[string]*WebRTCClient
	mutex       sync.RWMutex
	config      *Config
}

// WebRTCClient WebRTC客户端结构
type WebRTCClient struct {
	id               string
	peerConnection   *webrtc.PeerConnection
	videoTrackRTP    *webrtc.TrackLocalStaticRTP
	videoTrackSample *webrtc.TrackLocalStaticSample
	robotConn        RobotConn
	onClose          func()
}

// ProxyRequest 代理请求结构
type ProxyRequest struct {
	Action   string      `json:"action"`
	RobotIP  string      `json:"robot_ip"`
	Token    string      `json:"token"`
	NCode    string      `json:"ncode"`
	Command  string      `json:"command,omitempty"`
	Data     interface{} `json:"data,omitempty"`
	ClientID string      `json:"client_id,omitempty"`
}

// ProxyResponse 代理响应结构
type ProxyResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewWebRTCProxy 创建新的WebRTC代理
func NewWebRTCProxy(config *Config) *WebRTCProxy {
	return &WebRTCProxy{
		connections: make(map[string]RobotConn),
		clients:     make(map[string]*WebRTCClient),
		config:      config,
	}
}

// generateClientID 生成客户端ID
func (proxy *WebRTCProxy) generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}

// NewWebRTCClient 创建新的WebRTC客户端
func NewWebRTCClient(id string, robotConn RobotConn, webrtcConfig *Config) *WebRTCClient {
	config := webrtc.Configuration{
		// 使用配置中的ICE服务器
		ICEServers: webrtcConfig.GetICEServers(),
		// 强制使用所有ICE传输策略
		ICETransportPolicy: webrtc.ICETransportPolicyAll,
		BundlePolicy:       webrtc.BundlePolicyMaxBundle,
		RTCPMuxPolicy:      webrtc.RTCPMuxPolicyRequire,
		SDPSemantics:       webrtc.SDPSemanticsUnifiedPlan,
		// 增加ICE候选地址池大小
		ICECandidatePoolSize: 20,
	}

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Printf("创建WebRTC客户端PeerConnection失败: %v", err)
		return nil
	}

	client := &WebRTCClient{
		id:             id,
		peerConnection: peerConnection,
		robotConn:      robotConn,
	}

	// 设置连接状态变化回调
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("🎉 WebRTC客户端 %s 连接状态变化: %s", id, s.String())
		switch s {
		case webrtc.PeerConnectionStateConnected:
			log.Printf("🎉 WebRTC客户端 %s 连接成功！", id)
		case webrtc.PeerConnectionStateFailed:
			log.Printf("❌ WebRTC客户端 %s 连接失败", id)
			// 连接失败时清理客户端
			if client.onClose != nil {
				client.onClose()
			}
		case webrtc.PeerConnectionStateNew:
			log.Printf("🆕 WebRTC客户端 %s 连接新建状态", id)
		case webrtc.PeerConnectionStateConnecting:
			log.Printf("🔄 WebRTC客户端 %s 连接中...", id)
		case webrtc.PeerConnectionStateDisconnected:
			log.Printf("🔌 WebRTC客户端 %s 连接断开", id)
			// 连接断开时清理客户端
			if client.onClose != nil {
				client.onClose()
			}
		case webrtc.PeerConnectionStateClosed:
			log.Printf("🔒 WebRTC客户端 %s 连接已关闭", id)
			// 连接关闭时清理客户端
			if client.onClose != nil {
				client.onClose()
			}
		}
	})

	// 设置ICE连接状态变化回调
	peerConnection.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		log.Printf("🎉 WebRTC客户端 %s ICE连接状态变化: %s", id, s.String())
		switch s {
		case webrtc.ICEConnectionStateConnected:
			log.Printf("🎉 WebRTC客户端 %s ICE连接成功！", id)
		case webrtc.ICEConnectionStateFailed:
			log.Printf("❌ WebRTC客户端 %s ICE连接失败，但继续处理视频流（本地网络）", id)
		case webrtc.ICEConnectionStateChecking:
			log.Printf("🔍 WebRTC客户端 %s ICE连接检查中...", id)
		case webrtc.ICEConnectionStateNew:
			log.Printf("🆕 WebRTC客户端 %s ICE连接新建状态", id)
		case webrtc.ICEConnectionStateCompleted:
			log.Printf("✅ WebRTC客户端 %s ICE连接完成！", id)
		case webrtc.ICEConnectionStateDisconnected:
			log.Printf("🔌 WebRTC客户端 %s ICE连接断开", id)
		case webrtc.ICEConnectionStateClosed:
			log.Printf("🔒 WebRTC客户端 %s ICE连接已关闭", id)
		}
	})

	// 设置ICE候选地址收集完成回调
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			log.Printf("🎉 WebRTC客户端 %s ICE候选地址收集完成", id)
		} else {
			log.Printf("🎯 WebRTC客户端 %s 新的ICE候选地址: %s", id, candidate.String())
		}
	})

	// 设置轨道回调
	peerConnection.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("🎬 WebRTC客户端 %s 收到远程轨道: %s", id, remoteTrack.Kind().String())
	})

	return client
}

func (client *WebRTCClient) AddVideoTrackStaticSample() error {
	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "pion")
	if err != nil {
		log.Printf("AddVideoTrackStaticSample 创建H.264视频轨道失败: %v", err)
		return fmt.Errorf("创建H.264视频轨道失败: %v", err)
	}
	client.videoTrackSample = videoTrack
	// 添加轨道到PeerConnection
	_, err = client.peerConnection.AddTrack(client.videoTrackSample)
	if err != nil {
		log.Printf("AddVideoTrackStaticSample 添加H.264视频轨道失败: %v", err)
		return fmt.Errorf("添加H.264视频轨道失败: %v", err)
	}

	log.Printf("AddVideoTrackStaticSample WebRTC客户端 %s H.264视频轨道已添加", client.id)
	return nil
}

// AddVideoTrack 添加视频轨道到WebRTC客户端
func (client *WebRTCClient) AddVideoTrackStaticRTP() error {
	videoTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "pion")
	if err != nil {
		log.Printf("AddVideoTrackStaticRTP 创建H.264视频轨道失败: %v", err)
		return fmt.Errorf("创建H.264视频轨道失败: %v", err)
	}
	client.videoTrackRTP = videoTrack
	// 添加轨道到PeerConnection
	_, err = client.peerConnection.AddTrack(client.videoTrackRTP)
	if err != nil {
		log.Printf("AddVideoTrackStaticRTP 添加H.264视频轨道失败: %v", err)
		return fmt.Errorf("添加H.264视频轨道失败: %v", err)
	}

	log.Printf("AddVideoTrackStaticRTP WebRTC客户端 %s H.264视频轨道已添加", client.id)
	return nil
}

// CreateOffer 创建SDP提议
func (client *WebRTCClient) CreateOffer() (*webrtc.SessionDescription, error) {
	offer, err := client.peerConnection.CreateOffer(nil)
	if err != nil {
		return nil, fmt.Errorf("创建提议失败: %v", err)
	}

	err = client.peerConnection.SetLocalDescription(offer)
	if err != nil {
		return nil, fmt.Errorf("设置本地描述失败: %v", err)
	}

	// 等待ICE候选地址收集完成
	log.Printf("⏳ WebRTC客户端 %s 等待ICE候选地址收集...", client.id)
	time.Sleep(3 * time.Second)

	// 获取更新后的本地描述（包含ICE候选地址）
	updatedOffer := client.peerConnection.LocalDescription()
	if updatedOffer != nil {
		log.Printf("✅ WebRTC客户端 %s SDP提议创建成功，包含ICE候选地址", client.id)
		return updatedOffer, nil
	}

	log.Printf("✅ WebRTC客户端 %s SDP提议创建成功", client.id)
	return &offer, nil
}

// SetRemoteDescription 设置远程描述
func (client *WebRTCClient) SetRemoteDescription(answer webrtc.SessionDescription) error {
	return client.peerConnection.SetRemoteDescription(answer)
}

// Close 关闭WebRTC客户端
func (client *WebRTCClient) Close() error {
	if client.peerConnection != nil {
		return client.peerConnection.Close()
	}
	return nil
}

func (proxy *WebRTCProxy) connectGo2(robotIP, token, ncode string) error {
	// 使用配置中的默认机器人IP
	robotIP = proxy.config.GetRobotIP(robotIP)
	token = proxy.config.GetDefaultToken(token)
	ncode = proxy.config.GetDefaultNCode(ncode)

	proxy.mutex.Lock()
	defer proxy.mutex.Unlock()

	// 检查是否已存在连接
	if _, exists := proxy.connections[ncode]; exists {
		log.Printf("connect Go2 失败: %s", ncode)
		return nil
	}

	var conn RobotConn
	if proxy.config.RunMode == "mock" {
		conn = NewMockConn(
			func() {
				log.Printf("PROXY 机器人验证成功: %s", ncode)
			},
			func(message interface{}, msgObj interface{}) {
				log.Printf("PROXY 收到消息: %v", message)
			},
			func() {
				log.Printf("PROXY 数据通道已打开: %s", ncode)
			},
			func(sample media.Sample) error {
				return proxy.broadcastVideoWriteSample(sample)
			},
		)
	} else {
		// 创建新的机器人连接
		conn = NewGo2Connection(
			func() {
				log.Printf("PROXY 机器人验证成功: %s", ncode)
			},
			func(message interface{}, msgObj interface{}) {
				log.Printf("PROXY 收到消息: %v", message)
			},
			func() {
				log.Printf("PROXY 数据通道已打开: %s", ncode)
			},
			func(rtp rtp.Packet) {
				proxy.broadcastVideoWriteRTP(rtp)
			},
		)
	}

	// 连接到机器人
	log.Printf("connect Go2: %s, %s, %s", proxy.config.RunMode, robotIP, token)
	if err := conn.Connect(robotIP, token); err != nil {
		return fmt.Errorf("connect Go2 失败: %v", err)
	}

	log.Printf("connect Go2 成功: %s", ncode)

	// 存储连接
	proxy.connections[ncode] = conn
	return nil
}

// handleConnect 处理连接请求
func (proxy *WebRTCProxy) handleConnect(w http.ResponseWriter, r *http.Request) {
	var req ProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求格式", http.StatusBadRequest)
		return
	}
	err := proxy.connectGo2(req.RobotIP, req.Token, req.NCode)
	if err != nil {
		http.Error(w, "连接机器人失败", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(ProxyResponse{
		Success: true,
		Message: "连接成功",
		Data: map[string]string{
			"NCode": req.NCode,
		},
	})
}

func (proxy *WebRTCProxy) broadcastVideoWriteSample(sample media.Sample) error {
	proxy.mutex.RLock()
	defer proxy.mutex.RUnlock()
	// 统计成功发送的客户端数量
	successCount := 0
	totalCount := 0

	// 向所有WebRTC客户端发送视频帧
	for clientID, client := range proxy.clients {
		if client.robotConn != nil && client.videoTrackSample != nil {
			totalCount++

			// 写入RTP包
			err := client.videoTrackSample.WriteSample(sample)
			if err != nil {
				log.Printf("❌ 客户端 %s 写入视频帧失败: %v", clientID, err)
			} else {
				successCount++
			}
		}
	}
	return nil
}

// broadcastVideoFrame 广播视频帧给所有WebRTC客户端
func (proxy *WebRTCProxy) broadcastVideoWriteRTP(rtp rtp.Packet) {
	proxy.mutex.RLock()
	defer proxy.mutex.RUnlock()

	// 统计成功发送的客户端数量
	successCount := 0
	totalCount := 0

	// 向所有WebRTC客户端发送视频帧
	for clientID, client := range proxy.clients {
		if client.robotConn != nil && client.videoTrackRTP != nil {
			totalCount++

			// 写入RTP包
			err := client.videoTrackRTP.WriteRTP(&rtp)
			if err != nil {
				log.Printf("❌ 客户端 %s 写入视频帧失败: %v", clientID, err)
			} else {
				successCount++
			}
		}
	}
	// log.Printf("🎬 成功发送视频帧: %d/%d", successCount, totalCount)
}

// handleWebRTCClient 处理WebRTC客户端连接
func (proxy *WebRTCProxy) handleWebRTCClient(w http.ResponseWriter, r *http.Request) {
	// 获取机器人连接ID TODO
	ncode := r.URL.Query().Get("ncode")
	log.Printf("收到WebRTC客户端请求: ncode=%s", ncode)
	ncode = proxy.config.GetDefaultNCode(ncode)
	clientID := proxy.generateClientID()
	log.Printf("生成连接ID: %s, 客户端ID: %s", ncode, clientID)

	proxy.mutex.RLock()
	robotConn, exists := proxy.connections[ncode]
	proxy.mutex.RUnlock()

	if !exists {
		log.Printf("Go2连接不存在: %s", ncode)
		http.Error(w, "Go2连接不存在", http.StatusNotFound)
		return
	}

	log.Printf("找到Go2连接: %s", ncode)

	// 创建WebRTC客户端
	client := NewWebRTCClient(clientID, robotConn, proxy.config)
	if client == nil {
		log.Printf("创建WebRTC客户端失败: %s", clientID)
		http.Error(w, "创建WebRTC客户端失败", http.StatusInternalServerError)
		return
	}

	log.Printf("WebRTC客户端创建成功: %s", clientID)

	if proxy.config.RunMode == "mock" {
		// 添加视频轨道
		if err := client.AddVideoTrackStaticSample(); err != nil {
			log.Printf("添加视频轨道失败: %v", err)
			client.Close()
			http.Error(w, "添加视频轨道失败", http.StatusInternalServerError)
			return
		}
	} else {
		// 添加视频轨道
		if err := client.AddVideoTrackStaticRTP(); err != nil {
			log.Printf("添加视频轨道失败: %v", err)
			client.Close()
			http.Error(w, "添加视频轨道失败", http.StatusInternalServerError)
			return
		}
	}

	log.Printf("视频轨道添加成功: %s", clientID)

	// 设置关闭回调
	client.onClose = func() {
		proxy.mutex.Lock()
		delete(proxy.clients, clientID)
		proxy.mutex.Unlock()
		log.Printf("WebRTC客户端 %s 已断开", clientID)
	}

	// 存储客户端
	proxy.mutex.Lock()
	proxy.clients[clientID] = client
	proxy.mutex.Unlock()

	log.Printf("WebRTC客户端已存储: %s", clientID)

	// 创建提议
	offer, err := client.CreateOffer()
	if err != nil {
		log.Printf("创建提议失败: %v", err)
		client.Close()
		http.Error(w, "创建提议失败", http.StatusInternalServerError)
		return
	}

	log.Printf("SDP提议创建成功: %s", clientID)

	// 返回SDP提议
	response := map[string]interface{}{
		"type":      "offer",
		"sdp":       offer.SDP,
		"client_id": clientID,
	}

	log.Printf("返回WebRTC客户端响应: client_id=%s", clientID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleWebRTCAnswer 处理WebRTC应答
func (proxy *WebRTCProxy) handleWebRTCAnswer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClientID string `json:"client_id"`
		Answer   struct {
			Type string `json:"type"`
			SDP  string `json:"sdp"`
		} `json:"answer"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	proxy.mutex.RLock()
	client, exists := proxy.clients[req.ClientID]
	proxy.mutex.RUnlock()

	if !exists {
		http.Error(w, "WebRTC客户端不存在", http.StatusNotFound)
		return
	}

	// 设置远程描述
	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  req.Answer.SDP,
	}

	if err := client.SetRemoteDescription(answer); err != nil {
		log.Printf("设置远程描述失败: %v", err)
		http.Error(w, "设置远程描述失败", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(ProxyResponse{
		Success: true,
		Message: "WebRTC连接建立成功",
	})
}

// handleDisconnect 处理断开连接请求
func (proxy *WebRTCProxy) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	var req ProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	// 使用配置中的默认机器人IP
	ncode := proxy.config.GetDefaultNCode("")

	proxy.mutex.Lock()
	defer proxy.mutex.Unlock()

	if conn, exists := proxy.connections[ncode]; exists {
		conn.Close()
		delete(proxy.connections, ncode)

		// 关闭所有相关的WebRTC客户端
		for clientID, client := range proxy.clients {
			if client.robotConn == conn {
				client.Close()
				delete(proxy.clients, clientID)
			}
		}

		json.NewEncoder(w).Encode(ProxyResponse{
			Success: true,
			Message: "断开连接成功",
		})
	} else {
		json.NewEncoder(w).Encode(ProxyResponse{
			Success: false,
			Message: "连接不存在",
		})
	}
}

// handleCommand 处理机器人命令请求
func (proxy *WebRTCProxy) handleCommand(w http.ResponseWriter, r *http.Request) {
	var req ProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求格式", http.StatusBadRequest)
		return
	}
	ncode := proxy.config.GetDefaultNCode(req.NCode)
	log.Printf("收到命令请求: %v, ncode=%s", req, ncode)
	if req.Command == "Shoot" {
		// 根据运行模式处理射击命令
		if proxy.config.IsRealMode() {
			// 真实模式：使用GPIO控制
			gpioCtrl := gpio.NewGPIOController(proxy.config.GPIO.ShootPin)
			gpioCtrl.Pulse(time.Millisecond * time.Duration(proxy.config.GPIO.PulseDuration))
			log.Printf("Shoot 命令已发送 (真实模式)")
		} else {
			// 模拟模式：仅记录日志
			log.Printf("Shoot 命令已发送 (模拟模式)")
		}
	} else {
		// 使用配置中的默认机器人IP
		proxy.mutex.RLock()
		conn, exists := proxy.connections[ncode]
		proxy.mutex.RUnlock()

		if !exists {
			http.Error(w, "连接不存在", http.StatusNotFound)
			return
		}
		conn.SendCommand(req.Command, req.Data)
	}

	json.NewEncoder(w).Encode(ProxyResponse{
		Success: true,
		Message: "命令发送成功",
	})
}

// cleanupDisconnectedClients 清理断开的客户端
func (proxy *WebRTCProxy) cleanupDisconnectedClients() {
	proxy.mutex.Lock()
	defer proxy.mutex.Unlock()

	cleanedCount := 0
	for clientID, client := range proxy.clients {
		if client.peerConnection != nil {
			state := client.peerConnection.ConnectionState()
			if state == webrtc.PeerConnectionStateFailed ||
				state == webrtc.PeerConnectionStateClosed ||
				state == webrtc.PeerConnectionStateDisconnected {
				delete(proxy.clients, clientID)
				cleanedCount++
				log.Printf("🧹 清理断开的WebRTC客户端: %s (状态: %s)", clientID, state.String())
			}
		}
	}

	if cleanedCount > 0 {
		log.Printf("🧹 清理了 %d 个断开的WebRTC客户端", cleanedCount)
	}
}

// Start 启动代理服务器
func (proxy *WebRTCProxy) Start() {
	router := mux.NewRouter()

	// HTTP API路由
	router.HandleFunc("/api/connect", proxy.handleConnect).Methods("POST")
	router.HandleFunc("/api/disconnect", proxy.handleDisconnect).Methods("POST")
	router.HandleFunc("/api/command", proxy.handleCommand).Methods("POST")

	// WebRTC客户端路由
	router.HandleFunc("/webrtc/client", proxy.handleWebRTCClient).Methods("GET")
	router.HandleFunc("/webrtc/answer", proxy.handleWebRTCAnswer).Methods("POST")

	// 静态文件服务
	router.PathPrefix("/").Handler(http.FileServer(http.Dir(proxy.config.Server.StaticDir)))

	// 启动定期清理任务
	go func() {
		ticker := time.NewTicker(30 * time.Second) // 每30秒清理一次
		defer ticker.Stop()
		for range ticker.C {
			proxy.cleanupDisconnectedClients()
		}
	}()

	log.Printf("WebRTC代理服务器启动在端口 %s", proxy.config.Server.Port)
	log.Fatal(http.ListenAndServe(":"+proxy.config.Server.Port, router))
}

func main() {
	// 加载配置
	config, err := LoadConfig("config.json")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建WebRTC代理
	proxy := NewWebRTCProxy(config)
	proxy.Start()
}
