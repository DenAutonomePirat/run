package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("Hej mide")
	web := NewWeb()

	measurements := make(chan Muxable)
	measure := NewMeasure(measurements)

	store := NewStore()

	go measure.Loop()

	go func() {
		for {

			measurement := <-measurements

			if measurementEnded, ok := measurement.(*MeasurementEnded); ok {
				id, err := store.Add(measurementEnded)

				if err != nil {
					log.Printf("Could not save measurementEnded: %s", err.Error())
					continue
				}

				measurementEnded.Id = id.Hex()
			}

			web.mux.Broadcast <- measurement
		}
	}()

	web.ListenAndServe(store)

}
