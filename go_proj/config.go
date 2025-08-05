package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pion/webrtc/v3"
)

// RunMode 运行模式
type RunMode string

const (
	RunModeMock RunMode = "mock" // 模拟模式
	RunModeReal RunMode = "real" // 真实模式
)

// Config 配置结构
type Config struct {
	DefaultRobotIP string `json:"default_robot_ip"`
	DefaultToken   string `json:"default_token"`
	DefaultUCode   string `json:"default_ucode"`

	// 运行模式
	RunMode RunMode `json:"run_mode"`

	// WebRTC配置
	WebRTC struct {
		// ICE服务器配置
		ICEServers []string `json:"ice_servers"`

		// 视频编码配置
		VideoCodec string `json:"video_codec"`

		// 连接超时时间（秒）
		ConnectionTimeout int `json:"connection_timeout"`
	} `json:"webrtc"`

	// 服务器配置
	Server struct {
		// 监听端口
		Port string `json:"port"`

		// 静态文件目录
		StaticDir string `json:"static_dir"`
	} `json:"server"`

	// GPIO配置（仅real模式）
	GPIO struct {
		// 射击按钮GPIO引脚
		ShootPin int `json:"shoot_pin"`

		// 脉冲持续时间（毫秒）
		PulseDuration int `json:"pulse_duration"`
	} `json:"gpio"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		DefaultRobotIP: "192.168.123.161", // 默认机器人IP
		DefaultToken:   "",                // 默认Token
		DefaultUCode:   "Go2_001",         // 默认UCode
		RunMode:        RunModeReal,       // 默认真实模式
		WebRTC: struct {
			ICEServers        []string `json:"ice_servers"`
			VideoCodec        string   `json:"video_codec"`
			ConnectionTimeout int      `json:"connection_timeout"`
		}{
			ICEServers: []string{
				"stun:stun.l.google.com:19302",
				"stun:stun1.l.google.com:19302",
			},
			VideoCodec:        "H264",
			ConnectionTimeout: 30,
		},
		Server: struct {
			Port      string `json:"port"`
			StaticDir string `json:"static_dir"`
		}{
			Port:      "8080",
			StaticDir: "static",
		},
		GPIO: struct {
			ShootPin      int `json:"shoot_pin"`
			PulseDuration int `json:"pulse_duration"`
		}{
			ShootPin:      27,
			PulseDuration: 70,
		},
	}
}

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	// 如果配置文件不存在，创建默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultConfig()
		if err := config.Save(configPath); err != nil {
			return nil, fmt.Errorf("保存默认配置失败: %v", err)
		}
		log.Printf("📝 创建默认配置文件: %s", configPath)
		return config, nil
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %v", err)
	}

	log.Printf("✅ 配置加载成功: %s", configPath)
	log.Printf("🤖 默认机器人IP: %s", config.DefaultRobotIP)
	log.Printf("🎮 运行模式: %s", config.RunMode)

	return &config, nil
}

// Save 保存配置到文件
func (c *Config) Save(configPath string) error {
	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证运行模式
	if c.RunMode != RunModeMock && c.RunMode != RunModeReal {
		return fmt.Errorf("无效的运行模式: %s", c.RunMode)
	}

	// 验证默认机器人IP
	if c.DefaultRobotIP == "" {
		return fmt.Errorf("默认机器人IP不能为空")
	}

	// 验证端口
	if c.Server.Port == "" {
		return fmt.Errorf("服务器端口不能为空")
	}

	// 验证GPIO配置（仅real模式）
	if c.RunMode == RunModeReal {
		if c.GPIO.ShootPin < 0 {
			return fmt.Errorf("射击按钮GPIO引脚不能为负数")
		}
		if c.GPIO.PulseDuration <= 0 {
			return fmt.Errorf("脉冲持续时间必须大于0")
		}
	}

	return nil
}

// IsMockMode 检查是否为模拟模式
func (c *Config) IsMockMode() bool {
	return c.RunMode == RunModeMock
}

// IsRealMode 检查是否为真实模式
func (c *Config) IsRealMode() bool {
	return c.RunMode == RunModeReal
}

// GetDefaultUCode 获取默认UCode
func (c *Config) GetDefaultUCode(ucode string) string {
	if ucode != "" {
		return ucode
	}
	return c.DefaultUCode
}

// GetRobotIP 获取机器人IP，如果为空则使用默认IP
func (c *Config) GetRobotIP(robotIP string) string {
	if robotIP != "" {
		return robotIP
	}
	return c.DefaultRobotIP
}

// GetDefaultToken 获取默认Token
func (c *Config) GetDefaultToken(token string) string {
	if token != "" {
		return token
	}
	return c.DefaultToken
}

// GetICEServers 获取ICE服务器配置
func (c *Config) GetICEServers() []webrtc.ICEServer {
	var servers []webrtc.ICEServer

	for _, url := range c.WebRTC.ICEServers {
		servers = append(servers, webrtc.ICEServer{
			URLs: []string{url},
		})
	}

	// 如果没有配置ICE服务器，使用默认的
	if len(servers) == 0 {
		servers = []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		}
	}

	return servers
}
