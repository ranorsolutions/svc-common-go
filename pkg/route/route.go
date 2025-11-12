package route

import "github.com/gin-gonic/gin"

// HTTPHandler defines a route that can be registered in an HTTP service.
// It is designed for declarative, data-driven route registration across services.
type Handler struct {
	Method  string
	Path    string
	Handler []gin.HandlerFunc
}
