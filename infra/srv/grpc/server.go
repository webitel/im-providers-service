package grpcsrv

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"time"

	"buf.build/go/protovalidate"
	validatemiddleware "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
	"github.com/webitel/im-delivery-service/config"
	grpcinterceptors "github.com/webitel/im-delivery-service/infra/server/grpc/interceptors"
	"github.com/webitel/im-providers-service/internal/service"
	intrcp "github.com/webitel/webitel-go-kit/pkg/interceptors"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var Module = fx.Module("grpc_server",
	fx.Provide(func(
		conf *config.Config,
		logger *slog.Logger,
		lc fx.Lifecycle,
		auther service.Auther,
	) (*Server, error) {
		srv, err := New(conf.Service.Address, logger, auther)
		if err != nil {
			return nil, err
		}

		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				go func() {
					// [LIFECYCLE] NON-BLOCKING START
					// Run the server in a separate goroutine to allow FX to finish initialization.
					logger.Info(fmt.Sprintf("listen grpc %s:%d", srv.Host(), srv.Port()))
					if err := srv.Listen(); err != nil {
						logger.Error("grpc server error", "err", err)
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				// [GRACEFUL_EXIT] DRAIN SESSIONS
				// Stop accepting new connections and wait for active streams to flush.
				if err := srv.Shutdown(); err != nil {
					logger.Error("error stopping grpc server", "err", err.Error())
					return err
				}
				return nil
			},
		})

		return srv, nil
	}),
)

type Server struct {
	*grpc.Server
	Addr     string
	host     string
	port     int
	log      *slog.Logger
	listener net.Listener
	auther   service.Auther
}

func New(addr string, log *slog.Logger, auther service.Auther) (*Server, error) {
	validator, err := protovalidate.New()
	if err != nil {
		return nil, err
	}

	// [KEEPALIVE_ENFORCEMENT_POLICY] ANTI-DOS PROTECTION
	// Defines strict rules for client-side pings to prevent resource exhaustion.
	kaep := keepalive.EnforcementPolicy{
		// [MIN_PING_INTERVAL] Rate limit for incoming pings. If clients ping faster,
		// the connection will be terminated to prevent CPU spam.
		MinTime: 5 * time.Second,
		// [PERMIT_IDLE_PINGS] Allow pings even if no active RPCs exist.
		// Critical for long-lived streams that might stay idle but must remain alive.
		PermitWithoutStream: true,
	}

	// [KEEPALIVE_SERVER_PARAMETERS] INFRASTRUCTURE_ADAPTABILITY
	// These settings ensure the service remains stable behind L4/L7 proxies (Nginx, HAProxy, AWS ELB).
	kasp := keepalive.ServerParameters{
		// [MAX_CONNECTION_IDLE] ZOMBIE_RECLAMATION
		// If a client (e.g. Flutter app) stays connected but sends no data for 15m,
		// we close the connection to free up file descriptors and memory.
		MaxConnectionIdle: 15 * time.Minute,

		// [MAX_CONNECTION_AGE] LOAD_BALANCER_FRIENDLY_SHAKEOUT
		// Critical for Docker Swarm/Kubernetes/Nginx. By forcing a reconnect every 30m,
		// we ensure that traffic is re-balanced across all horizontal replicas.
		// This prevents "Sticky Sessions" where one container is overloaded while others are idle.
		MaxConnectionAge: 30 * time.Minute,

		// [MAX_CONNECTION_AGE_GRACE] DRAIN_PERIOD
		// When MaxConnectionAge is reached, we send a GOAWAY frame but allow 10s
		// for active delivery streams to finish their current work gracefully.
		MaxConnectionAgeGrace: 10 * time.Second,

		// [TIME_LIVENESS_CHECK] NETWORK_SENSING
		// Server sends an HTTP/2 PING every 20s. This is vital to detect "Half-Open"
		// connections (e.g. user enters a tunnel or loses 4G/5G signal).
		Time: 20 * time.Second,

		// [TIMEOUT_RESPONSE_WINDOW] AGGRESSIVE_CLEANUP
		// If the client doesn't ACK the server's PING within 5s, the TCP connection
		// is terminated. This triggers the Hub's Unsubscribe logic immediately.
		Timeout: 5 * time.Second,
	}

	s := grpc.NewServer(
		// [OBSERVABILITY] TRACING_HANDLER
		// Injects OpenTelemetry hooks for tracing and metrics.
		grpc.StatsHandler(otelgrpc.NewServerHandler()),

		// [PROTOCOL_STABILITY] APPLY_KEEPALIVE
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),

		// [PIPELINE] UNARY_INTERCEPTORS
		// Sequence: Error Handling -> Authentication -> Validation.
		grpc.ChainUnaryInterceptor(
			intrcp.UnaryServerErrorInterceptor(),
			grpcinterceptors.NewUnaryAuthInterceptor(),
			validatemiddleware.UnaryServerInterceptor(validator),
		),
	)

	// [TRANSPORT_BINDING] TCP_SOCKET_INITIALIZATION
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	h, p, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return nil, err
	}
	port, _ := strconv.Atoi(p)

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
		auther:   auther,
	}, nil
}

func (s *Server) Listen() error {
	// [ACCEPT_LOOP] BLOCKING_SERVE
	// Starts the main loop for accepting incoming TCP/HTTP2 connections.
	return s.Serve(s.listener)
}

func (s *Server) Shutdown() error {
	s.log.Debug("initiating graceful shutdown of grpc server")

	// [PHASE 1] TRANSPORT-LEVEL TERMINATION
	// Stop accepting new TCP connections immediately by closing the listener.
	err := s.listener.Close()

	// [PHASE 2] GRACEFUL PROTOCOL EXIT
	// Send HTTP/2 'GOAWAY' frames to all connected clients. This informs them
	// that the server is shutting down, allowing them to reconnect to another
	// replica. It waits for all active RPC handlers to finish their cleanup
	// (including the 'DisconnectedEvent' transmission) before stopping completely.
	s.GracefulStop()

	return err
}

func (s *Server) Host() string {
	if e, ok := os.LookupEnv("PROXY_GRPC_HOST"); ok {
		return e
	}

	return s.host
}

func (s *Server) Port() int {
	return s.port
}

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
			// process IP address
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
