# Go2 WebRTC 代理系统

这是一个用于控制 Unitree Go2 机器人的 WebRTC 代理系统，它封装了复杂的机器人连接逻辑，提供了一个简单易用的 Web 界面来控制机器人和查看视频流。

## 🚀 功能特性

- **简化的连接流程**: 封装了复杂的 WebRTC 连接和验证逻辑
- **现代化 Web 界面**: 响应式设计，支持移动设备
- **实时视频流**: 通过 WebRTC 接收和显示机器人视频
- **丰富的控制命令**: 支持所有 Go2 机器人的运动命令
- **WebSocket 通信**: 实时双向通信，低延迟
- **连接状态管理**: 实时显示连接状态和错误信息
- **日志记录**: 详细的操作日志和调试信息

## 📋 系统架构

```
┌─────────────────┐    WebSocket    ┌─────────────────┐    WebRTC    ┌─────────────┐
│   Web 界面      │ ◄─────────────► │  Go 代理服务器  │ ◄──────────► │  Go2 机器人  │
│  (HTML/JS)      │                 │  (WebRTC Proxy) │              │             │
└─────────────────┘                 └─────────────────┘              └─────────────┘
```

## 🛠️ 安装和运行

### 前置要求

- Go 1.21 或更高版本
- 网络连接到 Go2 机器人

### 1. 克隆项目

```bash
git clone <repository-url>
cd go2-webrtc/go_proj
```

### 2. 安装依赖

```bash
make deps
```

### 3. 构建和运行

```bash
# 开发模式运行
make dev

# 或者分步执行
make build
./webrtc-proxy
```

### 4. 访问控制面板

打开浏览器访问: http://localhost:8080

## 🎮 使用说明

### 连接机器人

1. 在控制面板中输入机器人的 IP 地址（例如：192.168.1.100）
2. 输入访问令牌
3. 点击"连接机器人"按钮
4. 等待连接成功

### 控制机器人

#### 基本命令
- 使用下拉菜单选择命令，然后点击"发送命令"
- 或者使用快速命令按钮（如打招呼、舞蹈等）

#### 视频控制
- 点击"开启视频"开始接收视频流
- 点击"关闭视频"停止视频流

#### 支持的机器人命令

| 命令 | 描述 | 命令 | 描述 |
|------|------|------|------|
| Damp | 阻尼模式 | BalanceStand | 平衡站立 |
| StopMove | 停止移动 | StandUp | 站立 |
| StandDown | 蹲下 | RecoveryStand | 恢复站立 |
| Euler | 欧拉角 | Move | 移动 |
| Sit | 坐下 | RiseSit | 起立坐下 |
| SwitchGait | 切换步态 | Trigger | 触发 |
| BodyHeight | 身体高度 | FootRaiseHeight | 抬脚高度 |
| SpeedLevel | 速度等级 | Hello | 打招呼 |
| Stretch | 伸展 | TrajectoryFollow | 轨迹跟随 |
| ContinuousGait | 连续步态 | Content | 内容 |
| Wallow | 打滚 | Dance1 | 舞蹈1 |
| Dance2 | 舞蹈2 | FrontFlip | 前空翻 |
| FrontJump | 前跳 | FrontPounce | 前扑 |
| WiggleHips | 扭臀 | GetState | 获取状态 |
| EconomicGait | 经济步态 | FingerHeart | 手指爱心 |

## 🔧 API 接口

### HTTP API

#### 连接机器人
```http
POST /api/connect
Content-Type: application/json

{
  "robot_ip": "192.168.1.100",
  "token": "your_token"
}
```

#### 发送命令
```http
POST /api/command
Content-Type: application/json

{
  "robot_ip": "192.168.1.100",
  "token": "your_token",
  "command": "Hello"
}
```

#### 控制视频
```http
POST /api/video
Content-Type: application/json

{
  "robot_ip": "192.168.1.100",
  "token": "your_token",
  "command": "open"  // 或 "close"
}
```

### WebSocket API

连接 WebSocket: `ws://localhost:8080/ws`

#### 发送消息格式
```json
{
  "action": "connect|command|video",
  "robot_ip": "192.168.1.100",
  "token": "your_token",
  "command": "Hello"  // 可选
}
```

#### 接收消息格式
```json
{
  "success": true,
  "message": "连接成功",
  "data": {}  // 可选
}
```

## 🏗️ 项目结构

```
go_proj/
├── go2_webrtc.go      # 核心 WebRTC 连接逻辑
├── webrtc_proxy.go     # WebRTC 代理服务器
├── go.mod              # Go 模块文件
├── go.sum              # 依赖校验文件
├── Makefile            # 构建脚本
├── README.md           # 项目文档
└── static/             # 静态文件
    ├── index.html      # 主页面
    └── app.js          # 前端逻辑
```

## 🔍 调试和日志

### 查看日志
- 在 Web 界面的日志面板中查看实时日志
- 服务器端日志会输出到控制台

### 常见问题

1. **连接失败**
   - 检查机器人 IP 地址是否正确
   - 确认访问令牌有效
   - 检查网络连接

2. **视频无法显示**
   - 确保已开启视频流
   - 检查浏览器是否支持 WebRTC
   - 查看浏览器控制台错误信息

3. **命令无响应**
   - 确认机器人连接状态
   - 检查命令格式是否正确
   - 查看日志中的错误信息

## 🚀 开发

### 构建命令

```bash
# 安装依赖
make deps

# 构建项目
make build

# 运行测试
make test

# 代码格式化
make fmt

# 代码检查
make lint

# 清理构建文件
make clean
```

### 自定义配置

可以通过修改 `webrtc_proxy.go` 中的配置来调整服务器行为：

- 端口号（默认：8080）
- WebSocket 超时设置
- 日志级别
- 连接池大小

## 📄 许可证

本项目基于 MIT 许可证开源。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request 来改进这个项目！

## 📞 支持

如果您遇到问题或有建议，请：

1. 查看项目文档
2. 搜索已有的 Issue
3. 创建新的 Issue 并详细描述问题

---

**注意**: 使用此系统控制机器人时，请确保在安全的环境中进行，并遵循相关的安全操作规范。 