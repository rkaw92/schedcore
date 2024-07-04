package main

import (
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron"
	"github.com/rs/zerolog/log"
)

var scheduleParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.DowOptional)

type Timer struct {
	NextAt      time.Time
	TenantId    uuid.UUID
	TimerId     uuid.UUID
	Ushard      int16
	Schedule    string
	Enabled     bool
	Done        bool
	Payload     interface{}
	Destination string
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
