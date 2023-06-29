package service

import (
	"fmt"
	"sync"

	"github.com/Volkov-Stanislav/silences-sheduler/metrics"
	"github.com/Volkov-Stanislav/silences-sheduler/models"
	"github.com/Volkov-Stanislav/silences-sheduler/stats"
	"go.uber.org/zap"
)

// Runner main service struct.
type Runner struct {
	addShed chan models.SheduleSection
	delShed chan string
	stop    chan bool
	sheds   map[string]*models.SheduleSection
	apiURL  string
	logger  *zap.Logger
	mux     sync.Mutex
	stat    *stats.Instance
	prom    *metrics.Instance
}

// NewRunner return configured Runner instance.
func NewRunner(apiURL string, logger *zap.Logger, stat *stats.Instance, prom *metrics.Instance) (*Runner, error) {
	var o Runner
	o.addShed = make(chan models.SheduleSection)
	o.delShed = make(chan string)
	o.stop = make(chan bool)
	o.sheds = make(map[string]*models.SheduleSection)
	o.apiURL = apiURL
	o.logger = logger
	o.stat = stat
	o.prom = prom

	return &o, nil
}

// Start runner.
func (o *Runner) Start() {
	go o.run()
}

// Stop Runner.
func (o *Runner) Stop() {
	fmt.Println("Runner (o *Instance) Stop()")
	o.stop <- true
}

// GetChannels return sync channels for  use on another gorutines for syncing.
func (o *Runner) GetChannels() (add chan models.SheduleSection, del chan string) {
	return o.addShed, o.delShed
}

func (o *Runner) run() {
	for {
		select {
		case <-o.stop:
			// stop cron shedules.
			for shedToken := range o.sheds {
				fmt.Printf("Stop Cron Token: %v", shedToken)
				o.sheds[shedToken].Stop()
			}

			return
		case <-o.stat.RecvStat:
			o.setShedulesForWeb()
			o.stat.SetOK <- true
		case shed := <-o.addShed:
			token := shed.GetToken()
			o.logger.Info(fmt.Sprintf("Start shedules %v \n with token %v \n", shed, shed.GetToken()))
			o.mux.Lock()
			o.sheds[token] = &shed
			o.sheds[token].Run(o.apiURL, o.logger, o.stat, o.prom)
			o.mux.Unlock()
		case token := <-o.delShed:
			if _, ok := o.sheds[token]; ok {
				o.logger.Info(fmt.Sprintf("Stop shedules %v \n with token %v \n", o.sheds[token], token))
				o.mux.Lock()
				o.sheds[token].Stop()
				delete(o.sheds, token)
				o.mux.Unlock()
			}
		}
	}
}

func (o *Runner) setShedulesForWeb() {
	var result []string

	o.mux.Lock()
	result = append(result, fmt.Sprintln("Cron;Comment;Matchers;NextTimeRun"))

	for _, shedInst := range o.sheds {
		result = append(result, shedInst.GetSectionForWeb()...)
	}
	o.mux.Unlock()
	o.stat.SetShedules(result)
}
