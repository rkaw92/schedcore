package main

import (
	"strings"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/alexdrl/zerowater"
	json "github.com/goccy/go-json"
	"github.com/rs/zerolog/log"
)

type RabbitGateway struct {
	publisher *amqp.Publisher
}

func NewRabbitGateway(url string) (*RabbitGateway, error) {
	config := amqp.Config{
		Connection: amqp.ConnectionConfig{AmqpURI: url},
		Marshaler:  amqp.DefaultMarshaler{},
		Exchange: amqp.ExchangeConfig{
			GenerateName: func(topic string) string {
				return strings.Split(topic, "/")[0]
			},
			Type:    "topic",
			Durable: true,
		},
		Publish: amqp.PublishConfig{
			GenerateRoutingKey: func(topic string) string {
				return strings.Join(strings.Split(topic, "/")[1:], "/")
			},
			ConfirmDelivery: true,
			ChannelPoolSize: 16,
		},
		TopologyBuilder: &amqp.DefaultTopologyBuilder{},
	}
	pub, err := amqp.NewPublisher(config, zerowater.NewZerologLoggerAdapter(
		log.Logger.With().Str("component", "RabbitGateway").Logger(),
	))
	if err != nil {
		return nil, err
	}
	return &RabbitGateway{
		publisher: pub,
	}, nil
}

func (gateway *RabbitGateway) GetDispatcherForRunner() (TimerDispatcher, error) {
	return &RabbitDispatcher{
		gateway,
	}, nil
}

type RabbitDispatcher struct {
	gateway *RabbitGateway
}

func (dispatcher *RabbitDispatcher) Dispatch(
	timer TimerMessage,
	update *TimerUpdate,
	results chan<- DispatchResult,
) {
	body, err := json.Marshal(timer)
	if err != nil {
		results <- DispatchResult{nil, err}
		return
	}
	// TODO: Deterministic invocation ID in body
	err = dispatcher.gateway.publisher.Publish(timer.Destination, message.NewMessage(
		// TODO: Deterministic message UUID
		watermill.NewUUID(),
		body,
	))
	if err != nil {
		results <- DispatchResult{nil, err}
	} else {
		results <- DispatchResult{update, nil}
	}
}

func (dispatcher *RabbitDispatcher) Destroy() error {
	// TODO: Figure out why this doesn't quit the process
	return dispatcher.gateway.publisher.Close()
}
