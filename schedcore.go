package main

import (
	"time"
)

const NUM_WORKERS int16 = 64

func broadcast(source <-chan time.Time, dest [NUM_WORKERS]chan time.Time) {
	for input := range source {
		for _, destChannel := range dest {
			destChannel <- input
		}
	}
}

func main() {
	db, err := NewScyllaStore([]string{"192.168.79.155"}, "timerapp")
	if err != nil {
		panic(err)
	}
	wallclockTicker := time.NewTicker(time.Second)
	secondsClock := make(chan time.Time, 1)
	go seconds(wallclockTicker.C, secondsClock)

	gateway, err := NewRabbitGateway("amqp://guest:guest@127.0.0.1/")
	if err != nil {
		panic(err)
	}

	var wallclocksForRunners [NUM_WORKERS]chan time.Time
	for i := range wallclocksForRunners {
		wallclocksForRunners[i] = make(chan time.Time, 10)
		go runner(int16(i), db, db, db, gateway, wallclocksForRunners[i])
	}
	go broadcast(secondsClock, wallclocksForRunners)
	neverQuit := make(chan interface{})
	<-neverQuit
}
