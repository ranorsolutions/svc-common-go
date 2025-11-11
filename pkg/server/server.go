package server

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/ranorsolutions/svc-common-go/pkg/http"
	"github.com/ranorsolutions/svc-common-go/pkg/service"
	"github.com/soheilhy/cmux"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type Server struct {
	GRPCServer *grpc.Server
	Listener   net.Listener
	Service    *service.Service
	Version    string
}

func New(svc *service.Service, version string, creds ...grpc.ServerOption) *Server {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", svc.Port))
	if err != nil {
		log.Fatal("fatal error creating net listener", err)
	}

	srv := &Server{
		Listener:   listener,
		GRPCServer: grpc.NewServer(creds...),
		Service:    svc,
		Version:    version,
	}

	return srv
}

func (s *Server) Run() {
	m := cmux.New(s.Listener)

	g := new(errgroup.Group)
	if os.Getenv("SERVICE_PROTOCOL") != "http" {
		grpcListener := m.MatchWithWriters(
			cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"),
			cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc+proto"),
		)
		g.Go(func() error {
			s.Service.Logger.Info(fmt.Sprintf("grpc service available: %s", s.Listener.Addr().String()))
			return s.GRPCServer.Serve(grpcListener)
		})
	}

	if os.Getenv("SERVICE_PROTOCOL") != "grpc" {
		httpListener := m.Match(cmux.HTTP1Fast())
		httpService, err := http.New(s.Service, s.Version)
		if err != nil {
			log.Fatal("fatal error creating http service", err)
		}

		g.Go(func() error { return httpService.ListenAndServe(httpListener) })
		g.Go(func() error { return m.Serve() })
	}

	s.Service.Logger.Info("run server:", g.Wait())
}
