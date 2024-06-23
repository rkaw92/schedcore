package main

import (
	"fmt"
	"time"
)

func runner(
	ushard int16,
	timerStore TimerStoreForRunner,
	gateway MessagingGateway,
	startAt time.Time,
	wallclock <-chan time.Time,
) {
	// TODO: Use runner state store to read and persist startAt
	workerTicker := make(chan time.Time, 10)
	go virtclock(wallclock, startAt, workerTicker)
	// TODO: Error handling for init
	dispatcher, _ := gateway.GetDispatcherForRunner()
	for timestamp := range workerTicker {
		pending, err := timerStore.GetPendingTimers(timestamp, ushard)
		if err != nil {
			panic(err)
		}
		if true || len(pending) > 0 {
			fmt.Printf("%d: %s %+v\n", ushard, timestamp.Format(time.RFC3339), pending)
		}
		var updates []TimerUpdate
		var failures []error
		resultsChan := make(chan DispatchResult, 100)
		expectedOutcomes := 0
		for _, timer := range pending {
			updateIfSuccessful := &TimerUpdate{
				timer.TenantId,
				timer.TimerId,
				timer.Ushard,
				timer.NextAt.Add(time.Minute),
				false,
			}
			go dispatcher.Dispatch(TimerMessage{
				timer.NextAt.Format(time.RFC3339),
				timer.TenantId,
				timer.TimerId,
				timer.Payload,
				timer.Destination,
			}, updateIfSuccessful, resultsChan)
			expectedOutcomes++
		}

		for {
			if len(updates)+len(failures) == expectedOutcomes {
				break
			}
			result := <-resultsChan
			if result.err != nil {
				failures = append(failures, result.err)
			} else {
				updates = append(updates, *result.update)
			}
		}

		err = timerStore.UpdateTimers(updates)
		if err != nil {
			panic(err)
		}
	}
}
