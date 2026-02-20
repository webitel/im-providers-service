package amqp

import (
	"context"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/webitel/im-providers-service/internal/adapter/pubsub"
	pubsubadapter "github.com/webitel/im-providers-service/internal/adapter/pubsub"
	"go.uber.org/fx"
)

var Module = fx.Module("amqp",
	fx.Provide(
		NewMessageHandler,
		pubsubadapter.NewSubscriberProvider,
		func(l *slog.Logger) (*message.Router, error) {
			return message.NewRouter(message.RouterConfig{}, watermill.NewSlogLogger(l))
		},
	),
	// [FIX] Add subProvider here to trigger ProvidePubSub
	fx.Invoke(func(
		lc fx.Lifecycle,
		h *MessageHandler,
		r *message.Router,
		subProvider *pubsub.SubscriberProvider,
		logger *slog.Logger,
	) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				// [CRITICAL] You must call this to trigger the chain!
				if err := h.RegisterHandlers(r, subProvider); err != nil {
					return err
				}

				go func() {
					if err := r.Run(context.Background()); err != nil {
						logger.Error("router failed", "err", err)
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				return r.Close()
			},
		})
	}),
)
