package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

func broadcast(source <-chan time.Time, dest []chan time.Time) {
	for input := range source {
		for _, destChannel := range dest {
			destChannel <- input
		}
	}
}

func run() {
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

	var wallclocksForRunners []chan time.Time
	for i := 0; i < USHARDS_TOTAL; i += 1 {
		wallclockForRunner := make(chan time.Time, 10)
		wallclocksForRunners = append(wallclocksForRunners, wallclockForRunner)
		go runner(int16(i), db, db, db, gateway, wallclockForRunner)
	}
	go broadcast(secondsClock, wallclocksForRunners)

}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// TODO: Replace this with child commands' quit channels (or at least WaitGroups)
	args := os.Args
	switch argv0 := args[0]; filepath.Base(argv0) {
	case "schedcore-runner":
		go run()
	case "schedcore-api":
		panic("Not implemented yet!")
	case "schedcore":
		panic("Not implemented yet! (Implement API first)")
	default:
		panic("Unknown entry point " + argv0)
	}
	neverQuit := make(chan interface{})
	<-neverQuit
}
