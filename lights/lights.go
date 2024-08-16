package lights

import (
	"context"
	"time"

	"github.com/scheerer/arcade-screen-colors/internal/logging"
)

var logger = logging.New("lights")

var ColorWhite = Color{Red: 255, Green: 255, Blue: 255}
var ColorBlack = Color{Red: 0, Green: 0, Blue: 0}

type Color struct {
	Red   uint8
	Green uint8
	Blue  uint8
}

type LightService interface {
	Start(ctx context.Context)
	Stop()
	LightCount() int
	SetColorWithDuration(ctx context.Context, color Color, duration time.Duration)
}
