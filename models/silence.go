package models

import (
	"fmt"
	"time"
)

// Silence type of Alertmanager silence.
type Silence struct {
	Comment   string     `json:"comment" yaml:"comment"`
	CreatedBy string     `json:"createdBy" yaml:"createdBy"`
	EndsAt    time.Time  `json:"endsAt" yaml:"endsAt"`
	StartsAt  time.Time  `json:"startsAt" yaml:"startAt"`
	Matchers  []Matchers `json:"matchers" yaml:"matchers"`
}

// String stringer interface.
func (o Silence) String() string {
	return fmt.Sprintf("%#v", o)
}

// SilenceID type for parsing reply from Alertmanager.
type SilenceID struct {
	SilenceID string `json:"silenceID"`
}

// String stringer interface.
func (o SilenceID) String() string {
	return fmt.Sprintf("SilenceID: %v \n", o.SilenceID)
}
