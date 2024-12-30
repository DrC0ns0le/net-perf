package metrics

import (
	"flag"
	"net/http"
	"strconv"

	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricsPort = flag.Int("metrics.port", 5120, "port for metrics server")

	metricsPath = flag.String("metrics.path", "/metrics", "path for metrics server")
)

func Serve(global *system.Node) {
	http.Handle("/hello", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	http.Handle(*metricsPath, promhttp.Handler())
	global.Logger.With("listener", ":"+strconv.Itoa(*metricsPort)).Info("http server running")
	http.ListenAndServe(":"+strconv.Itoa(*metricsPort), nil)
}
