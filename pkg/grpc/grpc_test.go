package grpc

import (
	"net"
	"testing"
	"time"

	"github.com/ranorsolutions/http-common-go/pkg/log/logger"
	"github.com/ranorsolutions/svc-common-go/pkg/service"
	"github.com/stretchr/testify/assert"
)

func newMockService(t *testing.T) *service.Service {
	log, err := logger.New("test-grpc", "1.0.0", true)
	assert.NoError(t, err)
	return &service.Service{
		Logger: log,
	}
}

func TestNewCreatesServer(t *testing.T) {
	svc := newMockService(t)
	g := New(svc)
	assert.NotNil(t, g.Server)
}

func TestListenAndServe(t *testing.T) {
	svc := newMockService(t)
	g := New(svc)

	l, _ := net.Listen("tcp", "127.0.0.1:0")

	go func() {
		time.Sleep(200 * time.Millisecond)
		g.GracefulStop()
	}()

	err := g.Serve(l)
	assert.NoError(t, err) // ✅ Graceful stop should exit cleanly
}

func TestListenAndServe_ForcedStop(t *testing.T) {
	svc := newMockService(t)
	g := New(svc)

	l, _ := net.Listen("tcp", "127.0.0.1:0")

	go func() {
		time.Sleep(100 * time.Millisecond)
		g.Server.Stop() // abrupt shutdown
	}()

	err := g.Serve(l)
	assert.NoError(t, err) // ✅ It's fine; stop before serve yields nil
}
