package http

import (
	"fmt"
	"net"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/ranorsolutions/http-common-go/pkg/middleware/context"
	"github.com/ranorsolutions/svc-common-go/pkg/service"
)

type HTTPService struct {
	Engine  *gin.Engine
	Server  *http.Server
	Service *service.Service
}

func New(svc *service.Service, version string) (*HTTPService, error) {
	engine := gin.New()

	// Attach middleware
	engine.Use(context.GinContextToContextMiddleware())
	engine.Use(gin.Recovery())

	group := engine.Group(fmt.Sprintf("/api/%s", version))

	// Attach all the routes
	for _, route := range svc.HTTPHandlers {
		switch route.Type {
		case "GET":
			group.GET(route.Path, route.Handler...)
			break
		case "PUT":
			group.PUT(route.Path, route.Handler...)
			break
		case "POST":
			group.POST(route.Path, route.Handler...)
			break
		case "DELETE":
			group.DELETE(route.Path, route.Handler...)
			break
		}
	}

	server := &http.Server{
		Handler: engine,
	}

	s := &HTTPService{
		Server:  server,
		Engine:  engine,
		Service: svc,
	}

	return s, nil
}

func (s *HTTPService) ListenAndServe(l net.Listener) error {
	s.Service.Logger.Info("http server listening %s", formatAddr(l.Addr().String()))
	return s.Server.Serve(l)
}

func formatAddr(addr string) string {
	re := regexp.MustCompile(`\[::\]`)
	return re.ReplaceAllString(addr, "http://localhost")
}
