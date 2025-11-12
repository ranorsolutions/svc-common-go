package server

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/ranorsolutions/http-common-go/pkg/log/logger"
	"github.com/ranorsolutions/svc-common-go/pkg/service"
	"github.com/stretchr/testify/assert"
)

func newMockService(t *testing.T) *service.Service {
	log, err := logger.New("test-server", "1.0.0", true)
	assert.NoError(t, err)
	return &service.Service{
		Logger: log,
		Port:   "0", // random port
	}
}

func TestNew_CreatesServer(t *testing.T) {
	svc := newMockService(t)
	s, err := New(svc, "v1")
	assert.NoError(t, err)
	assert.NotNil(t, s)
	assert.NotNil(t, s.Listener)
	s.Listener.Close()
}

func TestNew_NilService(t *testing.T) {
	s, err := New(nil, "v1")
	assert.Error(t, err)
	assert.Nil(t, s)
}

func TestRun_HTTPOnly(t *testing.T) {
	svc := newMockService(t)
	os.Setenv("SERVICE_PROTOCOL", "http")
	defer os.Unsetenv("SERVICE_PROTOCOL")

	s, err := New(svc, "v1")
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err = s.Run(ctx)
	assert.Error(t, err) // closed listener or context canceled
}

func TestRun_GRPCOnly(t *testing.T) {
	svc := newMockService(t)
	os.Setenv("SERVICE_PROTOCOL", "grpc")
	defer os.Unsetenv("SERVICE_PROTOCOL")

	s, err := New(svc, "v1")
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err = s.Run(ctx)
	assert.Error(t, err)
}

func TestRun_MixedProtocol(t *testing.T) {
	svc := newMockService(t)
	os.Unsetenv("SERVICE_PROTOCOL")

	s, err := New(svc, "v1")
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()

	err = s.Run(ctx)
	assert.Error(t, err)
}

func TestShutdownCompletes(t *testing.T) {
	svc := newMockService(t)
	s, _ := New(svc, "v1")
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	s.Listener = l

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := s.Shutdown(ctx)
	assert.NoError(t, err)
}
