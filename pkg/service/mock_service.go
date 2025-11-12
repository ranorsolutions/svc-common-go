package service

import (
	"database/sql"

	"github.com/ranorsolutions/http-common-go/pkg/log/logger"
	"google.golang.org/grpc"
)

// NewMock creates a lightweight mock Service for testing without DB or gRPC.
func NewMock() *Service {
	// Build a simple logger that prints to stderr.
	log, _ := logger.New("mock-service", "test", true)

	return &Service{
		DB:                 &sql.DB{}, // not actually connected
		ServiceConnections: map[string]*grpc.ClientConn{},
		Services:           map[string]any{},
		Logger:             log,
		Port:               "0",
	}
}
