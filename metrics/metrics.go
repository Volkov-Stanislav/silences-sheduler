package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics instance.
type Instance struct {
	metricsPort    string
	silencesSetted prometheus.Counter
	srv            *http.Server
}

// Return configured metrics instance.
func NewPrometheusInstance(metricsPort string) *Instance {
	var result Instance
	result.metricsPort = metricsPort
	result.register()
	result.srv = &http.Server{Addr: ":" + metricsPort}
	return &result
}

// Serve metrics instance on port from config.
func (o *Instance) Run() {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":"+o.metricsPort, nil)

		if err != nil {
			panic(err)
		}
	}()
}

// Stop and close metrics instance.
func (o *Instance) Stop() {
	o.srv.Close()
}

func (o *Instance) register() {
	// Register additional metrics.
	o.silencesSetted = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "silences_sheduler_silences_setted",
			Help: "How many silences setted since run.",
		},
	)

}

// Increase count runned silences.
func (o *Instance) AddSilencesCounter(count float64) {
	o.silencesSetted.Add(count)
}
