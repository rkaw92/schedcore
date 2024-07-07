package main

import (
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron"
	"github.com/rs/zerolog/log"
)

var scheduleParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.DowOptional)

type Timer struct {
	NextAt           time.Time
	TenantId         uuid.UUID
	TimerId          uuid.UUID
	Ushard           int16
	Schedule         string
	Done             bool
	Payload          interface{}
	Destination      string
	NextInvocationId uuid.UUID
}

func (timer *Timer) Next() time.Time {
	if timer.Schedule == "" {
		return timer.NextAt
	}
	schedule, err := scheduleParser.Parse(timer.Schedule)
	if err != nil {
		log.Error().Str("schedule", timer.Schedule).Msg("malformed schedule string, cannot parse")
		return timer.NextAt
	}
	return schedule.Next(timer.NextAt)
}

func GenInvocationId() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		log.Error().Err(err).Msg("cannot generate UUIDv7")
		return uuid.Nil
	}
	return id
}
