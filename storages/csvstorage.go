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
	"github.com/datadog/mmh3"
	"go.uber.org/zap"
)

type CSVstorage struct {
	directoryName  string // Directory with shedules configs.
	updateInterval int    // Update interval of config from files
	stop           chan bool
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

	storage.stop = make(chan bool)
	storage.sheds = make(map[string]bool)
	storage.logger = logger

	return &storage, nil
}

// запускает работу хранилища по первому заполнению и периодическому апдейту данных.
func (o *CSVstorage) Run(add chan models.SheduleSection, del chan string) {
	go o.run(add, del)
}

func (o *CSVstorage) Stop() {
	o.stop <- true
}

// функция читает все файлы в директории с конфигaми, заполняет shedules из них.
func (o *CSVstorage) FillAllShedules() (shedules map[string]models.SheduleSection, err error) {
	shedules = make(map[string]models.SheduleSection)

	err = filepath.Walk(o.directoryName,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && filepath.Ext(path) == ".csv" {
				shedSection, err := o.fillShedule(path, info)
				if err != nil {
					return err
				}

				shedules[shedSection.Token] = *shedSection
			}
			return nil
		})

	return
}

// читает отдельный файл с shedules, заполняет из него записи shedules нижнего уровня.
func (o *CSVstorage) fillShedule(fileName string, info os.FileInfo) (*models.SheduleSection, error) {
	var shedSect models.SheduleSection

	token := fileName + "|" + info.ModTime().String()

	shedSect.Token = hex.EncodeToString(mmh3.Hash128([]byte(token)).Bytes())
	shedSect.SectionName = info.Name()

	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	o.decode(file, &shedSect)

	return &shedSect, nil
}

// Декодирование CSV файла, создание сайленсов по шаблону из него.
func (o *CSVstorage) decode(file io.Reader, shed *models.SheduleSection) {
	var sheduleTemplate = models.Shedule{
		Cron:     "",
		Duration: 10800, // 3 hours in sec.
		Silence: models.Silence{
			Comment:   "Automatic silence for OS Update 3h",
			CreatedBy: "SilenceSheduler",
			Matchers: []models.Matchers{
				{
					IsEqual: true,
					IsRegex: true,
					Name:    "hostname",
					Value:   "~",
				},
			},
		},
	}

	csvReader := csv.NewReader(file)
	i := 0

	for {
		line, err := csvReader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			o.logger.Sugar().Errorf("Error reading CSV file: %v", err)
		}

		if i > 0 && len(line) == 3 { // omit header line & bad lines.
			rec := sheduleTemplate

			// Set host name
			rec.Silence.Matchers[0].Value = "~" + line[0]

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

			utc := 0
			utcsplit := strings.Split(line[2], ":")

			if len(utcsplit) == 3 {
				utc, _ = strconv.Atoi(utcsplit[0])
			}

			rec.Cron = "* * " + fmt.Sprint(hour+utc) + " * * " + dow + "#" + fmt.Sprint(weeknum)

			shed.Shedules = append(shed.Shedules, rec)

			fmt.Println(rec.String())
		}
		i++
	}
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
		case <-o.stop:
			o.logger.Error("singnal STOP received, stop work of CSVstorage")
			return
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
	newShed, err := o.FillAllShedules()
	if err != nil {
		return err
	}

	// Add New shedules.
	for key, val := range newShed {
		if _, ok := o.sheds[key]; !ok {
			fmt.Printf("Add new Entry: %v \n", val)
			add <- val

			o.sheds[key] = true
		}
	}

	// Remove non existent Shedules.
	for key := range o.sheds {
		if _, ok := newShed[key]; !ok {
			fmt.Printf("Del Entry: %v \n", key)
			del <- key
			delete(o.sheds, key)
		}
	}

	return nil
}
