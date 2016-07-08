package main

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

const (
	PREPOSTDISTANCEM    = 1.0
	POSTFINISHDISTANCEM = 20.0
	MAXSPEED            = 10.0
	MINSPEED            = 0.1
)

type Measure struct {
	output     chan Muxable
	pre        chan BarrierEvent
	post       chan BarrierEvent
	finish     chan BarrierEvent
	preRunners []Runner
	runners    []Runner
}
type Runner struct {
	Id                string        `json:"id" bson:"-"`
	preStarted        time.Time     `json:"-"`
	started           time.Time     `json:"started"`
	preSpeed          float64       `json:"-"`
	estimatedDuration time.Duration `json:"-"`
	estimatedEnded    time.Time     `json:"-"`
	Ended             time.Time     `json:"ended"`
	Duration          time.Duration `json:"durationNs"`
	DurationReadable  string        `json:"durationHumanReadable"`
	Speed             float64       `json:"speed"`
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

	pre := make(chan BarrierEvent)
	post := make(chan BarrierEvent)
	finish := make(chan BarrierEvent)

	NewBarriers(pre, post, finish)
	go keyboard(pre, post, finish)

	return &Measure{output: comms, pre: pre, post: post, finish: finish}
}

func (m *Measure) Loop() {
	for {
		select {
		case pre := <-m.pre:
			m.preBarrier(pre.Time)

		case post := <-m.post:
			m.postBarrier(post.Time)

		case finish := <-m.finish:
			m.finishBarrier(finish.Time)

		}
	}
}

func (m *Measure) preBarrier(t time.Time) {
	m.preRunners = append(m.preRunners, Runner{preStarted: t})
}

func (m *Measure) postBarrier(t time.Time) {
	if len(m.preRunners) < 1 {
		fmt.Println("no prerunners in array")
		return
	}

	r := m.preRunners[0]
	m.preRunners = m.preRunners[1:]

	dur := t.Sub(r.preStarted)
	r.preSpeed = PREPOSTDISTANCEM / dur.Seconds()

	r.estimatedDuration = (time.Duration(POSTFINISHDISTANCEM/r.preSpeed) * time.Second)

	r.estimatedEnded = r.preStarted.Add(r.estimatedDuration)
	fmt.Printf("estimated duration: %2.2v \n", r.estimatedDuration)
	if r.preSpeed > MAXSPEED {
		fmt.Printf("Too fast for this runner: %f, max: %f\n", r.preSpeed, MAXSPEED)
		return
	}

	if r.preSpeed < MINSPEED {
		fmt.Printf("Too slow though pre and post barriers: %f, min: %f\n", r.preSpeed, MINSPEED)
		return
	}

	r.started = t
	fmt.Printf("pre runner moved to runners, now there is %v remaining\n", len(m.preRunners))
	fmt.Printf("prerunner speed  %4.2f m/s \n", r.preSpeed)

	m.output <- &MeasurementStarted{
		Started: r.started,
		Speed:   r.preSpeed,
	}

	m.runners = append(m.runners, r)
}

func (m *Measure) finishBarrier(t time.Time) {
	if len(m.runners) < 1 {
		fmt.Println("no runners in array")

		return
	}

	lowestDiff := math.Abs(m.runners[0].estimatedEnded.Sub(t).Seconds())
	index := 0
	for i, _ := range m.runners {

		diff := math.Abs(m.runners[i].estimatedEnded.Sub(t).Seconds())
		fmt.Printf("index: %v \tDifference: %4.2f \n", i, diff)

		if diff < lowestDiff {
			lowestDiff = diff
			index = i
		}
	}
	fmt.Printf("Runner: %v \tDifference %4.2f\n", index, lowestDiff)

	r := &m.runners[index]

	r.Ended = t

	fmt.Printf("Runner took %s to finish, now there's %v runners running.\n",
		r.Ended.Sub(r.started),
		len(m.runners)-1,
	)

	m.output <- &MeasurementEnded{
		Started:          r.started,
		Ended:            r.Ended,
		Duration:         r.Ended.Sub(r.started),
		DurationReadable: fmt.Sprintf("%s", r.Ended.Sub(r.started)),
		Speed:            POSTFINISHDISTANCEM / r.Ended.Sub(r.started).Seconds(),
	}

	m.runners = append(m.runners[:index], m.runners[index+1:]...)
}
