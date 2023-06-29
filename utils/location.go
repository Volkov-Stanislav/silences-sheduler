package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// GetLocation convert offset into Location.
// inOffset = "hh24:mm:ss:", mean offset from UTC, func use only hour (hh24). Or, int value interpreted as osset too.
func GetLocation(inOffset string) *time.Location {
	tLocal := time.Local

	if inOffset == "" {
		return tLocal
	}

	TimeOffset := 0
	utcsplit := strings.Split(inOffset, ":")

	if len(utcsplit) == 3 {
		TimeOffset, _ = strconv.Atoi(utcsplit[0])
		if TimeOffset > 12 || TimeOffset < -12 {
			return tLocal
		}
	} else {
		if tOffset, err := strconv.Atoi(inOffset); err != nil {
			return tLocal
		} else {
			TimeOffset = tOffset
		}
	}

	return time.FixedZone("UTC"+fmt.Sprint(TimeOffset), TimeOffset*60*60)
}
