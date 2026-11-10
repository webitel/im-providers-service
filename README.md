# im-providers-service
Connector layer between the core im-system and external chat platforms (e.g., Telegram, Viber, Meta, etc.).

## Secrets

Secrets do not have to be stored in plain text in `/etc/default/webitel-im-providers`.
The service resolves them through
[webitel-go-kit appconfig](https://github.com/webitel/webitel-go-kit/blob/main/appconfig/secret/README.md), so each of these keys can also come
from a systemd credential (encrypted at rest) or a file:

| Config key | Service gets | systemd credential | File env var |
|---|---|---|---|
| `postgres.dsn` | contents | `postgres.dsn` | `POSTGRES_DSN_FILE` |
| `redis.password` | contents | `redis.password` | `REDIS_PASSWORD_FILE` |
| `service.secret_key` | contents | `service.secret_key` | `SERVICE_SECRET_KEY_FILE` |
| `service.conn.key` | path of the key file | `service.conn.key` | `SERVICE_CONN_KEY_FILE` |
| `service.conn.client.key` | path of the key file | `service.conn.client.key` | `SERVICE_CONN_CLIENT_KEY_FILE` |

A key set in plain text **and** present as a credential or `_FILE` makes the
service refuse to start, so remove the old value when you move a secret.

`postgres.replicas` is a list and is not resolved from credentials yet; replica
DSNs stay in the env file for now.

Example: move `postgres.dsn` and `service.secret_key` into encrypted credentials (systemd ≥ 250).

```sh
install -d -m 0700 /etc/credstore.encrypted
for name in postgres.dsn service.secret_key; do
  systemd-ask-password -n "$name:" \
    | systemd-creds encrypt --name="$name" - "/etc/credstore.encrypted/webitel-im-providers.$name"
done
```

```ini
# systemctl edit webitel-im-providers
[Service]
LoadCredentialEncrypted=postgres.dsn:/etc/credstore.encrypted/webitel-im-providers.postgres.dsn
LoadCredentialEncrypted=service.secret_key:/etc/credstore.encrypted/webitel-im-providers.service.secret_key
```

Then delete the matching lines from `/etc/default/webitel-im-providers` and run
`systemctl restart webitel-im-providers`. The unit's sandboxing needs no changes.

`webitel-im-providers migrate` started by hand does not get the unit's credentials. Run it
through systemd so it does:

```sh
systemd-run --pty --uid=webitel-svc \
  -p LoadCredentialEncrypted=postgres.dsn:/etc/credstore.encrypted/webitel-im-providers.postgres.dsn \
  /usr/local/bin/webitel-im-providers migrate
```

In Docker or Kubernetes mount the secret as a file and point the `_FILE` env
var at it, e.g. `POSTGRES_DSN_FILE=/run/secrets/postgres.dsn`. Rules, older systemd
and security limits: [secret/README.md](https://github.com/webitel/webitel-go-kit/blob/main/appconfig/secret/README.md).
