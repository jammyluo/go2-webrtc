package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

// WebRTCProxy WebRTC代理服务器
type WebRTCProxy struct {
	connections map[string]*Go2Connection
	clients     map[string]*WebRTCClient
	mutex       sync.RWMutex
	upgrader    websocket.Upgrader
}

// WebRTCClient WebRTC客户端结构
type WebRTCClient struct {
	id             string
	peerConnection *webrtc.PeerConnection
	videoTrack     *webrtc.TrackLocalStaticSample
	robotConn      *Go2Connection
	onClose        func()
}

// ProxyRequest 代理请求结构
type ProxyRequest struct {
	Action   string      `json:"action"`
	RobotIP  string      `json:"robot_ip"`
	Token    string      `json:"token"`
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
func NewWebRTCProxy() *WebRTCProxy {
	return &WebRTCProxy{
		connections: make(map[string]*Go2Connection),
		clients:     make(map[string]*WebRTCClient),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源
			},
		},
	}
}

// generateConnectionID 生成连接ID
func (proxy *WebRTCProxy) generateConnectionID(robotIP, token string) string {
	return fmt.Sprintf("%s_%s", robotIP, token)
}

// generateClientID 生成客户端ID
func (proxy *WebRTCProxy) generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}

// NewWebRTCClient 创建新的WebRTC客户端
func NewWebRTCClient(id string, robotConn *Go2Connection) *WebRTCClient {
	config := webrtc.Configuration{
		// 本地局域网环境，不需要STUN服务器
		ICEServers: []webrtc.ICEServer{},
		// 添加ICE配置，优化本地连接
		ICETransportPolicy: webrtc.ICETransportPolicyAll,
		BundlePolicy:       webrtc.BundlePolicyMaxBundle,
		RTCPMuxPolicy:      webrtc.RTCPMuxPolicyRequire,
		// 添加本地连接支持
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
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

	// 设置连接状态变化处理
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("WebRTC客户端 %s 连接状态: %s", id, s.String())
		if s == webrtc.PeerConnectionStateClosed {
			client.onClose()
		}
	})

	// 设置ICE连接状态变化
	peerConnection.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		log.Printf("WebRTC客户端 %s ICE连接状态: %s", id, s.String())
		// 本地网络环境，ICE状态可能不同
		if s == webrtc.ICEConnectionStateFailed {
			log.Printf("WebRTC客户端 %s ICE连接失败，但继续处理视频流（本地网络）", id)
		} else if s == webrtc.ICEConnectionStateConnected {
			log.Printf("WebRTC客户端 %s ICE连接成功", id)
		} else if s == webrtc.ICEConnectionStateChecking {
			log.Printf("WebRTC客户端 %s ICE连接检查中...", id)
		}
	})

	// 设置ICE候选地址处理
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			log.Printf("WebRTC客户端 %s ICE候选地址收集完成", id)
		} else {
			log.Printf("WebRTC客户端 %s 新的ICE候选地址: %s", id, candidate.String())
		}
	})

	// 设置数据通道状态变化
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		log.Printf("WebRTC客户端 %s 数据通道: %s", id, d.Label())
	})

	return client
}

// AddVideoTrack 添加视频轨道到WebRTC客户端
func (client *WebRTCClient) AddVideoTrack() error {
	// 创建视频轨道，使用简化的H.264配置
	videoTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeH264,
			ClockRate: 90000,
			Channels:  0,
			// 使用最基础的H.264配置
			SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f",
		},
		"video",
		"pion",
	)
	if err != nil {
		log.Printf("❌ 创建视频轨道失败: %v", err)
		return fmt.Errorf("创建视频轨道失败: %v", err)
	}

	client.videoTrack = videoTrack

	// 添加轨道到PeerConnection
	_, err = client.peerConnection.AddTrack(videoTrack)
	if err != nil {
		log.Printf("❌ 添加视频轨道失败: %v", err)
		return fmt.Errorf("添加视频轨道失败: %v", err)
	}

	log.Printf("✅ WebRTC客户端 %s 视频轨道已添加", client.id)
	return nil
}

// WriteVideoSample 写入视频样本
func (client *WebRTCClient) WriteVideoSample(frameData []byte, timestamp uint32) error {
	if client.videoTrack != nil {
		// 本地网络环境，更宽松的连接状态检查
		connectionState := client.peerConnection.ConnectionState()
		iceState := client.peerConnection.ICEConnectionState()
		log.Printf("🎬 客户端 %s 连接状态: %s, ICE状态: %s, 视频帧大小: %d 字节",
			client.id, connectionState.String(), iceState.String(), len(frameData))

		// 本地网络环境下，即使ICE失败也尝试写入视频
		if connectionState == webrtc.PeerConnectionStateConnected ||
			connectionState == webrtc.PeerConnectionStateConnecting ||
			connectionState == webrtc.PeerConnectionStateNew ||
			(connectionState == webrtc.PeerConnectionStateFailed && iceState == webrtc.ICEConnectionStateFailed) {

			// 检查视频帧数据
			if len(frameData) == 0 {
				log.Printf("⚠️ 客户端 %s 收到空视频帧", client.id)
				return nil
			}

			// 创建媒体样本
			sample := media.Sample{
				Data:            frameData,
				Duration:        time.Second / 30, // 30fps
				PacketTimestamp: timestamp,
			}

			// 写入视频样本
			err := client.videoTrack.WriteSample(sample)
			if err != nil {
				log.Printf("❌ 客户端 %s 写入视频样本失败: %v", client.id, err)
				return err
			} else {
				log.Printf("✅ 客户端 %s 成功写入视频样本: %d 字节", client.id, len(frameData))
			}

			return nil
		} else {
			log.Printf("⚠️ WebRTC客户端 %s 连接状态不佳 (%s, ICE: %s)，跳过视频帧",
				client.id, connectionState.String(), iceState.String())
			return nil
		}
	}
	log.Printf("❌ 客户端 %s 视频轨道未初始化", client.id)
	return fmt.Errorf("视频轨道未初始化")
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

// handleConnect 处理连接请求
func (proxy *WebRTCProxy) handleConnect(w http.ResponseWriter, r *http.Request) {
	var req ProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	connectionID := proxy.generateConnectionID(req.RobotIP, req.Token)

	proxy.mutex.Lock()
	defer proxy.mutex.Unlock()

	// 检查是否已存在连接
	if _, exists := proxy.connections[connectionID]; exists {
		json.NewEncoder(w).Encode(ProxyResponse{
			Success: true,
			Message: "连接已存在",
		})
		return
	}

	// 创建新的机器人连接
	conn := NewGo2Connection(
		req.RobotIP,
		req.Token,
		func() {
			log.Printf("PROXY 机器人验证成功: %s", connectionID)
			// 自动开启视频流
			proxy.connections[connectionID].OpenVideo()
			log.Printf("PROXY 机器人连接成功，视频流已自动开启: %s", connectionID)
		},
		func(message interface{}, msgObj interface{}) {
			log.Printf("PROXY 收到消息: %v", message)
		},
		func() {
			log.Printf("PROXY 数据通道已打开: %s", connectionID)
		},
	)

	// 设置视频帧回调，转发给所有WebRTC客户端
	conn.SetVideoFrameCallback(func(frameData []byte, frameType string, timestamp uint32) {
		proxy.broadcastVideoFrame(connectionID, frameData, frameType, timestamp)
	})

	// 连接到机器人
	if err := conn.ConnectRobot(); err != nil {
		http.Error(w, fmt.Sprintf("连接机器人失败: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("机器人连接成功: %s", connectionID)

	// 存储连接
	proxy.connections[connectionID] = conn

	json.NewEncoder(w).Encode(ProxyResponse{
		Success: true,
		Message: "连接成功",
		Data: map[string]string{
			"connection_id": connectionID,
		},
	})
}

// broadcastVideoFrame 广播视频帧给所有WebRTC客户端
func (proxy *WebRTCProxy) broadcastVideoFrame(connectionID string, frameData []byte, frameType string, timestamp uint32) {
	proxy.mutex.RLock()
	defer proxy.mutex.RUnlock()

	// 解析frameType中的RTP信息
	var rtpInfo map[string]interface{}
	if frameType != "" {
		if err := json.Unmarshal([]byte(frameType), &rtpInfo); err != nil {
			log.Printf("解析RTP信息失败: %v", err)
			return
		}
	}

	// 详细记录视频帧信息
	// log.Printf("🎬 收到视频帧: 连接ID=%s, 大小=%d字节, 时间戳=%d", connectionID, len(frameData), timestamp)
	// if rtpInfo != nil {
	// 	if payloadType, ok := rtpInfo["payload_type"].(float64); ok {
	// 		log.Printf("🎬 视频编码类型: %v", payloadType)
	// 	}
	// 	if sequence, ok := rtpInfo["sequence"].(float64); ok {
	// 		log.Printf("🎬 序列号: %v", sequence)
	// 	}
	// }

	// // 检查视频帧数据
	// if len(frameData) > 0 {
	// 	// 检查H.264 NAL单元
	// 	if len(frameData) >= 4 {
	// 		nalType := frameData[4] & 0x1F
	// 		log.Printf("🎬 NAL单元类型: %d", nalType)
	// 		if nalType == 7 {
	// 			log.Printf("🎬 检测到SPS帧")
	// 		} else if nalType == 8 {
	// 			log.Printf("🎬 检测到PPS帧")
	// 		} else if nalType == 5 {
	// 			log.Printf("🎬 检测到关键帧")
	// 		} else if nalType == 1 {
	// 			log.Printf("🎬 检测到P帧")
	// 		}
	// 	}
	// }

	// 广播给所有连接到该机器人的客户端
	clientCount := 0
	for clientID, client := range proxy.clients {
		if client.robotConn == proxy.connections[connectionID] {
			if err := client.WriteVideoSample(frameData, timestamp); err != nil {
				log.Printf("❌ 向客户端 %s 写入视频样本失败: %v", clientID, err)
			} else {
				clientCount++
				log.Printf("✅ 成功向客户端 %s 写入视频样本: %d 字节", clientID, len(frameData))
			}
		}
	}

	// if clientCount > 0 {
	// 	log.Printf("📡 向 %d 个客户端广播视频帧: %d 字节", clientCount, len(frameData))
	// } else {
	// 	log.Printf("⚠️ 没有客户端接收视频帧")
	// }
}

// handleWebRTCClient 处理WebRTC客户端连接
func (proxy *WebRTCProxy) handleWebRTCClient(w http.ResponseWriter, r *http.Request) {
	// 获取机器人连接ID
	robotIP := r.URL.Query().Get("robot_ip")
	token := r.URL.Query().Get("token")
	log.Printf("收到WebRTC客户端请求: robot_ip=%s, token=%s", robotIP, token)

	if robotIP == "" {
		log.Printf("缺少参数: robot_ip=%s, token=%s", robotIP, token)
		http.Error(w, "缺少robot_ip或token参数", http.StatusBadRequest)
		return
	}

	connectionID := proxy.generateConnectionID(robotIP, token)
	clientID := proxy.generateClientID()
	log.Printf("生成连接ID: %s, 客户端ID: %s", connectionID, clientID)

	proxy.mutex.RLock()
	robotConn, exists := proxy.connections[connectionID]
	proxy.mutex.RUnlock()

	if !exists {
		log.Printf("机器人连接不存在: %s", connectionID)
		http.Error(w, "机器人连接不存在", http.StatusNotFound)
		return
	}

	log.Printf("找到机器人连接: %s", connectionID)

	// 创建WebRTC客户端
	client := NewWebRTCClient(clientID, robotConn)
	if client == nil {
		log.Printf("创建WebRTC客户端失败: %s", clientID)
		http.Error(w, "创建WebRTC客户端失败", http.StatusInternalServerError)
		return
	}

	log.Printf("WebRTC客户端创建成功: %s", clientID)

	// 添加视频轨道
	if err := client.AddVideoTrack(); err != nil {
		log.Printf("添加视频轨道失败: %v", err)
		client.Close()
		http.Error(w, "添加视频轨道失败", http.StatusInternalServerError)
		return
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

	connectionID := proxy.generateConnectionID(req.RobotIP, req.Token)

	proxy.mutex.Lock()
	defer proxy.mutex.Unlock()

	if conn, exists := proxy.connections[connectionID]; exists {
		conn.Close()
		delete(proxy.connections, connectionID)

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

	connectionID := proxy.generateConnectionID(req.RobotIP, req.Token)

	proxy.mutex.RLock()
	conn, exists := proxy.connections[connectionID]
	proxy.mutex.RUnlock()

	if !exists {
		http.Error(w, "连接不存在", http.StatusNotFound)
		return
	}

	// 发送命令
	conn.SendCommand(req.Command, req.Data)

	json.NewEncoder(w).Encode(ProxyResponse{
		Success: true,
		Message: "命令发送成功",
	})
}

// handleVideo 处理视频控制请求
func (proxy *WebRTCProxy) handleVideo(w http.ResponseWriter, r *http.Request) {
	var req ProxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	connectionID := proxy.generateConnectionID(req.RobotIP, req.Token)

	proxy.mutex.RLock()
	conn, exists := proxy.connections[connectionID]
	proxy.mutex.RUnlock()

	if !exists {
		http.Error(w, "连接不存在", http.StatusNotFound)
		return
	}

	// 根据命令开启或关闭视频
	if req.Command == "open" {
		conn.OpenVideo()
	} else if req.Command == "close" {
		conn.CloseVideo()
	}

	json.NewEncoder(w).Encode(ProxyResponse{
		Success: true,
		Message: "视频控制成功",
	})
}

// Start 启动代理服务器
func (proxy *WebRTCProxy) Start(port string) {
	router := mux.NewRouter()

	// HTTP API路由
	router.HandleFunc("/api/connect", proxy.handleConnect).Methods("POST")
	router.HandleFunc("/api/disconnect", proxy.handleDisconnect).Methods("POST")
	router.HandleFunc("/api/command", proxy.handleCommand).Methods("POST")
	router.HandleFunc("/api/video", proxy.handleVideo).Methods("POST")

	// WebRTC客户端路由
	router.HandleFunc("/webrtc/client", proxy.handleWebRTCClient).Methods("GET")
	router.HandleFunc("/webrtc/answer", proxy.handleWebRTCAnswer).Methods("POST")

	// 静态文件服务
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("static")))

	log.Printf("WebRTC代理服务器启动在端口 %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func main() {
	proxy := NewWebRTCProxy()
	proxy.Start("8080")
}
