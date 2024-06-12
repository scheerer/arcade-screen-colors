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

	"github.com/scheerer/arcade-screen-colors/internal/lights"
	"github.com/scheerer/arcade-screen-colors/internal/logging"
	"github.com/scheerer/arcade-screen-colors/internal/screen"
	"github.com/scheerer/arcade-screen-colors/internal/util"
)

var logger = logging.New("main")

func main() {
	defer logger.Sync()

	screenNumber := util.Getenv("SCREEN_NUMBER", 0)
	pixelGridSize := util.Getenv("PIXEL_GRID_SIZE", 5)
	captureInterval := util.Getenv("CAPTURE_INTERVAL", 100*time.Millisecond)
	colorAlgo := util.Getenv("COLOR_ALGO", "AVERAGE")
	lightType := util.Getenv("LIGHT_TYPE", "LIFX")
	lightGroupName := util.Getenv("LIGHT_GROUP_NAME", "ARCADE")

	logger.With(
		zap.Int("SCREEN_NUMBER", screenNumber),
		zap.Int("PIXEL_GRID_SIZE", pixelGridSize),
		zap.Stringer("CAPTURE_INTERVAL", captureInterval),
		zap.String("COLOR_ALGO", colorAlgo),
		zap.String("LIGHT_TYPE", lightType),
		zap.String("LIGHT_GROUP_NAME", lightGroupName)).
		Info("Starting arcade lights")
	logger.Info("Adjust SCREEN_NUMBER to target a different screen. 0 is the primary screen.")
	logger.Info("Adjust PIXEL_GRID_SIZE to increase performance or accuracy. Lower values are slower but more accurate. 1 being the most accurate.")
	logger.Info("Adjust COLOR_ALGO to change color algorithm. Valid values are: [AVERAGE, SQUARED_AVERAGE, MEDIAN, MODE]")
	logger.Info("LIGHT_TYPE only supports LIFX at the moment.")
	logger.Info("Adjust LIGHT_GROUP_NAME to change the group of lights to control.")
	logger.Info("Press Ctrl+C to stop")

	ctx, cancel := context.WithCancel(context.Background())

	var lightService lights.LightService
	switch lightType {
	case "LIFX":
		lightService = lights.NewLifx(ctx, lightGroupName)
	default:
		logger.Fatalf("unknown light type: %v", lightType)
	}

	var computeColor func(image *image.RGBA, pixelGridSize int) color.RGBA
	switch colorAlgo {
	case "AVERAGE":
		computeColor = screen.AverageColor
	case "SQUARED_AVERAGE":
		computeColor = screen.SquaredAverageColor
	case "MEDIAN":
		computeColor = screen.MedianColor
	case "MODE":
		computeColor = screen.ModeColor
	default:
		logger.Fatalf("unknown color algorithm: %v", colorAlgo)
	}

	go func() {
		var lastWarning time.Time
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if lightService.LightCount() == 0 {
					time.Sleep(captureInterval)
					continue
				}

				startTime := time.Now()
				img, err := screen.CaptureDisplay(screenNumber)
				// img, err := screenshot.CaptureDisplay(screenNumber)
				captureScreenDuration := time.Since(startTime)
				if err != nil {
					logger.With(zap.Error(err)).Error("Failed to capture screen")
					untilNextTick := captureInterval - captureScreenDuration
					if untilNextTick > 0 {
						time.Sleep(untilNextTick)
					}
					continue
				}

				colorCalculationStart := time.Now()
				c := computeColor(img, pixelGridSize)
				colorCalculationDuration := time.Since(colorCalculationStart)

				color := lights.Color{
					Red:   uint8(c.R),
					Green: uint8(c.G),
					Blue:  uint8(c.B),
				}

				setColorStart := time.Now()
				lightService.SetColorWithDuration(ctx, color, 50*time.Millisecond)
				setColorDuration := time.Since(setColorStart)

				totalDuration := time.Since(startTime)
				if totalDuration > captureInterval {
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
					untilNextTick := captureInterval - totalDuration
					time.Sleep(untilNextTick)
				}
			}
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown
	logger.Info("Shutting down")
	cancel()
}
