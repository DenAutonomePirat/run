package main

import (
	"fmt"

	"time"

	"encoding/json"

	"github.com/davecheney/gpio"
)

const (
	PREPOSTDISTANCEM    = 0.15
	POSTFINISHDISTANCEM = 0.5
	MAXSPEED            = 10.0
	MINSPEED            = 0.1
)

type Measure struct {
	output chan Muxable
}

type MeasurementStarted struct {
	Started time.Time `json:"started"`
	Speed   float64   `json:"speed"`
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
	Speed            float64       `json:"speed"`
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
	preStartPin, err := gpio.OpenPin(gpio.GPIO17, gpio.ModeInput)
	if err != nil {
		fmt.Printf("Error opening pin! %s\n", err.Error())
		return
	}

	postStartPin, err := gpio.OpenPin(gpio.GPIO27, gpio.ModeInput)
	if err != nil {
		fmt.Print("Error opening pin %s\n", err.Error())
		return
	}

	finishPin, err := gpio.OpenPin(gpio.GPIO22, gpio.ModeInput)
	if err != nil {
		fmt.Printf("Error opening pin! %s\n", err.Error())
		return
	}

	var starters []MeasurementStarted = make([]MeasurementStarted, 0, 5)

	var currentStarter *time.Time

	err = preStartPin.BeginWatch(gpio.EdgeFalling, func() {
		fmt.Printf("Hej\n", "hej")
		tmp := time.Now()
		currentStarter = &tmp
	})

	err = postStartPin.BeginWatch(gpio.EdgeFalling, func() {
		fmt.Printf("Callback for start line called!\n")

		if currentStarter == nil {
			fmt.Println("No starting without running though first light barrier")
			return
		}

		started := time.Now()

		dur := started.Sub(*currentStarter)
		currentStarter = nil

		speed := PREPOSTDISTANCEM / dur.Seconds()

		if speed > MAXSPEED {
			// too fast
			fmt.Printf("Too fast though pre and post barriers: %f, max: %f\n", speed, MAXSPEED)
			return
		}

		if speed < MINSPEED {
			fmt.Printf("Too slow though pre and post barriers: %f, min: %f\n", speed, MINSPEED)
			// too slow
			return
		}

		mStarted := MeasurementStarted{Started: started, Speed: speed}

		starters = append(starters, mStarted)
		m.output <- &mStarted
	})

	err = finishPin.BeginWatch(gpio.EdgeFalling, func() {
		fmt.Printf("Callback for finish line triggered!\n")
		//remove this "starter"
		if len(starters) <= 0 {
			return
		}

		ended := time.Now()

		mStarted := starters[0]

		minDuration := (mStarted.Speed - 3) / POSTFINISHDISTANCEM
		maxDuration := (mStarted.Speed + 3) / POSTFINISHDISTANCEM

		duration := ended.Sub(mStarted.Started)

		if duration.Seconds() < minDuration {
			fmt.Printf("This runner is way too fast: %s, %f min\n", duration, minDuration)
			return
		}

		// remove this starter
		starters = starters[1:]

		if duration.Seconds() > maxDuration {
			fmt.Printf("This runner took to long: %s, %f allowed\n", duration, maxDuration)
			return
		}

		m.output <- &MeasurementEnded{Started: mStarted.Started,
			Ended:            ended,
			Duration:         duration,
			DurationReadable: fmt.Sprintf("%s", duration),
		}

	})
	// ended := time.Now()
	// duration := ended.Sub(started)

	// m.output <- &MeasurementEnded{Started: started,
	// 	Ended:            ended,
	// 	Duration:         duration,
	// 	DurationReadable: fmt.Sprintf("%s", duration),
	//  }
}
