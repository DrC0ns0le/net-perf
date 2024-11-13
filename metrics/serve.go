package metrics

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Serve(port string) {
	http.Handle("/hello", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Serving metrics on :" + port)
	http.ListenAndServe(":"+port, nil)
}
