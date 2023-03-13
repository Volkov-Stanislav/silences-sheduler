package models

import (
	"fmt"

	"github.com/Volkov-Stanislav/cron"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

type SheduleSection struct {
	SectionName string    `` // Section name, for filestorage = filename
	Token       string    `` // Token for identifiend datachange. (modified date for files for filestorage)
	Shedules    []Shedule `` // Shedules in section.
	cron        *cron.Cron
}

func (o *SheduleSection) String() string {
	return fmt.Sprintf("SectionName: %v \nToken: %v \nShedules: \n%v\n", o.SectionName, o.Token, o.Shedules)
}

func (o *SheduleSection) Run(apiURL string, logger *zap.Logger) {
	if logger != nil {
		log := zapr.NewLogger(logger)
		o.cron = cron.New(cron.WithSeconds(), cron.WithLogger(log))
	} else {
		o.cron = cron.New(cron.WithSeconds())
	}

	for key := range o.Shedules {
		shed := o.Shedules[key]
		_, err := o.cron.AddFunc(o.Shedules[key].Cron, func() {
			shed.Run(apiURL, logger)
		})

		if err != nil {
			logger.Error(fmt.Sprintf("Error add Shedule: %v , err: %v", o.Shedules[key], err))
		}
	}

	o.cron.Start()
}

func (o *SheduleSection) Stop() {
	if o.cron != nil {
		o.cron.Stop()
	} else {
		fmt.Printf("Shedule %v does not heave cron!!!\n", o)
	}
}
