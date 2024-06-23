package main

import (
	"errors"
	"strings"

	json "github.com/goccy/go-json"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitGateway struct {
	Connection *amqp.Connection
}

func NewRabbitGateway(url string) (*RabbitGateway, error) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return nil, err
	}
	return &RabbitGateway{conn}, nil
}

func (gateway *RabbitGateway) GetDispatcherForRunner() (TimerDispatcher, error) {
	newChan, err := gateway.Connection.Channel()
	if err != nil {
		return nil, err
	}
	err = newChan.Confirm(false)
	if err != nil {
		newChan.Close()
		return nil, err
	}
	return &RabbitDispatcher{newChan}, nil
}

type RabbitDispatcher struct {
	Channel *amqp.Channel
}

func (dispatcher *RabbitDispatcher) Dispatch(
	timer TimerMessage,
	update *TimerUpdate,
	results chan<- DispatchResult,
) {
	exchange, routingKey, sepFound := strings.Cut(timer.Destination, "/")
	if !sepFound {
		exchange = ""
		routingKey = timer.Destination
	}
	body, err := json.Marshal(timer)
	if err != nil {
		results <- DispatchResult{nil, err}
		return
	}
	deferred, err := dispatcher.Channel.PublishWithDeferredConfirm(
		exchange,
		routingKey,
		true,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		results <- DispatchResult{nil, err}
		return
	}
	published := deferred.Wait()
	if published {
		results <- DispatchResult{update, nil}
	} else {
		results <- DispatchResult{nil, errors.New("AMQP broker rejected message")}
	}
}

func (dispatcher *RabbitDispatcher) Destroy() error {
	return dispatcher.Channel.Close()
}
