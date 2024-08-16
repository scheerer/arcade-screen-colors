package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/caarlos0/env"
	"github.com/scheerer/arcade-screen-colors/arcade"
	"github.com/scheerer/arcade-screen-colors/internal/logging"
	"github.com/scheerer/arcade-screen-colors/lights/lifx"
)

var (
	logger = logging.New("main")
	config = arcade.ScreenColorConfig{}
)

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
	lightService := lifx.NewLifxFromScreenColorConfig(config)

	go lightService.Start(ctx)

	go arcade.RunScreenColors(ctx, config, lightService)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown
	logger.Info("Shutting down")
	cancel()
}
