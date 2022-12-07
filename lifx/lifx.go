package lifx

import (
	"context"
	"net"
	"screen_colors/util"
	"sync"
	"time"

	"go.yhsif.com/lifxlan"
	"go.yhsif.com/lifxlan/light"

	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

const DiscoveryTimeout = 5 * time.Second

func init() {
	logger = util.NewLogger("lifx")
}

type LifxClient struct {
	devicesMu sync.RWMutex
	devices   map[string]*Light
}

func NewLifxClient(ctx context.Context) *LifxClient {
	c := &LifxClient{
		devices: make(map[string]*Light),
	}
	go c.discoverLoop(ctx)
	return c
}

func (lc *LifxClient) Stop() {
	lc.devicesMu.Lock()
	for _, l := range lc.devices {
		_ = l.Conn.Close()
	}
	lc.devicesMu.Unlock()
}

func (lc *LifxClient) Reset(ctx context.Context) {
	lc.SetColor(ctx, DefaultColor(), 10*time.Millisecond)
}

func DefaultColor() *lifxlan.Color {
	return &lifxlan.Color{
		Hue:        0,
		Saturation: 0,
		Brightness: 32767,
		Kelvin:     5000,
	}
}

func (c *LifxClient) discoverLoop(ctx context.Context) {
	c.discoverDevices(ctx, DiscoveryTimeout)
	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ticker.C:
			c.discoverDevices(ctx, DiscoveryTimeout)
		case <-ctx.Done():
			return
		}
	}
}

func (c *LifxClient) discoverDevices(ctx context.Context, discoverTimeout time.Duration) {
	logger.Infof("starting discoverDevices with discoverTimeout %dms", discoverTimeout.Milliseconds())
	var discoverContext context.Context
	var cancel context.CancelFunc
	if discoverTimeout > 0 {
		discoverContext, cancel = context.WithTimeout(ctx, discoverTimeout)
	} else {
		discoverContext, cancel = context.WithCancel(ctx)
	}
	defer cancel()
	deviceChan := make(chan lifxlan.Device)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := lifxlan.Discover(discoverContext, deviceChan, ""); err != nil {
			if util.CheckContextError(err) {
				logger.Error("Discover failed: ", err)
			}
		}
	}()

	for device := range deviceChan {
		wg.Add(1)
		go func(device lifxlan.Device) {
			defer wg.Done()
			logger.Infof("Found %v, checking light capablities...", device)
			lifxLight, err := light.Wrap(discoverContext, device, false)
			if err != nil || lifxLight == nil {
				logger.Error("could not wrap light", err)
				return
			}
			label := lifxLight.Label()
			logger.Infof("discovered light: %s", lifxLight.Label())

			c.devicesMu.RLock()
			l, found := c.devices[label.String()]
			c.devicesMu.RUnlock()

			if found {
				logger.With(zap.String("label", label.String())).Info("light already known - checking heartbeat")
				err := l.Light.Echo(discoverContext, l.Conn, []byte("heartbeat"))
				if err == nil {
					logger.With(zap.String("label", label.String())).Info("heartbeat success - skipping light")
					return
				}

				logger.With(zap.String("label", label.String())).Error("heartbeat failed - removing")
				c.devicesMu.Lock()
				delete(c.devices, label.String())
				c.devicesMu.Unlock()
			}

			l, err = newLight(lifxLight)
			if err != nil {
				logger.Error("could not create new light", err)
				return
			}
			c.devicesMu.Lock()
			c.devices[label.String()] = l
			c.devicesMu.Unlock()
		}(device)
	}
	logger.Info("waiting for discovery to complete")
	wg.Wait()
	c.devicesMu.RLock()
	count := len(c.devices)
	c.devicesMu.RUnlock()
	logger.Infof("discovery complete - found %d lights", count)
}

func (c *LifxClient) SetColor(ctx context.Context, color *lifxlan.Color, duration time.Duration) {
	c.devicesMu.RLock()
	devices := c.devices
	c.devicesMu.RUnlock()

	var wg sync.WaitGroup
	for label, light := range devices {
		wg.Add(1)
		go func(label string, light *Light) {
			defer wg.Done()
			startTime := time.Now()

			done := make(chan bool)
			go func() {
				err := light.SetColor(ctx, color, duration)
				if err != nil {
					logger.With(zap.String("label", label)).Debug("Could not send color to light")
				} else {
					logger.With(zap.String("label", label), zap.Any("lifxColor", color)).Debug("Color sent to light")
				}
				done <- true
			}()
			select {
			case <-done:
				util.PrintLatency(logger, "light.SetColor", startTime)
			case <-ctx.Done():
				logger.With(zap.String("label", label)).Debug("Light took too long - giving up")
			}
		}(label, light)
	}
	wg.Wait()
}

type Light struct {
	Light light.Device
	Conn  net.Conn
}

func newLight(light light.Device) (*Light, error) {
	logger.With(zap.String("label", light.Label().String())).Info("connecting to light...")
	conn, err := light.Dial()
	if err != nil {
		return nil, err
	}
	logger.With(zap.String("label", light.Label().String())).Info("connected!")
	return &Light{
		Light: light,
		Conn:  conn,
	}, nil
}

func (l *Light) SetColor(ctx context.Context, color *lifxlan.Color, duration time.Duration) error {
	return l.Light.SetColor(ctx, l.Conn, color, duration, true)
}
