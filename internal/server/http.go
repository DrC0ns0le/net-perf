package server

import (
	"context"
	"flag"
	"net/http"
	"strconv"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"net/http/pprof"
)

var (
	metricsPort = flag.Int("http.port", 5120, "port for http server")
	metricsPath = flag.String("http.metrics.path", "/metrics", "path for metrics server")
)

type HTTPServer struct {
	listenAddress string
	server        *http.Server
	logger        logging.Logger

	wgUpdateCh chan netctl.WGInterface
	rtUpdateCh chan struct{}
}

func NewHTTPServer(global *system.Node) *HTTPServer {
	return &HTTPServer{
		listenAddress: ":" + strconv.Itoa(*metricsPort),
		logger:        global.Logger.With("component", "http"),
		wgUpdateCh:    global.WGUpdateCh,
		rtUpdateCh:    global.RTUpdateCh,
	}
}

func (s *HTTPServer) Start() error {
	mux := http.NewServeMux()
	mux.Handle("/hello", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	mux.Handle("GET /wg/{interface}", http.HandlerFunc(s.handleWGUpdate))
	mux.Handle("GET /rt", http.HandlerFunc(s.handleRTUpdate))

	mux.Handle(*metricsPath, promhttp.Handler())
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	s.server = &http.Server{
		Addr:    s.listenAddress,
		Handler: mux,
	}

	s.logger.With("listener", s.listenAddress).Info("http server running")
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Stop() error {
	s.logger.Info("stopping http server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// Handlers

func (s *HTTPServer) handleWGUpdate(w http.ResponseWriter, r *http.Request) {

	ifaceQuery := r.PathValue("interface")
	iface, err := netctl.ParseWGInterface(ifaceQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.logger.Infof("got wg update request from client %s for interface %s", r.RemoteAddr, iface.Name)

	select {
	case s.wgUpdateCh <- iface:
		w.Write([]byte("OK\n"))
	case <-time.After(5 * time.Second):
		w.Write([]byte("timeout\n"))
	}
}

func (s *HTTPServer) handleRTUpdate(w http.ResponseWriter, r *http.Request) {
	s.logger.Infof("got rt update request from client %s", r.RemoteAddr)
	select {
	case s.rtUpdateCh <- struct{}{}:
		w.Write([]byte("OK\n"))
	case <-time.After(5 * time.Second):
		w.Write([]byte("timeout\n"))
	}
}
