package main

import (
	"bytes"
	"crypto/aes"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pion/webrtc/v3"
)

// 常量定义
const (
	ValidationType = "validation"
	MessageType    = "msg"
	HeartbeatType  = "heartbeat"
)

// 机器人命令映射
var SportCmd = map[string]int{
	"Damp":               1001,
	"BalanceStand":       1002,
	"StopMove":           1003,
	"StandUp":            1004,
	"StandDown":          1005,
	"RecoveryStand":      1006,
	"Euler":              1007,
	"Move":               1008,
	"Sit":                1009,
	"RiseSit":            1010,
	"SwitchGait":         1011,
	"Trigger":            1012,
	"BodyHeight":         1013,
	"FootRaiseHeight":    1014,
	"SpeedLevel":         1015,
	"Hello":              1016,
	"Stretch":            1017,
	"TrajectoryFollow":   1018,
	"ContinuousGait":     1019,
	"Content":            1020,
	"Wallow":             1021,
	"Dance1":             1022,
	"Dance2":             1023,
	"GetBodyHeight":      1024,
	"GetFootRaiseHeight": 1025,
	"GetSpeedLevel":      1026,
	"SwitchJoystick":     1027,
	"Pose":               1028,
	"Scrape":             1029,
	"FrontFlip":          1030,
	"FrontJump":          1031,
	"FrontPounce":        1032,
	"WiggleHips":         1033,
	"GetState":           1034,
	"EconomicGait":       1035,
	"FingerHeart":        1036,
}

// Go2Connection 机器人连接结构体
type Go2Connection struct {
	ip               string
	token            string
	peerConnection   *webrtc.PeerConnection
	dataChannel      *webrtc.DataChannel
	validationResult string
	onValidated      func()
	onMessage        func(message interface{}, msgObj interface{})
	onOpen           func()
	heartbeatTimer   *time.Timer
	validationKey    string // 保存验证密钥
}

// Message 消息结构体
type Message struct {
	Type  string      `json:"type"`
	Topic string      `json:"topic"`
	Data  interface{} `json:"data"`
}

// SDPOffer SDP提议结构体
type SDPOffer struct {
	ID    string `json:"id"`
	SDP   string `json:"sdp"`
	Type  string `json:"type"`
	Token string `json:"token"`
}

// NewGo2Connection 创建新的Go2连接
func NewGo2Connection(ip, token string, onValidated func(), onMessage func(message interface{}, msgObj interface{}), onOpen func()) *Go2Connection {
	config := webrtc.Configuration{
		// ICEServers: []webrtc.ICEServer{
		// 	{
		// 		URLs: []string{"stun:stun.l.google.com:19302"},
		// 	},
		// },
	}

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Fatal("创建PeerConnection失败:", err)
	}

	conn := &Go2Connection{
		ip:               ip,
		token:            token,
		peerConnection:   peerConnection,
		validationResult: "PENDING",
		onValidated:      onValidated,
		onMessage:        onMessage,
		onOpen:           onOpen,
	}

	// 创建数据通道
	dataChannelInit := webrtc.DataChannelInit{
		ID:         func() *uint16 { id := uint16(1); return &id }(),
		Negotiated: func() *bool { negotiated := false; return &negotiated }(),
	}
	dataChannel, err := peerConnection.CreateDataChannel("data", &dataChannelInit)
	if err != nil {
		log.Fatal("创建数据通道失败:", err)
	}

	conn.dataChannel = dataChannel

	// 设置数据通道事件处理
	dataChannel.OnOpen(func() {
		log.Println("数据通道已打开")
		// 在数据通道打开后立即启动心跳
		conn.startHeartbeat()
		if conn.onOpen != nil {
			conn.onOpen()
		}
	})

	dataChannel.OnClose(func() {
		log.Println("数据通道已关闭")
		conn.stopHeartbeat()
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		conn.handleDataChannelMessage(msg)
	})

	// 设置连接状态变化处理
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("连接状态: %s", s.String())
	})

	return conn
}

// handleDataChannelMessage 处理数据通道消息
func (conn *Go2Connection) handleDataChannelMessage(msg webrtc.DataChannelMessage) {
	if msg.IsString {
		var messageObj Message
		if err := json.Unmarshal(msg.Data, &messageObj); err != nil {
			log.Printf("解析消息失败: %v", err)
			return
		}
		log.Printf("handleDataChannelMessage: %v", messageObj)

		// 检查是否是错误消息
		if messageObj.Type == "err" || messageObj.Type == "errors" {
			log.Printf("收到错误消息: %v", messageObj.Data)
			// 处理验证相关的错误
			if errData, ok := messageObj.Data.(map[string]interface{}); ok {
				if info, exists := errData["info"]; exists && info == "Validation Needed." {
					log.Println("收到验证需要错误，重新发送验证数据")
					// 重新发送验证数据
					if conn.validationResult != "SUCCESS" && conn.validationKey != "" {
						conn.sendValidationData(conn.validationKey)
					}
				}
			} else {
				// 如果Data为nil，记录完整的错误消息
				log.Printf("错误消息Data为nil，完整消息: %+v", messageObj)
			}
			return
		}

		if messageObj.Type == ValidationType {
			conn.validate(messageObj)
		}

		if conn.onMessage != nil {
			conn.onMessage(string(msg.Data), messageObj)
		}
	} else {
		// 机器人不支持二进制数据，记录警告
		log.Printf("收到二进制数据，但机器人不支持二进制数据格式")
	}
}

// validate 验证处理
func (conn *Go2Connection) validate(message Message) {
	log.Printf("验证消息: %v", message)
	if data, ok := message.Data.(string); ok && data == "Validation Ok." {
		conn.validationResult = "SUCCESS"
		log.Println("验证成功，启动心跳")
		// 验证成功后启动心跳
		conn.startHeartbeat()
		if conn.onValidated != nil {
			conn.onValidated()
		}
	} else {
		// 发送加密的验证数据
		if data, ok := message.Data.(string); ok {
			conn.validationKey = data // 保存验证密钥
			conn.sendValidationData(data)
		} else {
			log.Printf("验证消息数据不是字符串类型: %T", message.Data)
		}
	}
}

// sendValidationData 发送验证数据
func (conn *Go2Connection) sendValidationData(key string) {
	encryptedData := conn.encryptKey(key)
	conn.publish("", encryptedData, ValidationType)
}

// publish 发布消息
func (conn *Go2Connection) publish(topic string, data interface{}, msgType string) {
	if conn.dataChannel == nil || conn.dataChannel.ReadyState() != webrtc.DataChannelStateOpen {
		log.Printf("数据通道未打开，无法发送消息")
		return
	}

	payload := Message{
		Type:  msgType,
		Topic: topic,
		Data:  data,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("序列化消息失败: %v", err)
		return
	}

	// 记录原始payload，与Python版本保持一致
	log.Printf("-> Sending message %s", string(jsonData))

	// 发送消息
	err = conn.dataChannel.SendText(string(jsonData))
	if err != nil {
		log.Printf("发送消息失败: %v", err)
		return
	}
}

// encryptKey 加密密钥
func (conn *Go2Connection) encryptKey(key string) string {
	prefixedKey := "UnitreeGo2_" + key
	encrypted := encryptByMD5(prefixedKey)
	return hexToBase64(encrypted)
}

// encryptByMD5 MD5加密 utf-8
func encryptByMD5(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}

// hexToBase64 十六进制转Base64
func hexToBase64(hexStr string) string {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		log.Printf("十六进制解码失败: %v", err)
		return ""
	}
	return base64.StdEncoding.EncodeToString(bytes)
}

// generateAESKey 生成AES密钥
func generateAESKey() string {
	uuid := make([]byte, 16)
	rand.Read(uuid)
	return hex.EncodeToString(uuid)
}

// pad PKCS7填充
func pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

// unpad 移除PKCS7填充
func unpad(data []byte) []byte {
	length := len(data)
	if length == 0 {
		return data
	}
	padding := int(data[length-1])
	if padding > length {
		return data
	}
	return data[:length-padding]
}

// aesEncrypt AES加密
func aesEncrypt(data, key string) string {
	keyBytes := []byte(key)
	if len(keyBytes) > 32 {
		keyBytes = keyBytes[:32]
	} else if len(keyBytes) < 32 {
		keyBytes = append(keyBytes, make([]byte, 32-len(keyBytes))...)
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		log.Printf("创建AES加密器失败: %v", err)
		return ""
	}

	paddedData := pad([]byte(data), aes.BlockSize)
	encrypted := make([]byte, len(paddedData))

	// ECB模式加密
	for i := 0; i < len(paddedData); i += aes.BlockSize {
		block.Encrypt(encrypted[i:i+aes.BlockSize], paddedData[i:i+aes.BlockSize])
	}

	return base64.StdEncoding.EncodeToString(encrypted)
}

// aesDecrypt AES解密
func aesDecrypt(encryptedData, key string) string {
	keyBytes := []byte(key)
	if len(keyBytes) > 32 {
		keyBytes = keyBytes[:32]
	} else if len(keyBytes) < 32 {
		keyBytes = append(keyBytes, make([]byte, 32-len(keyBytes))...)
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		log.Printf("创建AES解密器失败: %v", err)
		return ""
	}

	encryptedBytes, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		log.Printf("Base64解码失败: %v", err)
		return ""
	}

	decrypted := make([]byte, len(encryptedBytes))

	// ECB模式解密
	for i := 0; i < len(encryptedBytes); i += aes.BlockSize {
		block.Decrypt(decrypted[i:i+aes.BlockSize], encryptedBytes[i:i+aes.BlockSize])
	}

	unpadded := unpad(decrypted)
	return string(unpadded)
}

// rsaLoadPublicKey 加载RSA公钥
func rsaLoadPublicKey(pemData string) (*rsa.PublicKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(pemData)
	if err != nil {
		return nil, err
	}

	publicKey, err := x509.ParsePKIXPublicKey(keyBytes)
	if err != nil {
		return nil, err
	}

	if rsaKey, ok := publicKey.(*rsa.PublicKey); ok {
		return rsaKey, nil
	}
	return nil, fmt.Errorf("不是RSA公钥")
}

// rsaEncrypt RSA加密
func rsaEncrypt(data string, publicKey *rsa.PublicKey) string {
	dataBytes := []byte(data)
	maxChunkSize := publicKey.Size() - 11
	var encryptedBytes []byte

	for i := 0; i < len(dataBytes); i += maxChunkSize {
		end := i + maxChunkSize
		if end > len(dataBytes) {
			end = len(dataBytes)
		}
		chunk := dataBytes[i:end]

		encryptedChunk, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, chunk)
		if err != nil {
			log.Printf("RSA加密失败: %v", err)
			return ""
		}
		encryptedBytes = append(encryptedBytes, encryptedChunk...)
	}

	return base64.StdEncoding.EncodeToString(encryptedBytes)
}

// calcLocalPathEnding 计算本地路径结尾
func calcLocalPathEnding(data1 string) string {
	strArr := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}

	if len(data1) < 10 {
		return ""
	}

	last10Chars := data1[len(data1)-10:]
	var arrayList []int

	for i := 0; i < len(last10Chars); i += 2 {
		if i+1 < len(last10Chars) {
			secondChar := string(last10Chars[i+1])
			for j, str := range strArr {
				if str == secondChar {
					arrayList = append(arrayList, j)
					break
				}
			}
		}
	}

	var result strings.Builder
	for _, index := range arrayList {
		result.WriteString(strconv.Itoa(index))
	}

	return result.String()
}

// makeLocalRequest 发送本地请求
func makeLocalRequest(path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("POST", path, body)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}

// getPeerAnswer 获取对等方应答
func (conn *Go2Connection) getPeerAnswer(sdpOffer *webrtc.SessionDescription, ip, token string) (map[string]interface{}, error) {
	sdpOfferJSON := SDPOffer{
		ID:    "STA_localNetwork",
		SDP:   sdpOffer.SDP,
		Type:  sdpOffer.Type.String(),
		Token: token,
	}

	newSDP, err := json.Marshal(sdpOfferJSON)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("http://%s:9991/con_notify", ip)
	resp, err := makeLocalRequest(url, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解码Base64响应
	decodedResponse, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		return nil, err
	}

	var decodedJSON map[string]interface{}
	if err := json.Unmarshal(decodedResponse, &decodedJSON); err != nil {
		return nil, err
	}

	log.Printf("getPeerAnswer I newSDP: %s", string(newSDP))
	log.Printf("getPeerAnswer I url: %s", url)
	log.Printf("getPeerAnswer I resp: %s", decodedJSON)

	data1, ok := decodedJSON["data1"].(string)
	if !ok {
		return nil, fmt.Errorf("data1字段不存在")
	}

	// 提取公钥
	publicKeyPEM := data1[10 : len(data1)-10]
	pathEnding := calcLocalPathEnding(data1)

	// 生成AES密钥
	aesKey := generateAESKey()

	// 加载公钥
	publicKey, err := rsaLoadPublicKey(publicKeyPEM)
	if err != nil {
		return nil, err
	}

	// 加密SDP和AES密钥
	bodyData := map[string]string{
		"data1": aesEncrypt(string(newSDP), aesKey),
		"data2": rsaEncrypt(aesKey, publicKey),
	}

	bodyJSON, err := json.Marshal(bodyData)
	if err != nil {
		return nil, err
	}

	// 第二个请求的URL
	url2 := fmt.Sprintf("http://%s:9991/con_ing_%s", ip, pathEnding)

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	// 使用字符串形式的body，与Python版本一致
	resp, err = makeLocalRequest(url2, strings.NewReader(string(bodyJSON)), headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("加密请求失败，状态码: %d", resp.StatusCode)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解密响应
	decryptedResponse := aesDecrypt(string(body), aesKey)

	var peerAnswer map[string]interface{}
	if err := json.Unmarshal([]byte(decryptedResponse), &peerAnswer); err != nil {
		return nil, err
	}

	log.Printf("getPeerAnswer II url2: %s", url2)
	log.Printf("getPeerAnswer II headers: %s", headers)
	log.Printf("getPeerAnswer II resp.body: %s", string(decryptedResponse))
	log.Printf("getPeerAnswer II peerAnswer: %s", peerAnswer)

	return peerAnswer, nil
}

// ConnectRobot 连接到机器人
func (conn *Go2Connection) ConnectRobot() error {
	// 创建提议
	offer, err := conn.peerConnection.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("创建提议失败: %v", err)
	}

	// 设置本地描述
	err = conn.peerConnection.SetLocalDescription(offer)
	if err != nil {
		return fmt.Errorf("设置本地描述失败: %v", err)
	}

	sdp_offer := conn.peerConnection.LocalDescription()
	log.Printf("ConnectRobot I sdp_offer: %v", sdp_offer)

	// 获取对等方应答
	peerAnswer, err := conn.getPeerAnswer(sdp_offer, conn.ip, conn.token)
	if err != nil {
		return fmt.Errorf("获取对等方应答失败: %v", err)
	}

	// 设置远程描述
	sdp, ok := peerAnswer["sdp"].(string)
	if !ok {
		return fmt.Errorf("应答中缺少SDP")
	}

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	}

	err = conn.peerConnection.SetRemoteDescription(answer)
	if err != nil {
		return fmt.Errorf("设置远程描述失败: %v", err)
	}

	log.Println("成功连接到机器人")
	return nil
}

func generate_id() int {
	return int(
		time.Now().UnixMilli() % 2147483648,
	)
}

// {"type": "msg", "topic": "rt/api/sport/request", "data": {"header": {"identity": {"id": 1626023453, "api_id": 1005}}, "parameter": "1005"}}
// {"type": "msg", "topic": "rt/api/sport/request"," data": {"header": {"identity": {"api_id": 1004, "id": 1626306583}}, "parameter": "1004"}}
// SendCommand 发送机器人命令
func (conn *Go2Connection) SendCommand(command string, data interface{}) {
	if cmdID, exists := SportCmd[command]; exists {
		conn.publish("rt/api/sport/request", map[string]interface{}{
			"header":    map[string]interface{}{"identity": map[string]interface{}{"id": generate_id(), "api_id": cmdID}},
			"parameter": strconv.Itoa(cmdID),
		}, MessageType)
	} else {
		log.Printf("未知命令: %s", command)
	}
}

// startHeartbeat 启动心跳
func (conn *Go2Connection) startHeartbeat() {
	log.Println("启动心跳机制")
	// conn.sendHeartbeat()
}

// sendHeartbeat 发送心跳
func (conn *Go2Connection) sendHeartbeat() {
	if conn.dataChannel != nil && conn.dataChannel.ReadyState() == webrtc.DataChannelStateOpen {
		currentTime := time.Now()
		data := map[string]interface{}{
			"timeInStr": currentTime.Format("2006-01-02 15:04:05"),
			"timeInNum": int(currentTime.Unix()),
		}
		conn.publish("", data, HeartbeatType)
	}

	// 2秒后发送下一次心跳
	conn.heartbeatTimer = time.AfterFunc(2*time.Second, conn.sendHeartbeat)
}

// stopHeartbeat 停止心跳
func (conn *Go2Connection) stopHeartbeat() {
	if conn.heartbeatTimer != nil {
		conn.heartbeatTimer.Stop()
		conn.heartbeatTimer = nil
	}
}

// Close 关闭连接
func (conn *Go2Connection) Close() error {
	// 停止心跳
	conn.stopHeartbeat()

	if conn.peerConnection != nil {
		return conn.peerConnection.Close()
	}
	return nil
}

// 示例使用
func main() {
	// 创建连接
	conn := NewGo2Connection(
		"192.168.123.161", // 机器人IP
		"",                // 机器人令牌
		func() {
			log.Println("验证成功")
		},
		func(message interface{}, msgObj interface{}) {
			// log.Printf("收到消息: %v", message)
		},
		func() {
			log.Println("连接已打开")
		},
	)

	// 连接到机器人
	err := conn.ConnectRobot()
	if err != nil {
		log.Fatal("连接失败:", err)
	}

	// 等待连接建立
	time.Sleep(2 * time.Second)

	// 发送命令示例
	// conn.SendCommand("Hello", nil)
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Second)
		conn.SendCommand("StandUp", nil)
		time.Sleep(10 * time.Second)
		conn.SendCommand("StandDown", nil)
	}

	// 保持连接一段时间
	time.Sleep(1000 * time.Second)

	// 关闭连接
	conn.Close()
}
