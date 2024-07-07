package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	_ "github.com/joho/godotenv/autoload"
)

func broadcast(source <-chan interface{}, dest []chan interface{}) {
	for input := range source {
		for _, destChannel := range dest {
			destChannel <- input
		}
	}
	for _, destChannel := range dest {
		close(destChannel)
	}
}

func supervisor(
	ushard int16,
	timerDb TimerStoreForRunner,
	runnerDb RunnerStore,
	historyDb HistoryStore,
	gateway MessagingGateway,
	quitRequest <-chan interface{},
	wg *sync.WaitGroup,
) {
	wallclockTicker := NewCustomTicker()
	secondsClock := make(chan time.Time, 1)
	go seconds(wallclockTicker.Ticks, secondsClock)

	quitRequested := false

	for !quitRequested {
		runnerQuit := make(chan error)
		go runner(ushard, timerDb, runnerDb, historyDb, gateway, secondsClock, runnerQuit)
		isRunnerExited := false
		for !isRunnerExited {
			select {
			case runnerError := <-runnerQuit:
				isRunnerExited = true
				if runnerError != nil {
					log.Error().Int16("ushard", ushard).Err(runnerError).Msg("runner quit unexpectedly")
					<-time.After(time.Second * 5)
				} else {
					log.Info().Int16("ushard", ushard).Msg("runner stopped")
				}
			case <-quitRequest:
				quitRequested = true
				wallclockTicker.Destroy()
			}
		}
	}
	wg.Done()
}

func run(
	timerDb TimerStoreForRunner,
	runnerDb RunnerStore,
	historyDb HistoryStore,
	config Config,
) {
	gateway, err := NewRabbitGateway(config.BROKER_URL)
	if err != nil {
		panic(err)
	}

	wg := &sync.WaitGroup{}
	globalQuit := make(chan interface{})
	quitChannels := make([]chan interface{}, 0, USHARDS_TOTAL)

	for i := uint16(0); i < USHARDS_TOTAL; i += 1 {
		wg.Add(1)
		quit := make(chan interface{}, 1)
		quitChannels = append(quitChannels, quit)
		go supervisor(int16(i), timerDb, runnerDb, historyDb, gateway, quit, wg)
	}
	go broadcast(globalQuit, quitChannels)

	go func() {
		// TODO: Move to main?
		endSignal := make(chan os.Signal, 1)
		signal.Notify(endSignal, syscall.SIGINT, syscall.SIGTERM)
		gotSignal := <-endSignal
		log.Info().Str("signal", gotSignal.String()).Msg("preparing to terminate, will catch up to real time first - send the signal again to skip")
		globalQuit <- nil
		// If we get another signal, it means the user is impatient and we should terminate right now.
		<-endSignal
		os.Exit(0)
	}()

	wg.Wait()
}

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }

	config := NewConfigFromEnv()

	db, err := NewScyllaStore(config.DB_URL)
	if err != nil {
		panic(err)
	}

	args := os.Args
	switch argv0 := args[0]; filepath.Base(argv0) {
	case "schedcore-runner":
		run(db, db, db, config)
	case "schedcore-api":
		runAPI(db)
	case "schedcore":
		panic("Not implemented yet! (Implement API first)")
	default:
		panic("Unknown entry point " + argv0)
	}
}
