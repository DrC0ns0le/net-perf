package metrics

import (
	"flag"
	"net/http"
	"strconv"

	"github.com/DrC0ns0le/net-perf/pkg/logging"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricsPort = flag.Int("metrics.port", 5120, "port for metrics server")

	metricsPath = flag.String("metrics.path", "/metrics", "path for metrics server")
)

func Serve() {
	http.Handle("/hello", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	http.Handle(*metricsPath, promhttp.Handler())
	logging.Infof("Serving metrics on :%d", *metricsPort)
	http.ListenAndServe(":"+strconv.Itoa(*metricsPort), nil)
}
