# GPIO控制器 - 高低电平控制版

## 概述

这是一个精简的GPIO控制器，使用`periph.io/x`库，专注于GPIO引脚的高低电平控制，去除了蜂鸣器相关功能，提供了更通用的GPIO控制能力。

## 主要功能

### 1. 基础电平控制
```go
// 设置高电平
err := controller.SetHigh()

// 设置低电平
err := controller.SetLow()

// 设置数值（0=低电平，1=高电平）
err := controller.SetValue(1)  // 高电平
err := controller.SetValue(0)  // 低电平
```

### 2. 高级控制功能
```go
// 切换引脚状态
err := controller.Toggle()

// 产生脉冲（高电平-低电平）
err := controller.Pulse(100 * time.Millisecond)

// 闪烁模式（可控制次数）
err := controller.Blink(200*time.Millisecond, 5)  // 闪烁5次，每次间隔200ms
```

### 3. 状态查询
```go
// 获取当前电平
level, err := controller.GetLevel()

// 获取完整状态
status := controller.GetStatus()
// 返回:
// {
//   "pin_num": 18,
//   "exported": true,
//   "platform": "linux",
//   "library": "periph.io/x",
//   "level": "Low",
//   "value": 0
// }
```

## API接口

### 基础操作
```go
// 创建控制器
controller := NewGPIOController(pinNum)

// 设置高电平
err := controller.SetHigh()

// 设置低电平
err := controller.SetLow()

// 设置数值
err := controller.SetValue(value)  // value: 0 或 1

// 切换状态
err := controller.Toggle()

// 产生脉冲
err := controller.Pulse(duration)

// 闪烁模式
err := controller.Blink(interval, count)

// 获取电平
level, err := controller.GetLevel()

// 清理资源
err := controller.Cleanup()

// 获取状态
status := controller.GetStatus()
```

## 使用示例

### 基础控制
```go
package main

import (
    "time"
    "remote-ctrl-robot/cmd/client/robot"
)

func main() {
    // 创建GPIO控制器
    controller := robot.NewGPIOController(18)
    defer controller.Cleanup()

    // 基础高低电平控制
    controller.SetHigh()
    time.Sleep(1 * time.Second)
    controller.SetLow()
    time.Sleep(1 * time.Second)

    // 数值控制
    controller.SetValue(1)  // 高电平
    time.Sleep(500 * time.Millisecond)
    controller.SetValue(0)  // 低电平
}
```

### 高级控制
```go
// 脉冲控制
controller.Pulse(100 * time.Millisecond)

// 闪烁控制
controller.Blink(200*time.Millisecond, 3)  // 闪烁3次

// 切换控制
controller.Toggle()
time.Sleep(100 * time.Millisecond)
controller.Toggle()
```

### 状态监控
```go
// 获取当前状态
status := controller.GetStatus()
fmt.Printf("GPIO状态: %+v\n", status)

// 获取当前电平
if level, err := controller.GetLevel(); err == nil {
    fmt.Printf("当前电平: %v\n", level)
}
```

## 优势

1. **通用性强**: 适用于各种GPIO控制场景
2. **功能丰富**: 支持脉冲、闪烁等高级功能
3. **错误处理**: 完善的错误处理机制
4. **并发安全**: 使用`sync.RWMutex`保护共享状态
5. **跨平台支持**: `periph.io/x`提供更好的跨平台兼容性

## 测试

运行测试：
```bash
cd cmd/client/robot
go test -v
```

测试包括：
- 控制器创建
- 状态获取
- 高低电平设置
- 数值设置
- 切换功能
- 脉冲功能
- 闪烁功能
- 电平读取
- 资源清理
- 并发访问

## 注意事项

1. **平台限制**: 在非Linux系统上，GPIO操作会失败，这是正常的
2. **权限要求**: 在Linux系统上需要适当的GPIO访问权限
3. **引脚编号**: 使用BCM引脚编号（如GPIO18）
4. **输出引脚**: 当前实现仅支持输出引脚，不支持输入读取

## 依赖

```go
require (
    periph.io/x/conn/v3 v3.7.0
    periph.io/x/host/v3 v3.8.2
    github.com/rs/zerolog v1.31.0
)
```

## 变更历史

### v2.0.0
- ✅ 去除蜂鸣器相关功能
- ✅ 新增高低电平控制
- ✅ 新增脉冲功能
- ✅ 新增闪烁功能
- ✅ 新增切换功能
- ✅ 新增数值设置功能
- ✅ 优化状态查询功能 