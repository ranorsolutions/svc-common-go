package http

import (
	"fmt"
	"net"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	ctxmw "github.com/ranorsolutions/http-common-go/pkg/middleware/context"
	"github.com/ranorsolutions/svc-common-go/pkg/service"
)

type HTTPService struct {
	Engine  *gin.Engine
	Server  *http.Server
	Service *service.Service
}

// New creates a Gin HTTP service wrapping a given `service.Service`.
// It auto-registers all handlers defined in svc.HTTPHandlers and mounts them under /api/{version}.
func New(svc *service.Service, version string) (*HTTPService, error) {
	if svc == nil {
		return nil, fmt.Errorf("service cannot be nil")
	}

	engine := gin.New()
	engine.Use(ctxmw.GinContextToContextMiddleware())
	engine.Use(gin.Recovery())

	group := engine.Group(fmt.Sprintf("/api/%s", version))
	for _, route := range svc.HTTPHandlers {
		switch route.Method {
		case http.MethodGet:
			group.GET(route.Path, route.Handler...)
		case http.MethodPut:
			group.PUT(route.Path, route.Handler...)
		case http.MethodPost:
			group.POST(route.Path, route.Handler...)
		case http.MethodDelete:
			group.DELETE(route.Path, route.Handler...)
		default:
			svc.Logger.Warn("unrecognized HTTP method for route %s", route.Path)
		}
	}

	server := &http.Server{Handler: engine}

	return &HTTPService{
		Server:  server,
		Engine:  engine,
		Service: svc,
	}, nil
}

// ListenAndServe starts serving requests on the given listener.
func (s *HTTPService) ListenAndServe(l net.Listener) error {
	s.Service.Logger.Info("HTTP server listening on %s", formatAddr(l.Addr().String()))
	return s.Server.Serve(l)
}

// formatAddr normalizes the listener address for readable logs.
func formatAddr(addr string) string {
	re := regexp.MustCompile(`\[::\]`)
	return re.ReplaceAllString(addr, "http://localhost")
}
