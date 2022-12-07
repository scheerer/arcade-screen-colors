package main

import (
	"context"
	"net/http"
	"os"
	"screen_colors/util"
	"time"

	"github.com/urfave/cli/v2"

	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func init() {
	logger = util.NewLogger("main")
}

func main() {
	defer logger.Sync()
	app := cli.App{
		Name: "screen-colors",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "host",
				Usage:   "host to listen on",
				EnvVars: []string{"HOST"},
				Value:   "0.0.0.0",
			},
			&cli.IntFlag{
				Name:    "port",
				Usage:   "port to listen on",
				EnvVars: []string{"PORT"},
				Value:   3000,
			},
			&cli.DurationFlag{
				Name:    "capture-interval",
				Usage:   "how often to attempt to capture screen color",
				EnvVars: []string{"SCREEN_CAPTURE_INTERVAL"},
				Value:   100 * time.Millisecond,
			},
			&cli.StringFlag{
				Name:    "lifx-group",
				Usage:   "name of LIFX group to control lights (default ARCADE)",
				EnvVars: []string{"LIFX_GROUP"},
				Value:   "ARCADE",
			},
			&cli.StringFlag{
				Name:    "pprof-addr",
				Value:   "0.0.0.0:8080",
				EnvVars: []string{"PPROF_ADDR"},
			},
		},
		Before: func(c *cli.Context) error {
			logger.Info("Initializing screen color application...")

			if pprofAddr := c.String("pprof-addr"); pprofAddr != "" {
				go func() {
					logger.Fatal(http.ListenAndServe(pprofAddr, nil))
				}()
			}

			return nil
		},
		Action: func(c *cli.Context) error {
			return runScreenColors(c.Context,
				c.String("host"),
				c.Int("port"),
				c.Duration("capture-interval"),
				c.String("lifx-group"),
			)
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Fatalf("unable to run app: %s", err)
	}
}

func runScreenColors(ctx context.Context, host string, port int, captureInterval time.Duration, lifxGroup string) error {
	appDeps := newDependencies(ctx, Configuration{
		Host:            host,
		Port:            port,
		LifxGroupName:   lifxGroup,
		CaptureInterval: captureInterval,
		PixelDensity:    10,
	})

	go appDeps.screenColorService.Start(ctx)
	defer func() {
		appDeps.cleanup()
		logger.Info("Shutdown complete")
	}()

	appDeps.httpServer.Start()
	return nil
}
