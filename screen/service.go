package screen

import (
	"context"
	"image"
	"image/color"
	"math"
	"screen_colors/lifx"
	"screen_colors/types"
	"screen_colors/util"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/kbinani/screenshot"
	"go.uber.org/zap"
	"go.yhsif.com/lifxlan"

	"github.com/cskr/pubsub"
)

var logger *zap.SugaredLogger

func init() {
	logger = util.NewLogger("screen-colors-service")
}

const screenColorsTopic = "screenColors"

type ColorsService struct {
	lifxClient      *lifx.LifxClient
	lifxGroup       string
	captureInterval time.Duration
	pixelDensity    int

	events  *pubsub.PubSub
	running *atomic.Bool
}

func NewColorService(ctx context.Context, lifxClient *lifx.LifxClient, lifxGroupName string, captureInterval time.Duration, pixelDensity int) (*ColorsService, error) {
	service := &ColorsService{
		lifxClient:      lifxClient,
		lifxGroup:       lifxGroupName,
		captureInterval: captureInterval,
		pixelDensity:    pixelDensity,
		events:          pubsub.New(5),
		running:         new(atomic.Bool),
	}

	lightBroadcastCh := make(chan *types.ScreenColor, 1)
	go service.subscribeToColors(ctx, lightBroadcastCh)
	go service.handleNewColors(ctx, lightBroadcastCh)

	return service, nil
}

func (svc *ColorsService) Start(ctx context.Context) {
	logger.Info("starting screen color loop")
	svc.running.Store(true)
	// Check screens and bounds once at startup
	n := screenshot.NumActiveDisplays()
	bounds := screenshot.GetDisplayBounds(n - 1) // last display

	for {
		screenCaptureAttemptStart := time.Now()
		img, err := captureScreenWithBackoff(bounds)

		if err == nil {
			screenColor := &types.ScreenColor{
				Timestamp: time.Now(),
				Algorithms: map[string]color.Color{
					"squaredAvgRgb": squaredAvgColor(img, svc.pixelDensity),
				},
			}
			logger.With("screenColor", screenColor).Debug("broadcasting new color")
			svc.events.TryPub(screenColor, screenColorsTopic)
		}

		if !svc.running.Load() {
			return
		}

		sleepTime := svc.captureInterval - time.Since(screenCaptureAttemptStart)
		if sleepTime > 0 {
			time.Sleep(sleepTime)
		}
	}
}

func (svc *ColorsService) Stop() {
	logger.Info("stopping screen color loop...")
	svc.running.Swap(false)
}

func (svc *ColorsService) subscribeToColors(ctx context.Context, lightBroadcastCh chan *types.ScreenColor) {
	ch := svc.events.Sub(screenColorsTopic)
	for {
		select {
		case screenColor := <-ch:
			// drain color chan
			select {
			case <-lightBroadcastCh:
				logger.Info("draining message - possible backup")
			default:
				break
			}
			broadcastStart := time.Now()
			lightBroadcastCh <- screenColor.(*types.ScreenColor)
			util.PrintLatency(logger, "msgBroadcast", broadcastStart)
		case <-ctx.Done():
			logger.Info("shutting down color subscriber")
			return
		}
	}
}

func (svc *ColorsService) handleNewColors(ctx context.Context, lightBroadcastCh <-chan *types.ScreenColor) {
	for {
		select {
		case color := <-lightBroadcastCh:
			logger.Debug("sending color to lifxClient")
			screenColor := color.Algorithms["squaredAvgRgb"]
			lifxColor := lifxlan.FromColor(screenColor, 5000)

			cancelCtx, cancel := context.WithTimeout(ctx, svc.captureInterval)
			setColorStart := time.Now()
			svc.lifxClient.SetColor(cancelCtx, lifxColor, 5*time.Millisecond)
			cancel()
			util.PrintLatency(logger, "lifxClient.SetColor", setColorStart)
		case <-ctx.Done():
			logger.Info("shutting down broadcast handler")
			return
		}
	}
}

func captureScreenWithBackoff(bounds image.Rectangle) (*image.RGBA, error) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 5 * time.Second
	bo.MaxElapsedTime = 0 // never stop
	var img *image.RGBA
	var err error
	if err := backoff.RetryNotify(func() error {
		captureDisplayStart := time.Now()
		img, err = screenshot.CaptureRect(bounds)
		util.PrintLatency(logger, "captureDisplay", captureDisplayStart)
		return err
	}, bo, func(err error, t time.Duration) {
		logger.Warn("could not capture screen.. backing off: ", err)
	}); err != nil {
		logger.Errorf("total failure capturing screen")
	}
	return img, err
}

func squaredAvgColor(img image.Image, pixelDensity int) color.Color {
	startTime := time.Now()
	defer util.PrintLatency(logger, "squaredAvgColor", startTime)
	sumr := 0.0
	sumg := 0.0
	sumb := 0.0
	pixelsScanned := 0.0

	for x := 0; x < img.Bounds().Dx(); x += pixelDensity {
		for y := 0; y < img.Bounds().Dy(); y += pixelDensity {
			r, g, b, _ := img.At(x, y).RGBA()

			fr := float64(r) / 65535 * 255
			fg := float64(g) / 65535 * 255
			fb := float64(b) / 65535 * 255

			sumr += fr * fr
			sumg += fg * fg
			sumb += fb * fb

			pixelsScanned++
		}
	}
	return color.RGBA{
		R: uint8(math.Sqrt(sumr / pixelsScanned)),
		G: uint8(math.Sqrt(sumg / pixelsScanned)),
		B: uint8(math.Sqrt(sumb / pixelsScanned)),
		A: 0,
	}
}
