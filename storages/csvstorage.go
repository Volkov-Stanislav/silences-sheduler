// Storage that periodicaly load shedules from CSV files in specified derectory.
// CSV file format:
//  "hostname","shedule","timeshift from sheduler server"
// Example:
// "udbs01","SCCM-Updates-MW_1_Thu_02","03:00:00"
// in shedule field, split by '_':
// "backup_system_name",
// week number in month
// day in week
// hour

package storages

import (
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Volkov-Stanislav/silences-sheduler/models"
	"github.com/Volkov-Stanislav/silences-sheduler/utils"
	"github.com/datadog/mmh3"
	"go.uber.org/zap"
)

type CSVstorage struct {
	directoryName  string // Directory with shedules configs.
	updateInterval int    // Update interval of config from files
	sheds          map[string]bool
	logger         *zap.Logger
}

func GetCSVStorage(config map[string]string, logger *zap.Logger) (*CSVstorage, error) {
	var (
		storage CSVstorage
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

func (o *CSVstorage) Run(add chan models.SheduleSection, del chan string) {
	go o.run(add, del)
}

func (o *CSVstorage) FillAllShedules() (shedules []models.SheduleSection, err error) {
	shedules = append(shedules, models.SheduleSection{})

	err = filepath.Walk(o.directoryName,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && filepath.Ext(path) == ".csv" {
				shedSections, err := o.fillShedule(path, info)
				if err != nil {
					return err
				}

				shedules = append(shedules, shedSections...)
			}

			return nil
		})

	return
}

// читает отдельный файл с shedules, заполняет из него записи shedules нижнего уровня.
func (o *CSVstorage) fillShedule(fileName string, info os.FileInfo) ([]models.SheduleSection, error) {
	var shedSect []models.SheduleSection

	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	shedSect = o.decode(file, fileName, info) // теперь много секций, с группировкой по смещениям к таймзоне сервера.

	fmt.Printf("After decoding: %v\n", shedSect)

	return shedSect, nil
}

// Декодирование CSV файла, создание сайленсов по шаблону из него.
func (o *CSVstorage) decode(file io.Reader,
	fileName string,
	info os.FileInfo) []models.SheduleSection {
	var shedd []models.SheduleSection

	var sheduleTemplate = models.Shedule{
		Cron:     "",
		Duration: 10800, // 3 hours in sec.
		Silence: models.Silence{
			Comment:   "Automatic silence for OS Update 3h",
			CreatedBy: "SilenceSheduler",
			Matchers: []models.Matchers{
				{IsEqual: true, IsRegex: true, Name: "hostname", Value: "~"},
			},
		},
	}

	csvReader := csv.NewReader(file)
	csvReader.Comma = ','
	csvReader.LazyQuotes = true

	lines, err := csvReader.ReadAll()
	if err != nil {
		o.logger.Sugar().Errorf("Error reading CSV file: %v", err)
		return shedd
	}
	// Remove header of CSV file from readed strings.
	lines = lines[1:]
	sectOfLines := utils.SorterSplitter(lines).Split()

	var location *time.Location

	for _, shedSect := range sectOfLines {
		if len(shedSect) > 0 && len(shedSect[0]) > 0 {
			location = utils.GetLocation(shedSect[0][2])
		} else {
			location = time.Local
		}

		shedd = append(shedd, models.SheduleSection{})
		token := fileName + "|" + info.ModTime().String() + location.String() // теперь в токене также будет и смещение таймзоны в часах.
		shedd[len(shedd)-1].SetToken(hex.EncodeToString(mmh3.Hash128([]byte(token)).Bytes()))
		shedd[len(shedd)-1].SetSectionName(info.Name())

		_, offset := time.Now().In(location).Zone()
		shedd[len(shedd)-1].TimeOffset = fmt.Sprint(int(offset / 60 / 60))

		for _, line := range shedSect {
			rec := sheduleTemplate

			// Set host name
			rec.Silence.Matchers = []models.Matchers{
				{
					IsEqual: true,
					IsRegex: true,
					Name:    "hostname",
					Value:   "",
				},
			}
			rec.Silence.Matchers[0].Value = line[0] + ".+"

			// Set Cron shedule
			timeArr := strings.Split(line[1], "_")
			if len(timeArr) < 3 {
				continue
			}

			weeknum, err := strconv.Atoi(timeArr[len(timeArr)-3])
			if err != nil {
				continue
			}

			dow := timeArr[len(timeArr)-2]
			hour, err := strconv.Atoi(timeArr[len(timeArr)-1])

			if err != nil {
				continue
			}

			rec.Cron = "0 0 " + fmt.Sprint(hour) + " * * " + dow + "#" + fmt.Sprint(weeknum)
			rec.Silence.Comment = line[0] + " | " + line[1] + " | " + line[2]
			shedd[len(shedd)-1].Shedules = append(shedd[len(shedd)-1].Shedules, rec)

			fmt.Println(rec.String())
		}
	}

	return shedd
}

// горутина, в которой идет периодическое чтение конфига, и если требуются обновления идут сигналы в каналы.
func (o *CSVstorage) run(add chan models.SheduleSection, del chan string) {
	err := o.update(add, del)
	if err != nil {
		return
	}

	// цикл чтения с диска, сравнения со текущим конфигом, обновления через каналы руннера.
	tim := time.NewTicker(time.Second * time.Duration(o.updateInterval*3600))
	defer tim.Stop()

	for {
		select {
		case t := <-tim.C:
			o.logger.Sugar().Infof("Tick on %v", t)

			err := o.update(add, del)
			if err != nil {
				return
			}
		}
	}
}

func (o *CSVstorage) update(add chan models.SheduleSection, del chan string) error {
	allShed, err := o.FillAllShedules()
	if err != nil {
		return err
	}

	newShed := make(map[string]bool)

	// Add New shedules.
	for _, val := range allShed {
		if _, ok := o.sheds[val.GetToken()]; !ok {
			add <- val

			o.sheds[val.GetToken()] = true
		}

		newShed[val.GetToken()] = true
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
