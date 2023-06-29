package stats

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

const statsCount = 500

// Instance struct for web statistic of runned service.
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

// NewInstance return configured Instance.
func NewInstance(port string, logger *zap.Logger) *Instance {
	var result Instance
	result.port = port
	result.logger = logger
	result.srv = &http.Server{Addr: ":" + port}
	result.RecvStat = make(chan bool)
	result.SetOK = make(chan bool)

	return &result
}

// Run begin cillecting stats.
func (o *Instance) Run() {
	go func() {
		err := o.serve()
		if err != nil {
			o.logger.Sugar().Info("Stats HTTP server exit, err: ", err)
		}
	}()
}

// Stop collecting stats.
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
	_, err := w.Write([]byte("Data Post Silence;Silence StartsAt;Silence EndsAt;Silence Comment;Silence Matchers\n"))
	if err != nil {
		o.logger.Sugar().Errorf("write in http.ResponseWriter failed: error %v", err)
		return
	}
	// first print old stats (after o.shedCountIndex)
	for _, stat := range o.stat[o.statsCountIndex:] {
		if stat == "" {
			break
		}

		_, err := w.Write([]byte(stat))
		if err != nil {
			o.logger.Sugar().Errorf("write in http.ResponseWriter failed: error %v", err)
			return
		}
	}

	// second print new stats (before o.shedCountIndex)
	if o.statsCountIndex == 0 { // all records already printed
		return
	}

	for _, stat := range o.stat[:o.statsCountIndex] {
		if stat == "" {
			continue
		}

		_, err := w.Write([]byte(stat))
		if err != nil {
			o.logger.Sugar().Errorf("write in http.ResponseWriter failed: error %v", err)
			return
		}
	}
}

// AddSheduleRun increase statistic of shedule run.
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
		_, err := w.Write([]byte(shed))
		if err != nil {
			o.logger.Sugar().Errorf("write in http.ResponseWriter failed: error %v", err)
			return
		}
	}
}

// SetShedules update info of runned shedules.
func (o *Instance) SetShedules(sheds []string) {
	o.sheds = sheds
}
