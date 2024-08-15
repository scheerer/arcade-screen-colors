package main

import (
	"context"
	"image"
	"image/color"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/caarlos0/env"
	"github.com/kbinani/screenshot"
	"github.com/scheerer/arcade-screen-colors/internal/lights"
	"github.com/scheerer/arcade-screen-colors/internal/lights/lifx"
	"github.com/scheerer/arcade-screen-colors/internal/logging"
	"github.com/scheerer/arcade-screen-colors/internal/util"
)

var (
	logger = logging.New("main")
	config = ScreenColorConfig{}
)

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

func main() {
	defer logger.Sync()

	err := env.Parse(&config)
	if err != nil {
		logger.With(zap.Error(err)).Fatal("Failed to parse environment variables")
	}

	logger.With(zap.Any("config", config)).Info("Starting arcade lights")

	logger.Info("Adjust CAPTURE_INTERVAL to change how often the screen is captured. (should generally be greater than 50ms due to screen capture latency)")
	logger.Info("Adjust COLOR_ALGO to change color algorithm. Valid values are: [AVERAGE, SQUARED_AVERAGE, MEDIAN, MODE]")
	logger.Info("Adjust PIXEL_GRID_SIZE to increase performance or accuracy. Lower values are slower but more accurate. 1 being the most accurate.")
	logger.Info("LIGHT_TYPE only supports LIFX at the moment.")
	logger.Info("Adjust LIGHT_GROUP_NAME to change the group of lights to control.")
	logger.Info("Adjust MIN_BRIGHTNESS between 0 and 1.")
	logger.Info("Adjust MAX_BRIGHTNESS between 0 and 1.")
	logger.Info("Adjust SCREEN_NUMBER to target a different screen. 0 is the primary screen.")
	logger.Info("Press Ctrl+C to stop")

	ctx, cancel := context.WithCancel(context.Background())

	go Run(ctx, config)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown
	logger.Info("Shutting down")
	cancel()
}

func Run(ctx context.Context, config ScreenColorConfig) {
	var err error
	var lightService lights.LightService
	switch config.LightType {
	case "LIFX":
		lightService, err = lifx.NewLifx(ctx, lifx.Config{
			GroupName:     config.LightGroupName,
			MinBrightness: config.MinBrightness,
			MaxBrightness: config.MaxBrightness,
		})
		if err != nil {
			logger.With(zap.Error(err)).Fatal("Failed to create LIFX light service")
		}
	default:
		logger.Fatalf("unknown light type: %v", config.LightType)
	}

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
