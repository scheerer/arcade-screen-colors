package web

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"screen_colors/screen"
	"screen_colors/util"
	"strconv"
	"syscall"
	"time"

	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func init() {
	logger = util.NewLogger("web")
}

type Server struct {
	*http.Server
	screenColorsService *screen.ColorsService
}

func New(host string, port int, screenColorService *screen.ColorsService) *Server {
	srv := &http.Server{
		Addr:         host + ":" + strconv.Itoa(port),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}
	return &Server{
		Server:              srv,
		screenColorsService: screenColorService,
	}
}

func (s *Server) Start() {
	http.HandleFunc("/screen-colors", s.handleScreenColors)
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("closed server error %s", err.Error())
		}
	}()
	logger.Infof("listening on %s, ctrl-c to exit...", s.Addr)
	s.shutdown()
}

func (s *Server) shutdown() {
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT)
	sig := <-quit
	log.Printf("server is shutting down %s", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.SetKeepAlivesEnabled(false)
	if err := s.Shutdown(ctx); err != nil {
		log.Printf("could not gracefully shutdown the server %s", err.Error())
	}
	log.Printf("server stopped")
}

func (s *Server) handleScreenColors(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// TODO: implement SSE to listen to events
		w.WriteHeader(http.StatusOK)
	case http.MethodPost:
		go s.screenColorsService.Start(context.Background())
		w.WriteHeader(http.StatusCreated)
	case http.MethodDelete:
		s.screenColorsService.Stop()
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
