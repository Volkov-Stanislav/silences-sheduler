package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Volkov-Stanislav/silences-sheduler/metrics"
	"github.com/Volkov-Stanislav/silences-sheduler/service"
	"github.com/Volkov-Stanislav/silences-sheduler/stats"
	"github.com/Volkov-Stanislav/silences-sheduler/storages"
	"github.com/namsral/flag"

	"go.uber.org/zap"
)

var (
	shedulesDir    string
	updateInterval string
	apiurl         string
	metricsPort    string
	statPort       string
)

func main() {
	flag.String(flag.DefaultConfigFlagname, "config", "path to config file")
	flag.StringVar(&updateInterval, "update_interval", "60", "interval for reread shedule configs")
	flag.StringVar(&metricsPort, "metrics_port", "32112", "port for scraping metrics")
	flag.StringVar(&statPort, "statistic_port", "38080", "port for statistics")
	flag.StringVar(&shedulesDir, "shedules_dir", "shedule_configs", "path to shedule configs")
	flag.StringVar(&apiurl, "apiurl", "http://localhost:9093/api/v2/silences", "alertmanager API URL")
	flag.Parse()

	fmt.Println(updateInterval)
	fmt.Println(shedulesDir)
	fmt.Println(apiurl)

	config := make(map[string]string)
	config["shedules_dir"] = shedulesDir
	config["update_interval"] = updateInterval
	config["metrics_port"] = metricsPort
	config["statistic_port"] = statPort

	prom := metrics.NewPrometheusInstance(metricsPort)
	prom.Run()

	defer prom.Stop()

	log, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("Error initialithing logging:  (%v)?", err))
	}

	defer log.Sync()

	stat := stats.NewInstance(statPort, log)
	stat.Run()

	defer stat.Stop()

	serv, _ := service.NewRunner(apiurl, log, stat, prom)
	serv.Start()

	shedcsv, err := storages.GetCSVStorage(config, log)
	if err != nil {
		fmt.Printf("Error get CSV storage object: %v", err)
	}

	shedcsv.Run(serv.GetChannels())

	shedyaml, err := storages.GetYAMLStorage(config, log)
	if err != nil {
		fmt.Printf("Error get YAML storage object: %v", err)
	}

	shedyaml.Run(serv.GetChannels())

	var (
		hup  = make(chan os.Signal, 1)
		term = make(chan os.Signal, 1)
	)

	signal.Notify(hup, syscall.SIGHUP)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-hup:
			log.Info("Received SIGHUP, do nothing...")
		case <-term:
			log.Info("Received SIGTERM, exiting gracefully...")
			return
		}
	}
}
