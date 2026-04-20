package crypto

import (
	"fmt"

	"github.com/webitel/im-providers-service/config"
	"go.uber.org/fx"
)

// Module provides the encryption logic.
var Module = fx.Module("crypto",
	fx.Provide(
		func(cfg *config.Config) (Encryptor, error) {
			key := cfg.Service.SecretKey
			if len(key) != 32 {
				return nil, fmt.Errorf("crypto: secret key must be exactly 32 bytes, got %d", len(key))
			}
			return NewAESGCM(key), nil
		},
	),
)
