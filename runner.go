package main

import (
	"fmt"
	"time"
)

func runner(
	ushard int16,
	timerStore TimerStoreForRunner,
	startAt time.Time,
	wallclock <-chan time.Time,
) {
	// TODO: Use runner state store to read and persist startAt
	workerTicker := make(chan time.Time, 10)
	go virtclock(wallclock, startAt, workerTicker)
	for timestamp := range workerTicker {
		pending, err := timerStore.GetPendingTimers(timestamp, ushard)
		if err != nil {
			panic(err)
		}
		if true || len(pending) > 0 {
			fmt.Printf("%d: %s %+v\n", ushard, timestamp.Format(time.RFC3339), pending)
		}
		var updates []TimerUpdate
		for _, timer := range pending {
			updates = append(updates, TimerUpdate{
				timer.TenantId,
				timer.TimerId,
				timer.Ushard,
				timer.NextAt.Add(time.Minute),
				false,
			})
		}
		err = timerStore.UpdateTimers(updates)
		if err != nil {
			panic(err)
		}
	}
}
