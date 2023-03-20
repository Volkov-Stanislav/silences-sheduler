package service

import (
	"fmt"

	"github.com/Volkov-Stanislav/silences-sheduler/models"
	"go.uber.org/zap"
)

type Runner struct {
	addShed chan models.SheduleSection
	delShed chan string
	stop    chan bool
	sheds   map[string]*models.SheduleSection
	apiURL  string
	logger  *zap.Logger
}

func NewRunner(apiURL string, logger *zap.Logger) (*Runner, error) {
	var o Runner
	o.addShed = make(chan models.SheduleSection)
	o.delShed = make(chan string)
	o.sheds = make(map[string]*models.SheduleSection)
	o.apiURL = apiURL
	o.logger = logger

	return &o, nil
}

func (o *Runner) Start() {
	go o.run()
}

func (o *Runner) Stop() {
}

func (o *Runner) GetChannels() (add chan models.SheduleSection, del chan string) {
	return o.addShed, o.delShed
}

func (o *Runner) run() {
	// цикл чтения из каналов....
	for {
		select {
		case <-o.stop:
			return
		case shed := <-o.addShed:
			token := shed.GetToken()
			o.logger.Info(fmt.Sprintf("Start shedules %v \n with token %v \n", shed, shed.GetToken()))
			o.sheds[token] = &shed
			o.sheds[token].Run(o.apiURL, o.logger)
		case token := <-o.delShed:
			if _, ok := o.sheds[token]; ok {
				o.logger.Info(fmt.Sprintf("Stop shedules %v \n with token %v \n", o.sheds[token], token))
				o.sheds[token].Stop()
				delete(o.sheds, token)
			}
		}
	}
}
