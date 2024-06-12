package lights

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/pdf/golifx"
	"github.com/pdf/golifx/common"
	"github.com/pdf/golifx/protocol"
	"github.com/scheerer/arcade-screen-colors/internal/util"
	"go.uber.org/zap"
)

type LifxLights struct {
	groupName string
	client    *golifx.Client

	lightsMu sync.RWMutex
	group    common.Group
}

func NewLifx(ctx context.Context, groupName string) (*LifxLights, error) {
	client, err := golifx.NewClient(&protocol.V2{})
	if err != nil {
		return nil, err
	}

	l := &LifxLights{
		groupName: groupName,
		client:    client,
	}
	go l.Start(ctx)
	return l, nil
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

	g, err := l.client.GetGroupByLabel(l.groupName)
	if err != nil {
		return
	}

	l.lightsMu.Lock()
	l.group = g
	l.lightsMu.Unlock()
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

func (l *LifxLights) SetColorWithDuration(ctx context.Context, color Color, duration time.Duration) {
	lifxColor := newLifxColor(color)
	lifxColor = normalizeBrightness(lifxColor)

	logger.With(zap.Any("color", color),
		zap.Any("lifxColor", lifxColor)).
		Debug("Setting LIFX device color")

	err := l.group.SetColor(lifxColor, duration)
	if err != nil {
		logger.With(zap.Error(err)).Warn("Failed to set color for LIFX group")
	}
}

func newLifxColor(color Color) common.Color {
	// Convert RGB to HSB using uint16
	hue, saturation, brightness := util.RgbToHsb(color.Red, color.Green, color.Blue)

	return common.Color{
		Hue:        hue,
		Saturation: saturation,
		Brightness: brightness,
		Kelvin:     3500,
	}
}

// Keeps the brightness between 40% and 75% for colors with enough saturation to designate as a color
func normalizeBrightness(color common.Color) common.Color {
	// Normalize brightness to 0-65535
	seventyFivePercent := 0.75 * 65535
	fortyPercent := 0.4 * 65535

	if util.IsColorGreyish(color.Saturation) {
		color.Brightness = uint16(math.Min(seventyFivePercent, float64(color.Brightness)))
	} else {
		color.Brightness = uint16(math.Min(seventyFivePercent, math.Max(fortyPercent, float64(color.Brightness))))
	}

	return color
}
