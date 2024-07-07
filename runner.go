package main

import (
	"time"

	"github.com/rs/zerolog/log"
)

func runner(
	ushard int16,
	timerDb TimerStoreForRunner,
	runnerDb RunnerStore,
	historyDb HistoryStore,
	gateway MessagingGateway,
	wallclock <-chan time.Time,
	quit chan<- error,
) {
	log.Debug().Int16("ushard", ushard).Msg("runner starting")
	myState, err := runnerDb.GetState(ushard)
	if err != nil {
		quit <- err
		return
	}
	startAt := myState.Next
	isNew := false
	if startAt.IsZero() {
		startAt = time.Now()
		isNew = true
	}
	log.Info().Int16("ushard", ushard).Time("startAt", startAt).Bool("isNew", isNew).Msg("runner started")
	dispatcher, err := gateway.GetDispatcherForRunner()
	if err != nil {
		quit <- err
		return
	}

	workerTicker := make(chan time.Time, 10)
	go virtclock(wallclock, startAt, workerTicker)

	for timestamp := range workerTicker {
		myState.Next = timestamp
		err := runnerDb.SaveState(myState)
		if err != nil {
			quit <- err
			return
		}
		pending, err := timerDb.GetPendingTimers(timestamp, ushard)
		if err != nil {
			quit <- err
			return
		}

		updates := make([]TimerUpdate, 0, len(pending))
		invocations := make([]*Timer, 0, len(pending))
		var failures []error

		resultsChan := make(chan DispatchResult, 100)
		expectedOutcomes := 0
		for _, timer := range pending {
			if timer.Done {
				continue
			}
			updateIfSuccessful := &TimerUpdate{Timer: &timer, SetNextAt: timer.Next()}
			if timer.Schedule == "" {
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

		// TODO: Parallelize?
		err = historyDb.LogTimerInvocations(invocations)
		if err != nil {
			quit <- err
			return
		}
		err = timerDb.UpdateTimers(updates)
		if err != nil {
			quit <- err
			return
		}
	}

	// Signal to the supervisor that we're done.
	quit <- nil
}
