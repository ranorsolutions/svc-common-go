package route

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHandlerRegistration(t *testing.T) {
	r := gin.New()

	// Define a simple route
	h := Handler{
		Method:  http.MethodGet,
		Path:    "/ping",
		Handler: []gin.HandlerFunc{func(c *gin.Context) { c.JSON(200, gin.H{"pong": true}) }},
	}

	// Register dynamically
	switch h.Method {
	case http.MethodGet:
		r.GET(h.Path, h.Handler...)
	default:
		t.Fatalf("unexpected method: %s", h.Method)
	}

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, 200, rec.Code)
	assert.JSONEq(t, `{"pong":true}`, rec.Body.String())
}
