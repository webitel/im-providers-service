package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"

	"buf.build/go/protovalidate"
	grpcdefaultinterceptors "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	validatemiddleware "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	intrcp "github.com/webitel/webitel-go-kit/pkg/interceptors"

	"github.com/webitel/im-providers-service/config"
	"github.com/webitel/im-providers-service/infra/auth"
	"github.com/webitel/im-providers-service/infra/srv/grpc/interceptors"
	infratls "github.com/webitel/im-providers-service/infra/tls"
)

var Module = fx.Module("grpc_server",
	fx.Provide(
		fx.Annotate(
			ProvideServer,
		),
	),
)

// ProvideServer is the fx constructor for the gRPC server.
// It injects necessary dependencies like Auther and Contacter for the Auth Interceptor.
func ProvideServer(
	conf *config.Config,
	logger *slog.Logger,
	tlsConf *infratls.Config,
	auther auth.Authorizer,
	lc fx.Lifecycle,
) (*Server, error) {
	srv, err := New(conf.Service.GRPCAddr, func(c *Config) error {
		c.TLS = tlsConf.Server.Clone()
		c.Logger = logger
		c.Auther = auther

		return nil
	})
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				logger.Info(fmt.Sprintf("listen grpc %s:%d", srv.Host(), srv.Port()))
				if err := srv.Listen(); err != nil {
					logger.Error("grpc server error", "err", err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := srv.Shutdown(); err != nil {
				logger.Error("error stopping grpc server", "err", err.Error())

				return err
			}

			return nil
		},
	})

	return srv, nil
}

type Server struct {
	*grpc.Server
	Addr     string
	host     string
	port     int
	log      *slog.Logger
	listener net.Listener
}

type Config struct {
	// Settings
	TLS *tls.Config

	// Dependencies
	Logger *slog.Logger
	Auther auth.Authorizer
}

type Option func(*Config) error

// New initializes a new gRPC server with the provided options.
func New(addr string, opts ...Option) (*Server, error) {
	var conf Config

	// Apply options to configuration
	for _, opt := range opts {
		if err := opt(&conf); err != nil {
			return nil, err
		}
	}

	// Setup logger
	log := conf.Logger
	if log == nil {
		log = slog.Default()
	}

	// Default address if empty
	if addr == "" {
		addr = ":0"
	}

	// Initialize proto-validator
	validator, err := protovalidate.New()
	if err != nil {
		return nil, err
	}

	serverOpts := []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			intrcp.UnaryServerErrorInterceptor(),
			// Injected Auth Interceptor with required business logic clients
			selector.UnaryServerInterceptor(interceptors.NewUnaryAuthInterceptor(conf.Auther),
				selector.MatchFunc(func(ctx context.Context, callMeta grpcdefaultinterceptors.CallMeta) bool {
					method := fmt.Sprintf("%s/%s", callMeta.Service, callMeta.Method)
					return method != "webitel.im.api.gateway.v1.Account/Token"
				})),
			validatemiddleware.UnaryServerInterceptor(validator),
		),
	}

	// Configure TLS if provided
	if conf.TLS != nil {
		serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(conf.TLS)))
	}

	// Initialize gRPC server with interceptor chain
	s := grpc.NewServer(serverOpts...)

	// Start TCP listener
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	// Resolve actual host and port
	h, p, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return nil, err
	}
	port, _ := strconv.Atoi(p)

	// Fallback for IPv6 wildcard
	if h == "::" {
		h = publicAddr()
	}

	return &Server{
		Addr:     addr,
		Server:   s,
		log:      log,
		host:     h,
		port:     port,
		listener: l,
	}, nil
}

// Listen starts the gRPC server.
func (s *Server) Listen() error {
	return s.Serve(s.listener)
}

// Shutdown gracefully stops the gRPC server and closes the listener.
func (s *Server) Shutdown() error {
	s.log.Debug("receiving shutdown signal for grpc server")
	err := s.listener.Close()
	s.Server.GracefulStop()

	return err
}

// Host returns the public host address.
func (s *Server) Host() string {
	if e, ok := os.LookupEnv("PROXY_GRPC_HOST"); ok {
		return e
	}

	return s.host
}

// Port returns the server listening port.
func (s *Server) Port() int {
	return s.port
}

// --- Network Helpers ---

func publicAddr() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, i := range interfaces {
		addresses, err := i.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addresses {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			default:
				continue
			}

			if isPublicIP(ip) {
				return ip.String()
			}
		}
	}

	return ""
}

func isPublicIP(IP net.IP) bool {
	if IP.IsLoopback() || IP.IsLinkLocalMulticast() || IP.IsLinkLocalUnicast() {
		return false
	}

	return true
}
