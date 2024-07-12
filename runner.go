package main

import (
	"time"
)

func runner(
	ushard int16,
	clock <-chan time.Time,
	tickCompletions chan<- time.Time,
	timerDb TimerStoreForRunner,
	historyDb HistoryStore,
	gateway MessagingGateway,
	quit chan<- error,
) {
	defer close(tickCompletions)

	dispatcher, err := gateway.GetDispatcherForRunner()
	if err != nil {
		quit <- err
		return
	}

	for timestamp := range clock {
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
			updateIfSuccessful := &TimerUpdate{
				Timer:               &timer,
				SetNextAt:           timer.Next(),
				SetNextInvocationId: GenInvocationId(),
			}
			if timer.Schedule == "" {
				updateIfSuccessful.IsDone = true
			}

			go dispatcher.Dispatch(TimerMessage{
				timer.NextAt.Format(time.RFC3339),
				timer.TenantId,
				timer.TimerId,
				timer.NextInvocationId,
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

		// If we failed to publish at least one timer message, quit instead of progressing
		if len(failures) > 0 {
			quit <- failures[0]
			return
		}
		// Mark iteration as done to our supervisor
		tickCompletions <- timestamp
	}

	// Signal to the supervisor that we're all done.
	quit <- nil
}
