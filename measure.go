package main

import (
	"fmt"

	"time"

	"encoding/json"

	"github.com/davecheney/gpio"
)

type Measure struct {
	output chan Muxable
}

type MeasurementStarted struct {
	Started time.Time `json:"started"`
}

func (m *MeasurementStarted) Marshal() *[]byte {
	encoded, _ := json.Marshal(m)
	return &encoded

}

type MeasurementEnded struct {
	Id               string        `json:"id" bson:"-"`
	Started          time.Time     `json:"started"`
	Ended            time.Time     `json:"ended"`
	Duration         time.Duration `json:"durationNs"`
	DurationReadable string        `json:"durationHumanReadable"`
}

func (m *MeasurementEnded) Marshal() *[]byte {
	encoded, _ := json.Marshal(m)
	return &encoded
}

func NewMeasure(comms chan Muxable) *Measure {
	return &Measure{output: comms}
}

func (m *Measure) Loop() {

	// set GPIO22 to input mode
	pin, err := gpio.OpenPin(gpio.GPIO17, gpio.ModeInput)
	if err != nil {
		fmt.Printf("Error opening pin! %s\n", err)
		return
	}

	var started time.Time
	err = pin.BeginWatch(gpio.EdgeFalling, func() {
		started = time.Now()
		fmt.Printf("Callback for %d triggered!\n\n", gpio.GPIO22)
		m.output <- &MeasurementStarted{Started: started}
	})

	// ended := time.Now()
	// duration := ended.Sub(started)

	// m.output <- &MeasurementEnded{Started: started,
	// 	Ended:            ended,
	// 	Duration:         duration,
	// 	DurationReadable: fmt.Sprintf("%s", duration),
	//  }
}