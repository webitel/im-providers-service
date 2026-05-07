# Project Context: IM Providers Service

## Tech Stack

| Layer | Technology |
| --- | --- |
| Language | Go (Golang) |
| DI | Uber fx |
| Internal API | gRPC + Protocol Buffers |
| HTTP | `net/http` (webhook receiver) |
| DB | PostgreSQL (`sqlx`) |
| Cache (gates) | In-process LRU (`internal/store/lru`) |
| Cache (users) | Redis |
| Message broker | RabbitMQ (`amqp091-go` + Watermill) |
| Service discovery | Consul (`webitel-go-kit`) |
| Observability | OpenTelemetry (traces, metrics, logs) |
| Logging | `log/slog` (structured, English only) |

## Architecture

```text
HTTP webhook ──► internal/provider/<name>/webhook.go
                        │
gRPC handler ──► internal/handler/grpc/        (transport layer — maps proto ↔ domain)
                        │
               internal/service/               (business logic — orchestrates providers + store)
                        │
               internal/provider/<name>/       (provider adapter — Sender + Receiver)
               internal/store/postgres|lru|redis/
```

**Key rules:**

- Handlers only map proto ↔ domain and call services. No business logic.
- Services hold all orchestration logic.
- Provider adapters implement the `provider.Provider` interface against the official API docs of each platform.
- All new handlers/services/providers register via `fx.Provide` in their respective `module.go`.

## Provider System

### Interfaces (`internal/provider/provider.go`)

```go
type Sender   interface { Type() string; SendText; SendImage; SendDocument }
type Receiver interface { Type() string; HandleWebhook }
type Provider interface { Sender; Receiver }

// Optional — implement only when the platform requires it
type Verifier          interface { Verify(ctx, url.Values) (string, error) }
type SignatureValidator interface { ValidateSignature(ctx, header string, body []byte) error }
```

### Registry (`internal/provider/registry.go`)

All providers register themselves at startup via `fx.Provide` in `internal/provider/module.go`. The `Registry` resolves a `Provider` by `GateType.String()` (e.g. `"facebook"`).

### Adding a new provider — checklist

1. Create `internal/provider/<name>/` with:
   - `provider.go` — implements `provider.Provider` (and optionally `Verifier`, `SignatureValidator`)
   - `client.go` — HTTP client to the platform API (follow official docs only)
   - `webhook.go` — inbound event parsing and routing
   - `outbound.go` — `SendText`, `SendImage`, `SendDocument`
   - `endpoints.go` — URL constants from official docs
   - `module.go` — `fx.Provide(New)`
2. Add `TypeXxx` constant to `internal/domain/model/gate_types.go`.
3. Run `go generate ./internal/domain/model/...` to regenerate `stringer` files.
4. Add `WhatsAppStore` / `<Name>Store` interface to `internal/store/store.go` and implement under `internal/store/postgres/`.
5. Add a DB migration in `migrations/` for the new gate table.
6. Add gRPC handler under `internal/handler/grpc/` and register in `internal/handler/grpc/module.go`.
7. Add service under `internal/service/` and register in `internal/service/module.go`.
8. Register the provider module in `cmd/fx.go`.

## Gate Types (`internal/domain/model/gate_types.go`)

```go
TypeFacebook    // "facebook"    — implemented
TypeWhatsApp    // "whatsapp"    — implemented
TypeInstagram   // "instagram"   — planned
TypeTelegramBot // "telegram_bot"— planned
TypeTelegramApp // "telegram_app"— planned
// Viber                         — planned (type constant not yet added)
```

## Store (`internal/store/store.go`)

```text
Store
 ├── Gates()    GateStore       — paginated list, delete (cross-provider summary view)
 ├── Meta()     MetaAppStore    — Meta App credentials (AppID, AppSecret, VerifyToken)
 ├── Facebook() FacebookStore   — Facebook Page gates
 └── WhatsApp() WhatsAppStore   — WhatsApp Business gates

GateCache        — LRU in-process; key = "<uri>:<providerID>"; avoids DB hits on every webhook
ExternalUserCache— Redis; tracks known external users to avoid duplicate contact creation
```

Sentinel errors: `store.ErrNotFound`, `store.ErrConflict`.

## Webhook Routing

HTTP server listens on `cfg.Service.HTTPAddr`. Webhook path is injected into context via `provider.WebhookURIKey`. Each provider reads this key to identify which `MetaApp` / gate to resolve.

Facebook & WhatsApp webhooks also implement:

- `Verifier` — responds to Meta's `hub.challenge` handshake
- `SignatureValidator` — verifies `X-Hub-Signature-256` on each request

## Meta OAuth Flow

Handled by `internal/service/meta_oauth.go` and `internal/handler/grpc/meta_oauth.go`:

1. Client requests an authorization URL → service builds it using `MetaApp.AppID` + scopes.
2. Meta redirects to `MetaApp.OAuthRedirectURI` with `code`.
3. Service exchanges `code` for a long-lived Page token and stores it encrypted (`pkg/crypto/aes`).

## Sensitive Data Rules

- Never log tokens, secrets, or `AppSecret` — fields tagged `json:"-"` in models.
- AES encryption for stored tokens: `pkg/crypto/aes.go`.
- Auth identity from context: `auth.GetIdentityFromContext(ctx)` → `domain_id`.

## Code Style

- **Comments:** English only. Add a comment only when the WHY is non-obvious (hidden constraint, workaround, subtle invariant). Never comment what the code obviously does.
- **Provider code:** Always implement against the official API documentation of that specific platform. No assumptions, no guessing endpoint behavior.
- **gRPC errors:** Use `google.golang.org/grpc/status` + `codes` (`NotFound`, `Unauthenticated`, `Internal`, `Unimplemented`).
- **Logging:** Structured `slog`. Never log sensitive fields.
- **Not-yet-implemented methods:** Return `status.Error(codes.Unimplemented, "not implemented")`.
