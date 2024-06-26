package main

import "time"

// Translate a channel of Time instants to fall on full seconds.
func seconds(input <-chan time.Time, output chan<- time.Time) {
	for wallTime := range input {
		wallTimeRounded := wallTime.Truncate(time.Second)
		output <- wallTimeRounded
		// TODO: Monitor clock lateness - if the ticker's time is not the same as realtime, we're running late!
	}
}

// Produces a channel that emits virtual-time ticks, where time is accelerated as needed
// to match the wall-time ticks. This behaves exactly the same, except when late.
func virtclock(walltime <-chan time.Time, startAt time.Time, output chan<- time.Time) {
	virtualNow := startAt.Truncate(time.Second)
	for wallNow := range walltime {
		// Every time we get a tick from "upstream", fast-forward to it.
		for virtualNow.Before(wallNow) {
			output <- virtualNow
			virtualNow = virtualNow.Add(time.Second)
		}
		output <- virtualNow
	}
}
