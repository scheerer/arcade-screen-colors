package main

import (
	"context"
	"screen_colors/lifx"
	"screen_colors/screen"
	"screen_colors/web"
	"time"
)

type dependencies struct {
	lifxClient         *lifx.LifxClient
	screenColorService *screen.ColorsService
	httpServer         *web.Server
}

func newDependencies(ctx context.Context, config Configuration) *dependencies {
	lifxClient := lifx.NewLifxClient(ctx)

	screenColorService, err := screen.NewColorService(ctx, lifxClient, config.LifxGroupName, config.CaptureInterval, config.PixelDensity)
	if err != nil {
		logger.Fatalf("unable to create screen color screen: %s", err)
	}

	return &dependencies{
		lifxClient:         lifxClient,
		screenColorService: screenColorService,
		httpServer:         web.New(config.Host, config.Port, screenColorService),
	}
}

func (deps *dependencies) cleanup() {
	logger.Info("beginning clean up")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stopped := make(chan bool, 1)
	go func() {
		deps.screenColorService.Stop()
		deps.lifxClient.Stop()
		stopped <- true
	}()

	select {
	case <-stopped:
		logger.Info("finished cleaning up")
	case <-ctx.Done():
		logger.Warn("timeout waiting on cleanup")
	}
}
