package main

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/alexdrl/zerowater"
	json "github.com/goccy/go-json"
	"github.com/rs/zerolog/log"
)

const PUBLISH_TIMEOUT = time.Second * 10

type RabbitGateway struct {
	config amqp.Config
}

func NewRabbitGateway(url url.URL) (*RabbitGateway, error) {
	config := amqp.Config{
		Connection: amqp.ConnectionConfig{
			AmqpURI:   url.String(),
			Reconnect: amqp.DefaultReconnectConfig(),
		},
		Marshaler: amqp.DefaultMarshaler{},
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
	return &RabbitGateway{config}, nil
}

func (gateway *RabbitGateway) GetDispatcherForRunner() (TimerDispatcher, error) {
	pub, err := amqp.NewPublisher(gateway.config, zerowater.NewZerologLoggerAdapter(
		log.Logger.With().Str("component", "RabbitGateway").Logger(),
	))
	if err != nil {
		return nil, err
	}
	return &RabbitDispatcher{pub}, nil
}

type RabbitDispatcher struct {
	pub *amqp.Publisher
}

func (dispatcher *RabbitDispatcher) Dispatch(
	msg TimerMessage,
	update *TimerUpdate,
	results chan<- DispatchResult,
) {
	body, err := json.Marshal(msg)
	if err != nil {
		results <- DispatchResult{nil, err}
		return
	}
	lowLevelMsg := message.NewMessage(
		msg.InvocationId.String(),
		body,
	)
	timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), PUBLISH_TIMEOUT)
	defer cancelTimeout()
	lowLevelMsg.SetContext(timeoutCtx)
	err = dispatcher.pub.Publish(msg.Destination, lowLevelMsg)
	if err != nil {
		results <- DispatchResult{nil, err}
	} else {
		results <- DispatchResult{update, nil}
	}
}

func (dispatcher *RabbitDispatcher) Destroy() error {
	return dispatcher.pub.Close()
}
