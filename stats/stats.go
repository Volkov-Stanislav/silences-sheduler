package stats

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

const statsCount = 500

type Instance struct {
	RecvStat        chan bool
	SetOK           chan bool
	port            string
	stat            [statsCount]string
	statsCountIndex int
	sheds           []string
	logger          *zap.Logger
	srv             *http.Server
}

func NewInstance(port string, logger *zap.Logger) *Instance {
	var result Instance
	result.port = port
	result.logger = logger
	result.srv = &http.Server{Addr: ":" + port}
	result.RecvStat = make(chan bool)
	result.SetOK = make(chan bool)
	return &result
}

func (o *Instance) Run() {
	go func() {
		err := o.serve()
		if err != nil {
			o.logger.Sugar().Info("Stats HTTP server exit, err: ", err)
		}
	}()
}

func (o *Instance) Stop() {
	o.srv.Close()
}

func (o *Instance) serve() error {
	http.Handle("/stats", promhttp.InstrumentHandlerCounter(
		promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "silences_sheduler_requests_statistics_total",
				Help: "Total number of httpoint requests by HTTP code.",
			},
			[]string{"code"},
		),
		http.HandlerFunc(o.getStats),
	))

	http.Handle("/shedules", promhttp.InstrumentHandlerCounter(
		promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "silences_sheduler_requests_shedules_total",
				Help: "Total number of httpoint requests by HTTP code.",
			},
			[]string{"code"},
		),
		http.HandlerFunc(o.getShedules),
	))

	return o.srv.ListenAndServe()
}

func (o *Instance) getStats(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Data Post Silence;Silence StartsAt;Silence EndsAt;Silence Comment;Silence Matchers\n"))
	// first print old stats (after o.shedCountIndex)
	for _, stat := range o.stat[o.statsCountIndex:] {
		if stat == "" {
			break
		}
		w.Write([]byte(stat))
	}

	// second print new stats (before o.shedCountIndex)
	if o.statsCountIndex == 0 { // all records already printed
		return
	}

	for _, stat := range o.stat[:o.statsCountIndex] {
		if stat == "" {
			continue
		}
		w.Write([]byte(stat))
	}
}

func (o *Instance) AddSheduleRun(stat string) {
	o.stat[o.statsCountIndex] = stat

	o.statsCountIndex++
	if o.statsCountIndex == statsCount {
		o.statsCountIndex = 0
	}
}

func (o *Instance) getShedules(w http.ResponseWriter, r *http.Request) {
	o.RecvStat <- true
	<-o.SetOK

	for _, shed := range o.sheds {
		w.Write([]byte(shed))
	}
}

func (o *Instance) SetShedules(sheds []string) {
	o.sheds = sheds
}
