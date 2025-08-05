# 配置说明

## 概述

`config.go` 提供了灵活的配置管理系统，支持设置默认机器人IP和运行模式（mock/real）。

## 配置文件

程序会在首次运行时自动创建 `config.json` 配置文件。您也可以手动创建或修改此文件。

### 配置选项

#### 基本配置

- `default_robot_ip`: 默认机器人IP地址
- `run_mode`: 运行模式
  - `"mock"`: 模拟模式（不执行真实的GPIO操作）
  - `"real"`: 真实模式（执行真实的GPIO操作）

#### WebRTC配置

- `webrtc.ice_servers`: ICE服务器列表
- `webrtc.video_codec`: 视频编码格式（默认：H264）
- `webrtc.connection_timeout`: 连接超时时间（秒）

#### 服务器配置

- `server.port`: 服务器监听端口
- `server.static_dir`: 静态文件目录

#### GPIO配置（仅real模式）

- `gpio.shoot_pin`: 射击按钮GPIO引脚
- `gpio.pulse_duration`: 脉冲持续时间（毫秒）

## 使用示例

### 1. 模拟模式配置

```json
{
  "default_robot_ip": "192.168.1.100",
  "run_mode": "mock",
  "webrtc": {
    "ice_servers": [
      "stun:stun.l.google.com:19302"
    ],
    "video_codec": "H264",
    "connection_timeout": 30
  },
  "server": {
    "port": "8080",
    "static_dir": "static"
  },
  "gpio": {
    "shoot_pin": 27,
    "pulse_duration": 70
  }
}
```

### 2. 真实模式配置

```json
{
  "default_robot_ip": "192.168.1.100",
  "run_mode": "real",
  "webrtc": {
    "ice_servers": [
      "stun:stun.l.google.com:19302",
      "stun:stun1.l.google.com:19302"
    ],
    "video_codec": "H264",
    "connection_timeout": 30
  },
  "server": {
    "port": "8080",
    "static_dir": "static"
  },
  "gpio": {
    "shoot_pin": 27,
    "pulse_duration": 70
  }
}
```

## 运行模式说明

### Mock模式
- 适用于开发和测试环境
- 不会执行真实的GPIO操作
- 射击命令只会记录日志，不会触发硬件

### Real模式
- 适用于生产环境
- 会执行真实的GPIO操作
- 射击命令会触发硬件脉冲

## 配置验证

程序会自动验证配置的有效性：

1. 运行模式必须是 `mock` 或 `real`
2. 默认机器人IP不能为空
3. 服务器端口不能为空
4. Real模式下GPIO配置必须有效

## 启动程序

```bash
cd go_proj
go run .
```

程序会自动：
1. 检查 `config.json` 是否存在
2. 如果不存在，创建默认配置文件
3. 加载并验证配置
4. 根据配置启动服务器

## 配置热重载

目前配置在程序启动时加载，不支持运行时修改。如需修改配置，请重启程序。 