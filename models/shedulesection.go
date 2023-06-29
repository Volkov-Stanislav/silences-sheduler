package models

import (
	"fmt"

	"github.com/Volkov-Stanislav/cron"
	"github.com/Volkov-Stanislav/silences-sheduler/metrics"
	"github.com/Volkov-Stanislav/silences-sheduler/stats"
	"github.com/Volkov-Stanislav/silences-sheduler/utils"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

// SheduleSection set of Shedules from one config file and TimeOffset.
type SheduleSection struct {
	Shedules       []Shedule  `yaml:"shedules"`       // Shedules in section.
	TimeOffset     string     `yaml:"timeoffset"`     // Offset in hours from UTC. May be + and -. ""=local
	GlobalMatchers []Matchers `yaml:"globalmatchers"` // Matchers added in all silences in SeduleSection
	cron           *cron.Cron
	sectionName    string `` // Section name, for filestorage = filename
	token          string `` // Token for identifiend datachange. (modified date for files for filestorage)
}

// String interface.
func (o *SheduleSection) String() string {
	return fmt.Sprintf("%#v", o)
}

// GetSectionName return section name.
func (o *SheduleSection) GetSectionName() string {
	return o.sectionName
}

// SetSectionName set section name.
func (o *SheduleSection) SetSectionName(name string) {
	o.sectionName = name
}

// GetToken return token for section.
func (o *SheduleSection) GetToken() string {
	return o.token
}

// SetToken set token for section.
func (o *SheduleSection) SetToken(token string) {
	o.token = token
}

// Run begin executing shedules from section.
func (o *SheduleSection) Run(apiURL string, logger *zap.Logger, stat *stats.Instance, prom *metrics.Instance) {
	if logger != nil {
		log := zapr.NewLogger(logger)
		o.cron = cron.New(cron.WithSeconds(), cron.WithLogger(log), cron.WithLocation(utils.GetLocation(o.TimeOffset)))
	} else {
		o.cron = cron.New(cron.WithSeconds(), cron.WithLocation(utils.GetLocation(o.TimeOffset)))
	}

	for key := range o.Shedules {
		shed := o.Shedules[key]
		entryID, err := o.cron.AddFunc(o.Shedules[key].Cron, func() {
			shed.Run(apiURL, logger, stat, prom)
		})

		o.Shedules[key].SetEntryID(entryID)

		if err != nil {
			logger.Error(fmt.Sprintf("Error add Shedule: %v , err: %v", o.Shedules[key], err))
		}
	}

	o.cron.Start()
}

// Stop executing shedules from section.
func (o *SheduleSection) Stop() {
	fmt.Println("(o *SheduleSection) Stop()")

	if o.cron != nil {
		o.cron.Stop()
	}
}

// GetSectionForWeb return formatted text of runned section for web report.
func (o *SheduleSection) GetSectionForWeb() []string {
	var (
		result []string
		res    string
	)

	for shed := range o.Shedules {
		res = fmt.Sprintf("%v;%v;%v;%v\n",
			o.Shedules[shed].Cron,
			o.Shedules[shed].Silence.Comment,
			o.Shedules[shed].Silence.Matchers,
			o.cron.Entry(o.Shedules[shed].GetEntryID()).Next)
		result = append(result, res)
	}

	return result
}
