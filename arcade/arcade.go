package arcade

import (
	"context"
	"image"
	"image/color"
	"time"

	"github.com/kbinani/screenshot"
	"github.com/scheerer/arcade-screen-colors/internal/logging"
	"github.com/scheerer/arcade-screen-colors/internal/util"
	"github.com/scheerer/arcade-screen-colors/lights"
	"go.uber.org/zap"
)

var logger = logging.New("arcade")

type ScreenColorConfig struct {
	CaptureInterval time.Duration `env:"CAPTURE_INTERVAL" envDefault:"80ms"`
	ColorAlgo       string        `env:"COLOR_ALGO" envDefault:"AVERAGE"`
	LightType       string        `env:"LIGHT_TYPE" envDefault:"LIFX"`
	LightGroupName  string        `env:"LIGHT_GROUP_NAME" envDefault:"ARCADE"`
	MaxBrightness   float64       `env:"MAX_BRIGHTNESS" envDefault:"0.65"`
	MinBrightness   float64       `env:"MIN_BRIGHTNESS" envDefault:"0"`
	PixelGridSize   int           `env:"PIXEL_GRID_SIZE" envDefault:"5"`
	ScreenNumber    int           `env:"SCREEN_NUMBER" envDefault:"0"`
}

func RunScreenColors(ctx context.Context, config ScreenColorConfig, lightService lights.LightService) {

	var computeColor func(image *image.RGBA, pixelGridSize int) color.RGBA
	switch config.ColorAlgo {
	case "AVERAGE":
		computeColor = util.AverageColor
	case "SQUARED_AVERAGE":
		computeColor = util.SquaredAverageColor
	case "MEDIAN":
		computeColor = util.MedianColor
	case "MODE":
		computeColor = util.ModeColor
	default:
		logger.Fatalf("unknown color algorithm: %v", config.ColorAlgo)
	}

	var lastWarning time.Time
	for {
		select {
		case <-ctx.Done():
			break
		default:
			if lightService.LightCount() == 0 {
				time.Sleep(config.CaptureInterval)
				continue
			}

			startTime := time.Now()
			img, err := screenshot.CaptureDisplay(config.ScreenNumber)
			captureScreenDuration := time.Since(startTime)
			if err != nil {
				logger.With(zap.Error(err)).Error("Failed to capture screen")
				untilNextTick := config.CaptureInterval - captureScreenDuration
				if untilNextTick > 0 {
					time.Sleep(untilNextTick)
				}
				continue
			}

			colorCalculationStart := time.Now()
			c := computeColor(img, config.PixelGridSize)
			colorCalculationDuration := time.Since(colorCalculationStart)

			color := lights.Color{
				Red:   c.R,
				Green: c.G,
				Blue:  c.B,
			}

			if ctx.Err() != nil {
				// context is done - avoid setting color and break out of loop
				// this may have occurred while capturing the screen or calculating the color
				break
			}
			setColorStart := time.Now()
			lightService.SetColorWithDuration(ctx, color, 50*time.Millisecond)
			setColorDuration := time.Since(setColorStart)

			totalDuration := time.Since(startTime)
			if totalDuration > config.CaptureInterval {
				if time.Since(lastWarning) > 10*time.Second {
					logger.With(
						zap.Stringer("captureScreenDuration", captureScreenDuration),
						zap.Stringer("colorCalculationDuration", colorCalculationDuration),
						zap.Stringer("setColorDuration", setColorDuration),
						zap.Stringer("totalDuration", totalDuration)).
						Warn("Cannot keep up with CAPTURE_INTERVAL. Consider increasing PIXEL_GRID_SIZE or increasing CAPTURE_INTERVAL.")
					lastWarning = time.Now()
				}
			} else if totalDuration > 0 {
				untilNextTick := config.CaptureInterval - totalDuration
				time.Sleep(untilNextTick)
			}
		}
	}
}
