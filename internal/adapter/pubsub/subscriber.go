package pubsub

import (
	"github.com/ThreeDotsLabs/watermill/message"
	infrapubsub "github.com/webitel/im-providers-service/infra/pubsub"
	"github.com/webitel/im-providers-service/infra/pubsub/factory"
)

type SubscriberProvider struct {
	factory factory.Factory
}

func NewSubscriberProvider(p infrapubsub.Provider) *SubscriberProvider {
	return &SubscriberProvider{factory: p.GetFactory()}
}

// Build creates a subscriber with a temporary unique queue for fan-out messaging
func (sp *SubscriberProvider) Build(queue, exchange, routingKey string) (message.Subscriber, error) {
	// [STRATEGY] We use Exclusive + AutoDelete for per-node unique queues.
	// This allows every instance of delivery-service to receive a copy of the message.
	return sp.factory.BuildSubscriber("im-providers-service", &factory.SubscriberConfig{
		Exchange: factory.ExchangeConfig{
			Name:    exchange,
			Type:    "topic",
			Durable: true,
		},
		Queue:      queue,
		RoutingKey: routingKey,

		// [NODE_SPECIFIC_SETTINGS]
		DurableQueue:      false, // Temporary queue, do not persist on broker restart
		AutoDeleteQueue:   true,  // Delete queue automatically when node disconnects
		ExclusiveQueue:    true,  // Ensure no other node attaches to this specific queue
		ExclusiveConsumer: true,  // Single consumer per channel
	})
}
