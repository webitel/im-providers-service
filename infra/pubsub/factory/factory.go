package factory

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

// SubscriberFactory defines methods to create a Watermill subscriber
type SubscriberFactory interface {
	BuildSubscriber(name string, config *SubscriberConfig) (message.Subscriber, error)
}

// PublisherFactory defines methods to create a Watermill publisher
type PublisherFactory interface {
	BuildPublisher(config *PublisherConfig) (message.Publisher, error)
}

// Factory combines subscriber and publisher factories
type Factory interface {
	SubscriberFactory
	PublisherFactory
}

// ExchangeConfig holds RabbitMQ exchange-specific settings
type ExchangeConfig struct {
	Name    string
	Type    string
	Durable bool
}

// SubscriberConfig holds full subscription topology details
type SubscriberConfig struct {
	Exchange   ExchangeConfig
	Queue      string
	RoutingKey string

	// [MICROSERVICE_FLAGS] Added for per-node unique queue management
	ExclusiveQueue    bool // Queue used only by one connection
	AutoDeleteQueue   bool // Queue deleted when last consumer disconnects
	DurableQueue      bool // Queue persists after broker restart
	ExclusiveConsumer bool // Consumer has exclusive access to the queue
}

// PublisherConfig holds publication topology details
type PublisherConfig struct {
	Exchange ExchangeConfig
}
