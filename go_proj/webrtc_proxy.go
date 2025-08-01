package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// WebRTCProxy WebRTC代理服务器
type WebRTCProxy struct {
	connections map[string]*Go2Connection
	mutex       sync.RWMutex
	upgrader    websocket.Upgrader
}

// ProxyRequest 代理请求结构
type ProxyRequest struct {
	Action  string      `json:"action"`
	RobotIP string      `json:"robot_ip"`
	Token   string      `json:"token"`
	Command string      `json:"command,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ProxyResponse 代理响应结构
type ProxyResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// VideoFrame 视频帧结构
type VideoFrame struct {
	FrameData []byte `json:"frame_data"`
	FrameType string `json:"frame_type"`
	Timestamp uint32 `json:"timestamp"`
}

// NewWebRTCProxy 创建新的WebRTC代理
func NewWebRTCProxy() *WebRTCProxy {
	return &WebRTCProxy{
		connections: make(map[string]*Go2Connection),
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
			log.Printf("机器人验证成功: %s", connectionID)
		},
		func(message interface{}, msgObj interface{}) {
			log.Printf("收到消息: %v", message)
		},
		func() {
			log.Printf("数据通道已打开: %s", connectionID)
		},
	)

	// 设置视频帧回调
	conn.SetVideoFrameCallback(func(frameData []byte, frameType string, timestamp uint32) {
		// 这里可以处理视频帧数据
		log.Printf("收到视频帧: 类型=%s, 时间戳=%d, 大小=%d字节", frameType, timestamp, len(frameData))
	})

	// 连接到机器人
	if err := conn.ConnectRobot(); err != nil {
		http.Error(w, fmt.Sprintf("连接机器人失败: %v", err), http.StatusInternalServerError)
		return
	}

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

// handleWebSocket WebSocket处理
func (proxy *WebRTCProxy) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := proxy.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("WebSocket连接已建立")

	// 处理WebSocket消息
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("读取WebSocket消息失败: %v", err)
			break
		}

		var req ProxyRequest
		if err := json.Unmarshal(message, &req); err != nil {
			log.Printf("解析WebSocket消息失败: %v", err)
			continue
		}

		// 处理不同类型的请求
		switch req.Action {
		case "connect":
			proxy.handleWebSocketConnect(conn, req)
		case "command":
			proxy.handleWebSocketCommand(conn, req)
		case "video":
			proxy.handleWebSocketVideo(conn, req)
		default:
			log.Printf("未知的WebSocket动作: %s", req.Action)
		}
	}
}

// handleWebSocketConnect 处理WebSocket连接请求
func (proxy *WebRTCProxy) handleWebSocketConnect(conn *websocket.Conn, req ProxyRequest) {
	connectionID := proxy.generateConnectionID(req.RobotIP, req.Token)

	proxy.mutex.Lock()
	defer proxy.mutex.Unlock()

	// 检查是否已存在连接
	if _, exists := proxy.connections[connectionID]; exists {
		response := ProxyResponse{
			Success: true,
			Message: "连接已存在",
		}
		conn.WriteJSON(response)
		return
	}

	// 创建新的机器人连接
	go2Conn := NewGo2Connection(
		req.RobotIP,
		req.Token,
		func() {
			log.Printf("机器人验证成功: %s", connectionID)
			// 发送验证成功消息到WebSocket
			response := ProxyResponse{
				Success: true,
				Message: "验证成功",
			}
			conn.WriteJSON(response)
		},
		func(message interface{}, msgObj interface{}) {
			log.Printf("收到消息: %v", message)
			// 发送消息到WebSocket
			response := ProxyResponse{
				Success: true,
				Message: "收到消息",
				Data:    message,
			}
			conn.WriteJSON(response)
		},
		func() {
			log.Printf("数据通道已打开: %s", connectionID)
			// 发送连接成功消息到WebSocket
			response := ProxyResponse{
				Success: true,
				Message: "连接成功",
			}
			conn.WriteJSON(response)
		},
	)

	// 设置视频帧回调
	go2Conn.SetVideoFrameCallback(func(frameData []byte, frameType string, timestamp uint32) {
		// 发送视频帧数据到WebSocket
		videoFrame := VideoFrame{
			FrameData: frameData,
			FrameType: frameType,
			Timestamp: timestamp,
		}
		response := ProxyResponse{
			Success: true,
			Message: "视频帧",
			Data:    videoFrame,
		}
		conn.WriteJSON(response)
	})

	// 连接到机器人
	if err := go2Conn.ConnectRobot(); err != nil {
		response := ProxyResponse{
			Success: false,
			Message: fmt.Sprintf("连接机器人失败: %v", err),
		}
		conn.WriteJSON(response)
		return
	}

	// 存储连接
	proxy.connections[connectionID] = go2Conn

	response := ProxyResponse{
		Success: true,
		Message: "连接成功",
		Data: map[string]string{
			"connection_id": connectionID,
		},
	}
	conn.WriteJSON(response)
}

// handleWebSocketCommand 处理WebSocket命令请求
func (proxy *WebRTCProxy) handleWebSocketCommand(conn *websocket.Conn, req ProxyRequest) {
	connectionID := proxy.generateConnectionID(req.RobotIP, req.Token)

	proxy.mutex.RLock()
	go2Conn, exists := proxy.connections[connectionID]
	proxy.mutex.RUnlock()

	if !exists {
		response := ProxyResponse{
			Success: false,
			Message: "连接不存在",
		}
		conn.WriteJSON(response)
		return
	}

	// 发送命令
	go2Conn.SendCommand(req.Command, req.Data)

	response := ProxyResponse{
		Success: true,
		Message: "命令发送成功",
	}
	conn.WriteJSON(response)
}

// handleWebSocketVideo 处理WebSocket视频控制请求
func (proxy *WebRTCProxy) handleWebSocketVideo(conn *websocket.Conn, req ProxyRequest) {
	connectionID := proxy.generateConnectionID(req.RobotIP, req.Token)

	proxy.mutex.RLock()
	go2Conn, exists := proxy.connections[connectionID]
	proxy.mutex.RUnlock()

	if !exists {
		response := ProxyResponse{
			Success: false,
			Message: "连接不存在",
		}
		conn.WriteJSON(response)
		return
	}

	// 根据命令开启或关闭视频
	if req.Command == "open" {
		go2Conn.OpenVideo()
	} else if req.Command == "close" {
		go2Conn.CloseVideo()
	}

	response := ProxyResponse{
		Success: true,
		Message: "视频控制成功",
	}
	conn.WriteJSON(response)
}

// Start 启动代理服务器
func (proxy *WebRTCProxy) Start(port string) {
	router := mux.NewRouter()

	// HTTP API路由
	router.HandleFunc("/api/connect", proxy.handleConnect).Methods("POST")
	router.HandleFunc("/api/disconnect", proxy.handleDisconnect).Methods("POST")
	router.HandleFunc("/api/command", proxy.handleCommand).Methods("POST")
	router.HandleFunc("/api/video", proxy.handleVideo).Methods("POST")

	// WebSocket路由
	router.HandleFunc("/ws", proxy.handleWebSocket)

	// 静态文件服务
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("static")))

	log.Printf("WebRTC代理服务器启动在端口 %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func main() {
	proxy := NewWebRTCProxy()
	proxy.Start("8080")
}
