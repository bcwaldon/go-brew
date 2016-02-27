package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const DIR_DEVICES = "/sys/bus/w1/devices/"

var re_data = regexp.MustCompile(`^.*t=([0-9]*)$`)

type TempSensor struct {
	source string
}

func (t *TempSensor) TempF() (float64, error) {
	b, err := ioutil.ReadFile(t.source)
	if err != nil {
		return 0.0, err
	}
	lines := make([][]byte, 0)
	for _, l := range bytes.Split(b, []byte("\n")) {
		l := l
		if len(l) == 0 {
			continue
		}
		lines = append(lines, l)
	}

	if !bytes.HasSuffix(lines[0], []byte("YES")) {
		return 0.0, errors.New("not ready to read")
	}
	if l := len(lines); l != 2 {
		return 0.0, fmt.Errorf("expected 2 data lines, got %d", l)
	}
	matches := re_data.FindSubmatch(lines[1])[1:]
	if len(matches) == 0 {
		return 0.0, errors.New("invalid data line")
	}
	match := matches[0]
	v, err := strconv.ParseFloat(string(match), 64)
	if err != nil {
		return 0.0, fmt.Errorf("invalid data value: %v", string(match))
	}
	cel := v / 1000.0
	far := cel*9.0/5.0 + 32.0
	return far, nil
}

func (t *TempSensor) WatchF(rate time.Duration) (<-chan float64, <-chan error) {
	tempchan := make(chan float64)
	errchan := make(chan error)

	go func() {
		var last float64
		for _ = range time.Tick(rate) {
			v, err := t.TempF()
			if err != nil {
				errchan <- err
			} else if last != v {
				last = v
				tempchan <- v
			}
		}
	}()

	return (<-chan float64)(tempchan), (<-chan error)(errchan)
}

func NewTempSensor(source string) *TempSensor {
	return &TempSensor{source: source}
}

func TempSensors() ([]*TempSensor, error) {
	fis, err := ioutil.ReadDir(DIR_DEVICES)
	if err != nil {
		return nil, err
	}
	sensors := make([]*TempSensor, 0)
	for _, fi := range fis {
		if !strings.HasPrefix(fi.Name(), "28-") {
			continue
		}
		src := path.Join(DIR_DEVICES, fi.Name(), "w1_slave")
		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}
		sensors = append(sensors, NewTempSensor(src))
	}
	return sensors, nil
}

func main() {
	sensors, err := TempSensors()
	if err != nil {
		log.Fatalf("Failed loading temp sensors: %v", err)
	}
	log.Printf("Found %d temp sensors", len(sensors))
	if len(sensors) > 1 {
		log.Fatalf("Multiple sensors not supported")
	}

	sensor := sensors[0]
	tempchan, errchan := sensor.WatchF(time.Second)
	for {
		select {
		case v := <-tempchan:
			log.Printf("Sensor reading: %vF", v)
		case err := <-errchan:
			log.Printf("Sensor failure: %v", err)
		}
	}
}
