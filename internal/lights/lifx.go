package lights

import (
	"context"
	"maps"
	"math"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.yhsif.com/lifxlan"
	"go.yhsif.com/lifxlan/light"
)

type LifxLights struct {
	groupName string

	lightsMu sync.RWMutex
	lights   map[string]*lifxLight
}

type lifxLight struct {
	device light.Device
	conn   net.Conn
}

func (l *lifxLight) Connected() bool {
	return l.conn != nil
}

func (l *lifxLight) Disconnect() {
	if l.conn != nil {
		l.conn.Close()
		l.conn = nil
	}
}

func newLifxLight(device light.Device) (*lifxLight, error) {
	conn, err := device.Dial()
	if err != nil {
		return nil, err
	}

	return &lifxLight{
		device: device,
		conn:   conn,
	}, nil
}

func NewLifx(ctx context.Context, groupName string) *LifxLights {
	l := &LifxLights{
		groupName: groupName,
		lights:    make(map[string]*lifxLight),
	}
	go l.Start(ctx)
	return l
}

func (l *LifxLights) Start(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	l.discover(ctxWithTimeout)
	cancel()

	for {
		select {
		case <-ticker.C:
			ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
			l.discover(ctxWithTimeout)
			cancel()
		case <-ctx.Done():
			return
		}
	}

}

func (l *LifxLights) discover(ctx context.Context) {
	logger.Info("LIFX discovery starting...")
	devices := make(chan lifxlan.Device)
	go func() {
		if err := lifxlan.Discover(ctx, devices, ""); err != nil {
			if err != context.DeadlineExceeded {
				logger.With(zap.Error(err)).Error("Failed to discover LIFX devices")
			}
		}
		logger.Info("LIFX discovery complete")
	}()

	l.lightsMu.RLock()
	existingDeviceNames := make(map[string]bool, len(l.lights))
	for name := range l.lights {
		existingDeviceNames[name] = true
	}
	l.lightsMu.RUnlock()

	newDeviceNames := make([]string, 0)

DISCOVER_LOOP:
	for {
		select {
		case device, ok := <-devices:
			if !ok {
				break DISCOVER_LOOP
			}

			newLight, err := light.Wrap(ctx, device, false)
			if err != nil {
				logger.With(zap.Any("device", device), zap.Error(err)).Warn("Failed to wrap LIFX device as Light")
				continue
			}
			deviceName := newLight.Label().String()
			logger.With(zap.String("deviceName", deviceName), zap.Any("light", newLight)).Info("Found LIFX light")

			newDeviceNames = append(newDeviceNames, deviceName)

			l.lightsMu.RLock()
			lifxLight, found := l.lights[deviceName]
			l.lightsMu.RUnlock()
			if found {
				err := lifxLight.device.Echo(ctx, lifxLight.conn, []byte("arcade-ping"))
				if err != nil {
					logger.With(zap.String("deviceName", deviceName), zap.Error(err)).Error("Could not ping LIFX light - removing")

					l.lightsMu.Lock()
					lifxLight.Disconnect()
					delete(l.lights, deviceName)
					l.lightsMu.Unlock()
					continue
				}
				logger.With(zap.String("deviceName", deviceName), zap.Any("light", newLight)).Info("Reusing existing known LIFX light")
			} else {
				logger.With(zap.String("deviceName", deviceName), zap.Any("light", newLight)).Info("Connecting to LIFX light")
				lifxLight, err = newLifxLight(newLight)
				if err != nil {
					logger.With(zap.String("deviceName", deviceName), zap.Any("light", newLight), zap.Error(err)).Error("Could not connect to LIFX light")
					continue
				}
			}

			l.lightsMu.Lock()
			l.lights[deviceName] = lifxLight
			l.lightsMu.Unlock()
		case <-ctx.Done():
			break DISCOVER_LOOP
		}
	}

	// remove new devices from existing device list
	for _, newDeviceName := range newDeviceNames {
		delete(existingDeviceNames, newDeviceName)
	}

	// remove devices that are no longer found
	for deviceName := range existingDeviceNames {
		l.lightsMu.Lock()
		if light, found := l.lights[deviceName]; found {
			logger.With(zap.String("deviceName", deviceName), zap.Any("light", light)).Info("Removing LIFX light not found during discovery")
			light.Disconnect()
			delete(l.lights, deviceName)
		}
		l.lightsMu.Unlock()
	}
}

func (l *LifxLights) LightCount() int {
	l.lightsMu.RLock()
	count := len(l.lights)
	l.lightsMu.RUnlock()
	return count
}

func (l *LifxLights) SetColorWithDuration(ctx context.Context, color Color, duration time.Duration) {
	l.lightsMu.RLock()
	lights := maps.Clone(l.lights)
	l.lightsMu.RUnlock()

	lifxColor := newLifxColor(color)
	normalizeBrightness(lifxColor)

	var wg sync.WaitGroup
	for name, light := range lights {
		if !light.Connected() {
			continue
		}
		wg.Add(1)
		go func(name string, light *lifxLight) {
			defer wg.Done()
			logger.With(zap.String("deviceName", name),
				zap.Any("color", color),
				zap.Any("lifxColor", lifxColor)).
				Debug("Setting LIFX device color")

			ctxWithTimeout, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
			err := light.device.SetColor(ctxWithTimeout, light.conn, lifxColor, duration, false)
			cancel()
			if err != nil {
				logger.With(zap.String("deviceName", name), zap.Error(err)).Error("Failed to set color for LIFX device - removing")
				light.Disconnect()
				l.lightsMu.Lock()
				delete(l.lights, name)
				l.lightsMu.Unlock()
			}
		}(name, light)
	}
	wg.Wait()
}

func newLifxColor(color Color) *lifxlan.Color {
	// Convert RGB to HSB using uint16
	hue, saturation, brightness := rgbToHsb(color.Red, color.Green, color.Blue)

	return &lifxlan.Color{
		Hue:        hue,
		Saturation: saturation,
		Brightness: brightness,
		Kelvin:     3500,
	}
}

// Keeps the brightness between 40% and 70%
func normalizeBrightness(color *lifxlan.Color) {
	// Normalize brightness to 0-65535
	seventyPercent := 0.7 * 65535
	fortyPercent := 0.4 * 65535

	color.Brightness = uint16(math.Min(seventyPercent, math.Max(fortyPercent, float64(color.Brightness))))
}
