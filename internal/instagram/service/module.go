package service

import (
	"go.uber.org/fx"

	fbstore "github.com/webitel/im-providers-service/internal/facebook/store"
	igstore "github.com/webitel/im-providers-service/internal/instagram/store"
	"github.com/webitel/im-providers-service/pkg/crypto"
)

var Module = fx.Module("instagram/service",
	fx.Provide(
		fx.Annotate(
			NewInstagramOAuthService,
			fx.As(new(InstagramOAuthManager)),
		),
		fx.Annotate(
			NewInstagramService,
			fx.As(new(InstagramManager)),
		),
	),
)

// ServiceParams are the injectable dependencies for the Instagram service module.
type ServiceParams struct {
	fx.In

	MetaAppStore   fbstore.MetaAppStore
	InstagramStore igstore.InstagramStore
	Encryptor      crypto.Encryptor
}
