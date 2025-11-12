package server

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/ranorsolutions/svc-common-go/pkg/http"
	"github.com/ranorsolutions/svc-common-go/pkg/service"
	"github.com/soheilhy/cmux"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	grpcsvc "github.com/ranorsolutions/svc-common-go/pkg/grpc"
)

// Server represents a multiplexed gRPC + HTTP server.
type Server struct {
	GRPCServer *grpcsvc.GRPCService
	Listener   net.Listener
	Service    *service.Service
	Version    string
	cancel     context.CancelFunc
}

// New creates a new Server instance that can run gRPC, HTTP, or both.
func New(svc *service.Service, version string, creds ...grpc.ServerOption) (*Server, error) {
	if svc == nil {
		return nil, fmt.Errorf("service cannot be nil")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", svc.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	s := &Server{
		Listener:   listener,
		GRPCServer: grpcsvc.New(svc, creds...),
		Service:    svc,
		Version:    version,
	}

	return s, nil
}

// Run starts serving HTTP and/or gRPC depending on SERVICE_PROTOCOL env var.
func (s *Server) Run(ctx context.Context) error {
	ctx, s.cancel = context.WithCancel(ctx)
	m := cmux.New(s.Listener)
	g, ctx := errgroup.WithContext(ctx)

	protocol := os.Getenv("SERVICE_PROTOCOL")

	// cancel listener on context done
	go func() {
		<-ctx.Done()
		s.Service.Logger.Warn("context canceled, closing listener")
		_ = s.Listener.Close() // unblock cmux and grpc
	}()

	if protocol != "http" {
		grpcListener := m.MatchWithWriters(
			cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"),
			cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc+proto"),
		)
		g.Go(func() error {
			s.Service.Logger.Info("gRPC service available on %s", s.Listener.Addr().String())
			err := s.GRPCServer.Serve(grpcListener)
			s.Service.Logger.Warn("gRPC server stopped: %v", err)
			return err
		})
	}

	if protocol != "grpc" {
		httpListener := m.Match(cmux.HTTP1Fast())
		httpService, err := http.New(s.Service, s.Version)
		if err != nil {
			return fmt.Errorf("failed to initialize HTTP service: %w", err)
		}
		g.Go(func() error {
			s.Service.Logger.Info("HTTP service available on %s", s.Listener.Addr().String())
			err := httpService.ListenAndServe(httpListener)
			s.Service.Logger.Warn("HTTP server stopped: %v", err)
			return err
		})
	}

	// cmux Serve
	g.Go(func() error {
		s.Service.Logger.Info("cmux serving connections")
		err := m.Serve()
		s.Service.Logger.Warn("cmux stopped: %v", err)
		return err
	})

	err := g.Wait()
	if err != nil && ctx.Err() == nil {
		s.Service.Logger.Error("server run terminated: %v", err)
	}
	return err
}

// Shutdown gracefully stops all services.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}
	done := make(chan struct{})
	go func() {
		s.GRPCServer.GracefulStop()
		_ = s.Listener.Close()
		close(done)
	}()
	select {
	case <-done:
		s.Service.Logger.Info("server shutdown complete")
		return nil
	case <-ctx.Done():
		s.Service.Logger.Warn("server shutdown timed out")
		return ctx.Err()
	}
}
