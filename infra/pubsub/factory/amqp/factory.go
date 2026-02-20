package amqp

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/webitel/im-providers-service/infra/pubsub/factory"
)

type Factory struct {
	url    string
	logger watermill.LoggerAdapter
}

func NewFactory(url string, logger watermill.LoggerAdapter) (*Factory, error) {
	return &Factory{
		url:    url,
		logger: logger,
	}, nil
}

// BuildSubscriber creates a new AMQP subscriber with specific queue topology
// infra/pubsub/factory/amqp/factory.go

func (f *Factory) BuildSubscriber(name string, subConfig *factory.SubscriberConfig) (message.Subscriber, error) {
	if subConfig == nil {
		return nil, fmt.Errorf("no subscriber configured")
	}

	conf := amqp.Config{
		Connection: amqp.ConnectionConfig{
			AmqpURI: f.url,
		},
		Marshaler: amqp.DefaultMarshaler{},
		Exchange: amqp.ExchangeConfig{
			GenerateName: func(s string) string {
				return subConfig.Exchange.Name
			},
			Type:    subConfig.Exchange.Type,
			Durable: subConfig.Exchange.Durable,
		},
		Queue: amqp.QueueConfig{
			GenerateName: func(s string) string {
				return subConfig.Queue
			},
			Durable:    subConfig.DurableQueue,
			AutoDelete: subConfig.AutoDeleteQueue,
			Exclusive:  subConfig.ExclusiveQueue,
		},
		QueueBind: amqp.QueueBindConfig{
			GenerateRoutingKey: func(s string) string {
				return subConfig.RoutingKey
			},
		},
		Consume: amqp.ConsumeConfig{
			Consumer:  name,
			Exclusive: subConfig.ExclusiveConsumer,
		},
		Publish: amqp.PublishConfig{
			GenerateRoutingKey: func(s string) string {
				return s
			},
		},
		TopologyBuilder: &amqp.DefaultTopologyBuilder{},
	}

	return amqp.NewSubscriber(conf, f.logger)
}

// BuildPublisher creates a new AMQP publisher
func (f *Factory) BuildPublisher(pubConfig *factory.PublisherConfig) (message.Publisher, error) {
	conf := amqp.Config{
		Connection: amqp.ConnectionConfig{
			AmqpURI: f.url,
		},
		Marshaler: amqp.DefaultMarshaler{},
		Exchange: amqp.ExchangeConfig{
			GenerateName: func(s string) string {
				return pubConfig.Exchange.Name
			},
			Type:    pubConfig.Exchange.Type,
			Durable: pubConfig.Exchange.Durable,
		},
		TopologyBuilder: &amqp.DefaultTopologyBuilder{},
		Publish: amqp.PublishConfig{
			GenerateRoutingKey: func(s string) string {
				return s
			},
		},
	}
	return amqp.NewPublisher(conf, f.logger)
}
