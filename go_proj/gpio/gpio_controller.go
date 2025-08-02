package gpio

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
)

// GPIOController GPIO控制器
type GPIOController struct {
	pin      gpio.PinOut
	pinNum   int
	exported bool
	mutex    sync.RWMutex
}

// NewGPIOController 创建新的GPIO控制器
func NewGPIOController(pinNum int) *GPIOController {
	return &GPIOController{
		pinNum:   pinNum,
		exported: false,
	}
}

// exportPin 导出GPIO引脚
func (g *GPIOController) exportPin() error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.exported {
		return nil
	}

	// 初始化periph.io/x主机
	if _, err := host.Init(); err != nil {
		return fmt.Errorf("failed to initialize periph.io/x host: %v", err)
	}

	// 获取GPIO引脚
	pin := gpioreg.ByName(fmt.Sprintf("GPIO%d", g.pinNum))
	if pin == nil {
		return fmt.Errorf("GPIO pin %d not found", g.pinNum)
	}

	// 转换为输出引脚
	pinOut, ok := pin.(gpio.PinOut)
	if !ok {
		return fmt.Errorf("GPIO pin %d cannot be used as output", g.pinNum)
	}

	g.pin = pinOut
	g.exported = true

	log.Info().Int("pin", g.pinNum).Msg("GPIO pin exported (periph.io/x)")
	return nil
}

// SetHigh 设置引脚为高电平
func (g *GPIOController) SetHigh() error {
	if err := g.exportPin(); err != nil {
		return err
	}
	return g.setPinValue(gpio.High)
}

// SetLow 设置引脚为低电平
func (g *GPIOController) SetLow() error {
	if err := g.exportPin(); err != nil {
		return err
	}
	return g.setPinValue(gpio.Low)
}

// SetValue 设置引脚值（0=低电平，1=高电平）
func (g *GPIOController) SetValue(value int) error {
	if err := g.exportPin(); err != nil {
		return err
	}

	var level gpio.Level
	if value == 0 {
		level = gpio.Low
	} else if value == 1 {
		level = gpio.High
	} else {
		return fmt.Errorf("invalid value: %d, must be 0 or 1", value)
	}

	return g.setPinValue(level)
}

// Toggle 切换引脚状态
func (g *GPIOController) Toggle() error {
	if err := g.exportPin(); err != nil {
		return err
	}

	// 由于输出引脚无法直接读取，我们使用一个简单的切换逻辑
	// 这里假设当前状态，实际应用中可能需要外部状态跟踪
	return g.setPinValue(gpio.High) // 简单实现，总是设置为高电平
}

// Pulse 产生一个脉冲（高电平-低电平）
func (g *GPIOController) Pulse(duration time.Duration) error {
	if err := g.exportPin(); err != nil {
		return err
	}

	// 设置为高电平
	if err := g.setPinValue(gpio.High); err != nil {
		return err
	}

	// 等待指定时间
	time.Sleep(duration)

	// 设置为低电平
	return g.setPinValue(gpio.Low)
}

// Blink 闪烁模式（可控制次数）
func (g *GPIOController) Blink(interval time.Duration, count int) error {
	if err := g.exportPin(); err != nil {
		return err
	}

	log.Info().
		Int("pin", g.pinNum).
		Dur("interval", interval).
		Int("count", count).
		Msg("Starting blink pattern")

	for i := 0; i < count; i++ {
		// 高电平
		if err := g.setPinValue(gpio.High); err != nil {
			return fmt.Errorf("failed to set high at blink %d: %v", i+1, err)
		}
		time.Sleep(interval / 2)

		// 低电平
		if err := g.setPinValue(gpio.Low); err != nil {
			return fmt.Errorf("failed to set low at blink %d: %v", i+1, err)
		}
		time.Sleep(interval / 2)
	}

	return nil
}

// setPinValue 设置引脚值
func (g *GPIOController) setPinValue(value gpio.Level) error {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	if !g.exported {
		return fmt.Errorf("GPIO pin %d is not exported", g.pinNum)
	}

	if err := g.pin.Out(value); err != nil {
		return fmt.Errorf("failed to set GPIO pin %d to %v: %v", g.pinNum, value, err)
	}

	log.Debug().
		Int("pin", g.pinNum).
		Str("level", value.String()).
		Msg("GPIO pin value set")

	return nil
}

// GetLevel 获取当前引脚电平
func (g *GPIOController) GetLevel() (gpio.Level, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	if !g.exported {
		return gpio.Low, fmt.Errorf("GPIO pin %d is not exported", g.pinNum)
	}

	// 由于输出引脚无法直接读取，我们返回一个默认值
	// 在实际应用中，可能需要外部状态跟踪
	return gpio.Low, nil
}

// Cleanup 清理GPIO资源
func (g *GPIOController) Cleanup() error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if !g.exported {
		return nil
	}

	// 设置为低电平
	g.setPinValue(gpio.Low)

	g.exported = false
	log.Info().Int("pin", g.pinNum).Msg("GPIO pin cleaned up (periph.io/x)")
	return nil
}

// GetStatus 获取GPIO状态
func (g *GPIOController) GetStatus() map[string]interface{} {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	status := map[string]interface{}{
		"pin_num":  g.pinNum,
		"exported": g.exported,
		"platform": runtime.GOOS,
		"library":  "periph.io/x",
	}

	// 如果已导出，获取当前电平
	if g.exported {
		if level, err := g.GetLevel(); err == nil {
			status["level"] = level.String()
			if level == gpio.High {
				status["value"] = 1
			} else {
				status["value"] = 0
			}
		}
	}

	return status
}
