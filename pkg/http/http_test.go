package http

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ranorsolutions/http-common-go/pkg/log/logger"
	"github.com/ranorsolutions/svc-common-go/pkg/route"
	"github.com/ranorsolutions/svc-common-go/pkg/service"
	"github.com/stretchr/testify/assert"
)

func newMockService(t *testing.T) *service.Service {
	log, err := logger.New("test-http", "1.0.0", true)
	assert.NoError(t, err)

	svc := &service.Service{
		Logger: log,
		HTTPHandlers: []*route.Handler{
			{
				Method:  http.MethodGet,
				Path:    "/ping",
				Handler: []gin.HandlerFunc{func(c *gin.Context) { c.JSON(200, gin.H{"pong": true}) }},
			},
		},
	}
	return svc
}

func TestNew_HTTPServiceCreation(t *testing.T) {
	svc := newMockService(t)
	h, err := New(svc, "v1")
	assert.NoError(t, err)
	assert.NotNil(t, h)
	assert.NotNil(t, h.Engine)
	assert.NotNil(t, h.Server)
}

func TestNew_NilService(t *testing.T) {
	h, err := New(nil, "v1")
	assert.Error(t, err)
	assert.Nil(t, h)
}

func TestRoutesAreRegistered(t *testing.T) {
	svc := newMockService(t)
	h, err := New(svc, "v1")
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	rec := httptest.NewRecorder()
	h.Engine.ServeHTTP(rec, req)

	assert.Equal(t, 200, rec.Code)
	assert.JSONEq(t, `{"pong":true}`, rec.Body.String())
}

func TestFormatAddr(t *testing.T) {
	addr := "[::]:4000"
	formatted := formatAddr(addr)
	assert.Equal(t, "http://localhost:4000", formatted)
}

func TestListenAndServeLogsMessage(t *testing.T) {
	svc := newMockService(t)
	h, _ := New(svc, "v1")

	l, _ := net.Listen("tcp", "127.0.0.1:0")
	defer l.Close()

	// We won’t actually serve; just ensure logging call doesn’t panic.
	go func() {
		_ = l.Close() // force exit
	}()
	err := h.ListenAndServe(l)
	assert.Error(t, err) // server closed
}
