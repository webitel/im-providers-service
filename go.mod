module github.com/webitel/im-providers-service

go 1.26

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260415201107-50325440f8f2.1
	buf.build/go/protovalidate v1.2.0
	github.com/ThreeDotsLabs/watermill v1.5.2
	github.com/ThreeDotsLabs/watermill-amqp/v3 v3.1.0
	github.com/exaring/otelpgx v0.11.1
	github.com/fsnotify/fsnotify v1.10.1
	github.com/georgysavva/scany/v2 v2.1.4
	github.com/go-chi/chi/v5 v5.2.5
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.3
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/jackc/pgx/v5 v5.9.2
	github.com/rabbitmq/amqp091-go v1.11.0
	github.com/redis/go-redis/v9 v9.19.0
	github.com/spf13/pflag v1.0.10
	github.com/spf13/viper v1.21.0
	github.com/testcontainers/testcontainers-go v0.42.0
	github.com/testcontainers/testcontainers-go/modules/postgres v0.42.0
	github.com/urfave/cli/v2 v2.27.7
<<<<<<< HEAD
	github.com/webitel/storage v0.0.0-20260429142015-53a76b96eca1
	github.com/webitel/webitel-go-kit/infra/discovery v0.0.0-20260202123728-94059b0e19be
	github.com/webitel/webitel-go-kit/infra/otel v0.0.0-20251222125635-d60448d23a82
	github.com/webitel/webitel-go-kit/infra/transport v0.0.0-20260202123728-94059b0e19be
	github.com/webitel/webitel-go-kit/pkg/interceptors v0.0.0-20260202123728-94059b0e19be
	go.opentelemetry.io/contrib/bridges/otelslog v0.14.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.65.0
	go.opentelemetry.io/otel/sdk v1.40.0
=======
	github.com/webitel/storage v0.0.0-20260519164113-3976e11bfdc3
	github.com/webitel/webitel-go-kit/infra/discovery v0.0.0-20260520154553-61b75ea56e1e
	github.com/webitel/webitel-go-kit/infra/otel v0.1.0
	github.com/webitel/webitel-go-kit/infra/transport v0.0.0-20260520154553-61b75ea56e1e
	github.com/webitel/webitel-go-kit/pkg/errors v0.1.0
	github.com/webitel/webitel-go-kit/pkg/interceptors v0.1.1
	github.com/webitel/wlog v0.0.0-20250325101442-de4f125c1ec7
	go.opentelemetry.io/contrib/bridges/otelslog v0.18.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.68.0
	go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/sdk v1.43.0
>>>>>>> 57a13b6 (fix: rebase `main` into `whatsapp` branch)
	go.uber.org/fx v1.24.0
	golang.org/x/sync v0.20.0
	google.golang.org/genproto/googleapis/api v0.0.0-20260519071638-aa98bba5eb94
	google.golang.org/grpc v1.81.1
	google.golang.org/protobuf v1.36.11
)

require (
<<<<<<< HEAD
=======
	cel.dev/expr v0.25.1 // indirect
	dario.cat/mergo v1.0.2 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
>>>>>>> 57a13b6 (fix: rebase `main` into `whatsapp` branch)
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
<<<<<<< HEAD
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/hashicorp/consul/api v1.33.2 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.12.2 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.12.2 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.39.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.36.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.12.2 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.40.0 // indirect
	go.opentelemetry.io/proto/otlp v1.9.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
)

require (
	cel.dev/expr v0.25.1 // indirect
	github.com/ThreeDotsLabs/watermill v1.5.1
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/exaring/otelpgx v0.10.0
	github.com/fatih/color v1.18.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/google/cel-go v0.27.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-metrics v0.5.4 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/hashicorp/serf v0.10.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
=======
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/cpuguy83/dockercfg v0.3.2 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.6.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/ebitengine/purego v0.10.0 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/google/cel-go v0.28.0 // indirect
	github.com/hashicorp/consul/api v1.33.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.5.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/magiconair/properties v1.8.10 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/go-archive v0.2.0 // indirect
	github.com/moby/moby/api v1.54.1 // indirect
	github.com/moby/moby/client v0.4.0 // indirect
	github.com/moby/patternmatcher v0.6.1 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/sys/user v0.4.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
>>>>>>> 57a13b6 (fix: rebase `main` into `whatsapp` branch)
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/shirou/gopsutil/v4 v4.26.3 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tklauser/go-sysconf v0.3.16 // indirect
	github.com/tklauser/numcpus v0.11.0 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
<<<<<<< HEAD
	go.opentelemetry.io/otel v1.41.0
	go.opentelemetry.io/otel/log v0.15.0 // indirect
	go.opentelemetry.io/otel/metric v1.41.0 // indirect
	go.opentelemetry.io/otel/trace v1.41.0 // indirect
=======
	go.opentelemetry.io/contrib/bridges/otelzap v0.11.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.19.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.19.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.43.0 // indirect
	go.opentelemetry.io/otel/log v0.19.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.19.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
>>>>>>> 57a13b6 (fix: rebase `main` into `whatsapp` branch)
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/exp v0.0.0-20251209150349-8475f28825e9 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260511170946-3700d4141b60 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
