package main

import (
	"time"

	"github.com/google/uuid"
)

type TimerMessage struct {
	ISODate     string      `json:"isoDate"`
	TenantId    uuid.UUID   `json:"tenantId"`
	TimerId     uuid.UUID   `json:"timerId"`
	Payload     interface{} `json:"payload"`
	Destination string      `json:"-"`
}

type RunnerState struct {
	MyUshard int16
	Next     time.Time
}

type TimerUpdate struct {
	Timer     *Timer
	SetNextAt time.Time
	IsDone    bool
}

type TimerStoreForRunner interface {
	GetPendingTimers(next_at time.Time, ushard int16) ([]Timer, error)
	UpdateTimers(updates []TimerUpdate) error
}

type TimerStoreForAdmin interface {
	Create(timer *Timer) error
	Enable(tenantId uuid.UUID, timerId uuid.UUID)
	Disable(tenantId uuid.UUID, timerId uuid.UUID)
}

type RunnerStore interface {
	GetState(myUshard int16) (RunnerState, error)
	SaveState(newState RunnerState) error
}

type HistoryStore interface {
	LogTimerInvocations(timers []*Timer) error
}

type MessagingGateway interface {
	GetDispatcherForRunner() (TimerDispatcher, error)
}

type DispatchResult struct {
	update *TimerUpdate
	err    error
}

type TimerDispatcher interface {
	Dispatch(msg TimerMessage, update *TimerUpdate, results chan<- DispatchResult)
	Destroy() error
}
