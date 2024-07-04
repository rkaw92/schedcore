package main

import (
	"fmt"
	"time"

	"github.com/robfig/cron"
)

func runner(
	ushard int16,
	timerDb TimerStoreForRunner,
	runnerDb RunnerStore,
	historyDb HistoryStore,
	gateway MessagingGateway,
	wallclock <-chan time.Time,
) {
	scheduleParser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.DowOptional)
	myState, err := runnerDb.GetState(ushard)
	if err != nil {
		panic(err)
	}
	startAt := myState.Next
	if startAt.IsZero() {
		startAt = time.Now()
	}
	workerTicker := make(chan time.Time, 10)
	go virtclock(wallclock, startAt, workerTicker)
	dispatcher, err := gateway.GetDispatcherForRunner()
	if err != nil {
		panic(err)
	}
	for timestamp := range workerTicker {
		myState.Next = timestamp
		err := runnerDb.SaveState(myState)
		if err != nil {
			panic(err)
		}
		pending, err := timerDb.GetPendingTimers(timestamp, ushard)
		if err != nil {
			panic(err)
		}
		if true || len(pending) > 0 {
			fmt.Printf("%d: %s %+v\n", ushard, timestamp.Format(time.RFC3339), pending)
		}
		updates := make([]TimerUpdate, 0, len(pending))
		invocations := make([]*Timer, 0, len(pending))
		var failures []error
		resultsChan := make(chan DispatchResult, 100)
		expectedOutcomes := 0
		for _, timer := range pending {
			if timer.Done || !timer.Enabled {
				continue
			}
			updateIfSuccessful := &TimerUpdate{Timer: &timer, SetNextAt: timer.NextAt}
			if timer.Schedule != "" {
				schedule, err := scheduleParser.Parse(timer.Schedule)
				if err != nil {
					continue
				}
				updateIfSuccessful.SetNextAt = schedule.Next(timer.NextAt)
			} else {
				updateIfSuccessful.IsDone = true
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
				invocations = append(invocations, result.update.Timer)
			}
		}

		err = timerDb.UpdateTimers(updates)
		if err != nil {
			panic(err)
		}
		err = historyDb.LogTimerInvocations(invocations)
		if err != nil {
			panic(err)
		}
		// TODO: DON'T PANIC!
	}
}
