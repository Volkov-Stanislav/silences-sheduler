package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type Shedule struct {
	Cron     string  `yaml:"cron"`     // Crontab defaining time to start silence.
	Duration int     `yaml:"duration"` // Duration of silence in seconds.
	Silence  Silence `yaml:"silence"`  // Duration of silence in seconds.
}

func (o Shedule) String() string {
	return fmt.Sprintf("%#v",o)
}

func (o *Shedule) Run(apiURL string, log *zap.Logger) {
	o.Silence.StartsAt = time.Now()
	o.Silence.EndsAt = time.Now().Add(time.Duration(int64(o.Duration) * int64(time.Second)))

	_, err := postAPI(apiURL, o.Silence, log)
	if err != nil {
		log.Sugar().Errorf("Error POST in Alertmanager API:  %v", err)
	}
}

func postAPI(url string, data Silence, logger *zap.Logger) (SilenceID, error) {
	var result SilenceID

	ctx := context.Background()
	timeout := 30 * time.Second
	reqContext, cancel := context.WithTimeout(ctx, timeout)

	defer cancel()

	b, err := json.Marshal(data)
	if err != nil {
		return result, err
	}

	byteReader := bytes.NewReader(b)

	r, err := http.NewRequestWithContext(reqContext, "POST", url, byteReader)
	if err != nil {
		return result, err
	}

	r.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(r)
	if err != nil {
		return result, err
	}

	h, _ := json.Marshal(data)
	logger.Sugar().Infof("Called POST body %v result: %v \n", string(h), res)

	body, err := io.ReadAll(res.Body)

	res.Body.Close()

	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return result, err
	}

	return result, nil
}
