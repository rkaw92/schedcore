package main

import (
	"time"

	"github.com/google/uuid"
)

type Timer struct {
	NextAt   time.Time
	TimerId  uuid.UUID
	TenantId uuid.UUID
	Ushard   int16
	Schedule string
	Enabled  bool
	Done     bool
}

type RunnerState struct {
	MyUshard int16
	Next     time.Time
}

type TimerUpdate struct {
	TenantId  uuid.UUID
	TimerId   uuid.UUID
	Ushard    int16
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
	GetState(MyUshard int16) (RunnerState, error)
}
