package types

import "github.com/gin-gonic/gin"

type HTTPHandler struct {
	Type    string
	Path    string
	Handler []gin.HandlerFunc
}
