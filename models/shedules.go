package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Volkov-Stanislav/cron"
	"github.com/Volkov-Stanislav/silences-sheduler/metrics"
	"github.com/Volkov-Stanislav/silences-sheduler/stats"
	"go.uber.org/zap"
)

// Shedule define cron task for silence.
type Shedule struct {
	Cron     string       `yaml:"cron"`     // Crontab defaining time to start silence.
	Duration int          `yaml:"duration"` // Duration of silence in seconds.
	Silence  Silence      `yaml:"silence"`  // Silence define.
	entryID  cron.EntryID // ID of cron task.
}

func (o Shedule) String() string {
	return fmt.Sprintf("%#v", o)
}

// Run shedule.
func (o *Shedule) Run(apiURL string, log *zap.Logger, stat *stats.Instance, prom *metrics.Instance) {
	o.Silence.StartsAt = time.Now().UTC().Add(time.Duration(int64(-10) * int64(time.Minute)))
	o.Silence.EndsAt = time.Now().UTC().Add(time.Duration(int64(o.Duration) * int64(time.Second)))

	_, err := postAPI(apiURL, o.Silence, log, stat)
	if err != nil {
		log.Sugar().Errorf("Error POST in Alertmanager API:  %v", err)
	}
	
	prom.AddSilencesCounter(1)
}

// Return ID for cron task for this shedule.
func (o *Shedule) GetEntryID() cron.EntryID {
	return o.entryID
}

// Set ID for cron task for this shedule.
func (o *Shedule) SetEntryID(id cron.EntryID) {
	o.entryID = id
}

func postAPI(url string, data Silence, logger *zap.Logger, stat *stats.Instance) (SilenceID, error) {
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

	stat.AddSheduleRun(fmt.Sprintf("%v;%v;%v;%s;%#v\n", time.Now().UTC(), data.StartsAt, data.EndsAt, data.Comment, data.Matchers))

	return result, nil
}
