package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

type DummyDb struct {
	timers  []Timer
	history []Timer
}

func (db *DummyDb) GetPendingTimers(next_at time.Time, ushard int16) ([]Timer, error) {
	var pending []Timer
	for _, timer := range db.timers {
		if timer.NextAt == next_at && timer.Ushard == ushard {
			pending = append(pending, timer)
		}
	}
	return pending, nil
}

func (db *DummyDb) UpdateTimers(updates []TimerUpdate) error {
	// This is O(n²) but it's just a test :)
	for _, update := range updates {
		for dbIndex, timer := range db.timers {
			if update.Timer.TenantId == timer.TenantId && update.Timer.TimerId == timer.TimerId {
				db.timers[dbIndex].Done = update.IsDone
				db.timers[dbIndex].NextAt = update.SetNextAt
				db.timers[dbIndex].NextInvocationId = update.SetNextInvocationId
			}
		}
	}
	return nil
}

func (db *DummyDb) LogTimerInvocations(timers []*Timer) error {
	for _, timer := range timers {
		db.history = append(db.history, *timer)
	}
	return nil
}

type DummyGateway struct {
	messages []TimerMessage
}

func (gateway *DummyGateway) GetDispatcherForRunner() (TimerDispatcher, error) {
	return &DummyDispatcher{gateway}, nil
}

type DummyDispatcher struct {
	parent *DummyGateway
}

func (disp *DummyDispatcher) Dispatch(msg TimerMessage, update *TimerUpdate, results chan<- DispatchResult) {
	disp.parent.messages = append(disp.parent.messages, msg)
	results <- DispatchResult{update, nil}
}

func (disp *DummyDispatcher) Destroy() error {
	return nil
}

func TestRunnerBasicOperation(t *testing.T) {
	// Arrange
	db := &DummyDb{
		timers: []Timer{{
			// "we were throwing rocks at dinosaurs"
			NextAt:           time.Time{},
			TenantId:         uuid.MustParse("f95b37f3-a32d-401e-81d3-fb520d61f9b9"),
			TimerId:          uuid.MustParse("9d77c38d-8f83-459e-9ccc-f71c206c73c2"),
			Ushard:           42,
			Schedule:         "",
			Done:             false,
			Payload:          map[string]string{"Hello": "world"},
			Destination:      "test/topic",
			NextInvocationId: uuid.MustParse("300d142c-f2cf-479d-b9e5-8b1aaae8ef9b"),
		}},
	}
	gw := &DummyGateway{}
	const TEST_USHARD = 42
	clock := make(chan time.Time, 1)
	completions := make(chan time.Time, 1)
	quit := make(chan error, 1)
	// Act
	go runner(TEST_USHARD, clock, completions, db, db, gw, quit)
	clock <- time.Time{}
	close(clock)
	// Assert
	completion := <-completions
	runnerErr := <-quit
	if !completion.IsZero() {
		t.Fatalf("completion time is wrong, expect %v, got %v", time.Time{}, completion)
	}
	if runnerErr != nil {
		t.Fatalf("runner quit with error %v", runnerErr)
	}
	if len(db.history) != 1 {
		t.Fatal("runner did not register timer invocation in history DB")
	}
	if !db.timers[0].Done {
		t.Fatal("runner did not mark timer as done")
	}
	if len(gw.messages) != 1 {
		t.Fatal("runner did not send message via dispatcher")
	}
}
