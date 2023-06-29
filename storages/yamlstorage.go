// Package storages implements functions that periodicaly load shedules from YAML files in specified derectory.
package storages

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Volkov-Stanislav/silences-sheduler/models"
	"github.com/datadog/mmh3"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// YAMLstorage implementation persing YAML config files.
type YAMLstorage struct {
	directoryName  string // Directory with shedules configs.
	updateInterval int    // Update interval of config from files
	sheds          map[string]bool
	logger         *zap.Logger
}

// GetYAMLStorage return configured yaml storage.
func GetYAMLStorage(config map[string]string, logger *zap.Logger) (*YAMLstorage, error) {
	var (
		storage YAMLstorage
		err     error
	)

	dirName, ok := config["shedules_dir"]
	if !ok {
		return nil, fmt.Errorf("config Param -shedules_dir- not found")
	}

	storage.directoryName = dirName

	intrvl, ok := config["update_interval"]
	if !ok {
		return nil, fmt.Errorf("config Param -update_interval- not found")
	}

	storage.updateInterval, err = strconv.Atoi(intrvl)
	if err != nil {
		logger.Sugar().Errorf("parsing 'update_interval' parameter: %v error: %v", intrvl, err)
	}

	storage.sheds = make(map[string]bool)
	storage.logger = logger

	return &storage, nil
}

// Run parsing and update checking of yaml files.
func (o *YAMLstorage) Run(add chan models.SheduleSection, del chan string) {
	go o.run(add, del)
}

// Parse all shedules from yaml file.
func (o *YAMLstorage) FillAllShedules() (shedules map[string]models.SheduleSection, err error) {
	shedules = make(map[string]models.SheduleSection)

	err = filepath.Walk(o.directoryName,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && filepath.Ext(path) == ".yaml" {
				shedSection, err := o.fillShedule(path, info)
				if err != nil {
					return err
				}

				shedules[shedSection.GetToken()] = *shedSection
			}
			return nil
		})

	return
}

func (o *YAMLstorage) fillShedule(fileName string, info os.FileInfo) (*models.SheduleSection, error) {
	var shedSect models.SheduleSection

	token := fileName + "|" + info.ModTime().String()

	shedSect.SetToken(hex.EncodeToString(mmh3.Hash128([]byte(token)).Bytes()))
	shedSect.SetSectionName(info.Name())

	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)

	err = decoder.Decode(&shedSect)
	if err != nil {
		o.logger.Sugar().Errorf("decode file '%v' error: %v", fileName, err)
		return nil, err
	}

	return &shedSect, nil
}

func (o *YAMLstorage) run(add chan models.SheduleSection, del chan string) {
	err := o.update(add, del)
	if err != nil {
		return
	}

	tim := time.NewTicker(time.Second * time.Duration(o.updateInterval))
	defer tim.Stop()

	for {
		t := <-tim.C
		o.logger.Sugar().Infof("Tick on %v", t)

		err := o.update(add, del)
		if err != nil {
			return
		}
	}
}

func (o *YAMLstorage) update(add chan models.SheduleSection, del chan string) error {
	newShed, err := o.FillAllShedules()
	if err != nil {
		return err
	}

	// Add New shedules.
	for key, val := range newShed {
		if _, ok := o.sheds[key]; !ok {
			add <- val

			o.sheds[key] = true
		}
	}

	// Remove non existent Shedules.
	for key := range o.sheds {
		if _, ok := newShed[key]; !ok {
			del <- key
			delete(o.sheds, key)
		}
	}

	return nil
}
