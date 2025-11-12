package service

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ranorsolutions/http-common-go/pkg/db/postgres"
	"github.com/ranorsolutions/http-common-go/pkg/log/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// --- Test Setup Helpers ---

func setMinimalEnv(t *testing.T) {
	t.Setenv("SERVICE", "test-service")
	t.Setenv("VERSION", "1.0.0")
	t.Setenv("PORT", "8080")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "user")
	t.Setenv("DB_PASSWORD", "pass")
	t.Setenv("DB_NAME", "testdb")
	t.Setenv("DB_SSL_MODE", "disable")
	t.Setenv("SERVICE_DEPS", "")
}

func clearEnv() {
	for _, v := range []string{
		"SERVICE", "VERSION", "PORT",
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSL_MODE",
		"SERVICE_DEPS",
	} {
		os.Unsetenv(v)
	}
}

// --- Mocks ---

// mockPostgresConnect temporarily replaces postgres.Connect
var originalConnect = postgres.Connect

func mockConnect(_ *postgres.Connection) (*sql.DB, error) {
	return &sql.DB{}, nil
}

func mockConnectFail(_ *postgres.Connection) (*sql.DB, error) {
	return nil, errors.New("db connection failed")
}

// mockGrpcDial temporarily replaces grpc.Dial
var originalGrpcDial = grpc.Dial

func mockGrpcDial(_ string, _ ...grpc.DialOption) (*grpc.ClientConn, error) {
	return &grpc.ClientConn{}, nil
}

func mockGrpcDialFail(_ string, _ ...grpc.DialOption) (*grpc.ClientConn, error) {
	return nil, fmt.Errorf("grpc dial failed")
}

// --- Tests ---

func TestNew_Success(t *testing.T) {
	setMinimalEnv(t)
	connectPostgres = func(_ *postgres.Connection) (*sql.DB, error) {
		return &sql.DB{}, nil
	}

	dialGRPC = func(_ string, _ ...grpc.DialOption) (*grpc.ClientConn, error) {
		return &grpc.ClientConn{}, nil
	}

	svc, err := New()
	assert.NoError(t, err)
	assert.NotNil(t, svc)
	assert.Equal(t, "8080", svc.Port)
	assert.NotNil(t, svc.DB)
	assert.IsType(t, &logger.Logger{}, svc.Logger)
}

func TestNew_DBConnectFailure(t *testing.T) {
	setMinimalEnv(t)

	// swap mocks
	originalConnect := connectPostgres
	connectPostgres = func(_ *postgres.Connection) (*sql.DB, error) {
		return nil, errors.New("db connection failed")
	}
	defer func() { connectPostgres = originalConnect }()

	_, err := New()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create db connection")
}
func TestNew_WithServiceDeps_Success(t *testing.T) {
	setMinimalEnv(t)
	t.Setenv("SERVICE_DEPS", "auth@localhost:5001,users@localhost:5002")

	// Patch before calling New()
	origConnect := connectPostgres
	connectPostgres = func(_ *postgres.Connection) (*sql.DB, error) { return &sql.DB{}, nil }
	defer func() { connectPostgres = origConnect }()

	origDial := dialGRPC
	dialGRPC = func(_ string, _ ...grpc.DialOption) (*grpc.ClientConn, error) {
		// non-nil pointer; do NOT call methods on it
		return new(grpc.ClientConn), nil
	}
	defer func() { dialGRPC = origDial }()

	svc, err := New()
	require.NoError(t, err)
	require.NotNil(t, svc)
	require.NotNil(t, svc.ServiceConnections)

	connAuth, okAuth := svc.ServiceConnections["auth"]
	connUsers, okUsers := svc.ServiceConnections["users"]

	require.True(t, okAuth)
	require.True(t, okUsers)
	require.NotNil(t, connAuth)
	require.NotNil(t, connUsers)

	// Do not call connAuth.GetState() etc. on this stub
}

func TestNew_WithServiceDeps_Failure(t *testing.T) {
	setMinimalEnv(t)
	t.Setenv("SERVICE_DEPS", "auth@localhost:5001")

	origConnect := connectPostgres
	connectPostgres = func(_ *postgres.Connection) (*sql.DB, error) { return &sql.DB{}, nil }
	defer func() { connectPostgres = origConnect }()

	origDial := dialGRPC
	dialGRPC = func(_ string, _ ...grpc.DialOption) (*grpc.ClientConn, error) {
		return nil, fmt.Errorf("mock grpc dial failed")
	}
	defer func() { dialGRPC = origDial }()

	svc, err := New()
	require.Error(t, err)
	assert.Nil(t, svc)
	assert.Contains(t, err.Error(), "mock grpc dial failed")
}

func TestHandleErr_WithMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log, _ := logger.New("test", "1.0", true)
	svc := &Service{Logger: log}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	svc.Logger, _ = logger.New("test", "1.0", true)
	svc.HandleErr(c, errors.New("failure"), "something went wrong", 400)

	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "failure")
	assert.Contains(t, w.Body.String(), "something went wrong")
}

func TestHandleErr_WithoutMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &Service{}
	svc.Logger, _ = logger.New("test", "1.0", true)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	svc.HandleErr(c, errors.New("failure"), "", 404)
	assert.Equal(t, 404, w.Code)
	assert.Contains(t, w.Body.String(), "failure")
}
