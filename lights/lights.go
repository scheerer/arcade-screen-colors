package lights

import (
	"context"
	"time"

	"github.com/scheerer/arcade-screen-colors/internal/logging"
)

var logger = logging.New("lights")

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
