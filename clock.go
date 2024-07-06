package main

import "time"

type CustomTicker struct {
	realTicker     *time.Ticker
	destroyChannel chan struct{}
	Ticks          chan time.Time
}

func NewCustomTicker() *CustomTicker {
	c := &CustomTicker{
		realTicker:     time.NewTicker(time.Second),
		destroyChannel: make(chan struct{}),
		Ticks:          make(chan time.Time, 1),
	}
	go func() {
		end := false
		for !end {
			select {
			case now := <-c.realTicker.C:
				c.Ticks <- now
			case <-c.destroyChannel:
				end = true
			}
		}
		close(c.Ticks)
	}()
	return c
}

func (c *CustomTicker) Destroy() {
	c.destroyChannel <- struct{}{}
}

// Translate a channel of Time instants to fall on full seconds.
func seconds(input <-chan time.Time, output chan<- time.Time) {
	for wallTime := range input {
		wallTimeRounded := wallTime.Truncate(time.Second)
		output <- wallTimeRounded
	}
	close(output)
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
	close(output)
}
