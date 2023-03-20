package models

import "fmt"

type Matchers struct {
	IsEqual bool   `json:"isEqual" yaml:"isEqual"`
	IsRegex bool   `json:"isRegex" yaml:"isRegex"`
	Name    string `json:"name" yaml:"name"`
	Value   string `json:"value" yaml:"value"`
}

func (o Matchers) String() string {
	return fmt.Sprintf("%#v",o)
}
