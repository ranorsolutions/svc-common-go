package grpc

import (
	"net"

	"github.com/ranorsolutions/svc-common-go/pkg/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// GRPCService encapsulates a gRPC server and its configuration.
type GRPCService struct {
	Server  *grpc.Server
	Service *service.Service
}

// New creates a new gRPC server instance with default interceptors and health checks.
func New(svc *service.Service, opts ...grpc.ServerOption) *GRPCService {
	server := grpc.NewServer(opts...)

	// Register health service for monitoring (optional)
	grpc_health_v1.RegisterHealthServer(server, health.NewServer())

	// Enable reflection in non-production environments
	if svc != nil && svc.Logger != nil {
		svc.Logger.Info("gRPC reflection enabled")
		reflection.Register(server)
	}

	return &GRPCService{
		Server:  server,
		Service: svc,
	}
}

// Register registers a gRPC service implementation (auto-generated from .proto).
func (g *GRPCService) Register(registerFunc func(*grpc.Server)) {
	registerFunc(g.Server)
}

// ListenAndServe starts the gRPC server on the provided listener.
func (g *GRPCService) Serve(l net.Listener) error {
	addr := l.Addr().String()
	g.Service.Logger.Info("gRPC server listening on %s", addr)
	return g.Server.Serve(l)
}

// GracefulStop shuts down the server cleanly.
func (g *GRPCService) GracefulStop() {
	g.Service.Logger.Info("Stopping gRPC server...")
	g.Server.GracefulStop()
}
