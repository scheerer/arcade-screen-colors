package lifx

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/pdf/golifx"
	"github.com/pdf/golifx/common"
	"github.com/pdf/golifx/protocol"
	"github.com/scheerer/arcade-screen-colors/internal/lights"
	"github.com/scheerer/arcade-screen-colors/internal/logging"
	"github.com/scheerer/arcade-screen-colors/internal/util"
	"go.uber.org/zap"
)

var logger = logging.New("lifx")

type LifxLights struct {
	config Config
	client *golifx.Client

	lightsMu sync.RWMutex
	group    common.Group
}

type Config struct {
	GroupName     string
	MaxBrightness float64
	MinBrightness float64
}

func NewLifx(ctx context.Context, config Config) (*LifxLights, error) {
	client, err := golifx.NewClient(&protocol.V2{})
	if err != nil {
		return nil, err
	}

	l := &LifxLights{
		config: config,
		client: client,
	}
	go l.Start(ctx)
	return l, nil
}

func (l *LifxLights) Start(ctx context.Context) {
	discoveryInterval := 15 * time.Second
	ticker := time.NewTicker(discoveryInterval)
	defer ticker.Stop()

	l.client.SetDiscoveryInterval(discoveryInterval)

	timeout := 5 * time.Second
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	l.discover(ctxWithTimeout)
	cancel()

	for {
		select {
		case <-ticker.C:
			ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
			l.discover(ctxWithTimeout)
			cancel()
		case <-ctx.Done():
			return
		}
	}

}

func (l *LifxLights) discover(ctx context.Context) {
	logger.With(zap.String("group", l.config.GroupName)).Info("LIFX discovery starting...")

	completed := make(chan error)

	var g common.Group
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		g, err = l.client.GetGroupByLabel(l.config.GroupName)
		if err != nil {
			logger.With(zap.Error(err)).Warn("Failed to get LIFX group by label")
		}
		completed <- err
	}()

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	select {
	case <-ctxWithTimeout.Done():
		logger.With(zap.Error(ctxWithTimeout.Err())).Warn("LIFX discovery timed out.")
		// l.resetClient()
	case <-completed:
		if g != nil {
			logger.With(zap.String("group", g.GetLabel())).Info("LIFX group found")
			l.lightsMu.Lock()
			l.group = g
			l.lightsMu.Unlock()
		} else {
			logger.With(zap.Error(ctxWithTimeout.Err())).Warn("Couldn't discover group.")
			// l.resetClient()
		}
	}

	logger.Info("LIFX discovery complete")
}

func (l *LifxLights) LightCount() int {
	if l.group == nil {
		return 0
	}

	l.lightsMu.RLock()
	count := 0
	for range l.group.Lights() {
		count++
	}
	l.lightsMu.RUnlock()
	return count
}

func (l *LifxLights) SetColorWithDuration(ctx context.Context, color lights.Color, duration time.Duration) {
	lifxColor := newLifxColor(color)
	lifxColor = adjustColor(lifxColor, l.config)

	logger.With(zap.Any("color", color),
		zap.Any("lifxColor", lifxColor)).
		Info("Setting LIFX device color")

	err := l.group.SetColor(lifxColor, duration)
	if err != nil {
		logger.With(zap.Error(err)).Warn("Failed to set color for LIFX group")
	}
}

func newLifxColor(color lights.Color) common.Color {
	// Convert RGB to HSB using uint16
	hue, saturation, brightness := util.RgbToHsb(color.Red, color.Green, color.Blue)

	return common.Color{
		Hue:        hue,
		Saturation: saturation,
		Brightness: brightness,
		Kelvin:     3500,
	}
}

func adjustColor(color common.Color, config Config) common.Color {
	blackThreshold := 0.015 * 0xFFFF
	if color.Brightness <= uint16(blackThreshold) && color.Saturation <= uint16(blackThreshold) {
		// blackish color - turn off the light
		return common.Color{
			Hue:        0,
			Saturation: 0,
			Brightness: 0,
			Kelvin:     3500,
		}
	}

	color.Brightness = uint16(math.Min(config.MaxBrightness*0xFFFF, math.Max(config.MinBrightness*0xFFFF, float64(color.Brightness))))

	return color
}
