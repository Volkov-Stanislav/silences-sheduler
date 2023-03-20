package models

import (
	"fmt"
	"time"
)

type Silence struct {
	Comment   string     `json:"comment" yaml:"comment"`
	CreatedBy string     `json:"createdBy" yaml:"createdBy"`
	EndsAt    time.Time  `json:"endsAt" yaml:"endsAt"`
	StartsAt  time.Time  `json:"startsAt" yaml:"startAt"`
	Matchers  []Matchers `json:"matchers" yaml:"matchers"`
}

func (o Silence) String() string {
	return fmt.Sprintf("%#v",o)
}

type SilenceID struct {
	SilenceID string `json:"silenceID"`
}

func (o SilenceID) String() string {
	return fmt.Sprintf("SilenceID: %v \n", o.SilenceID)
}
