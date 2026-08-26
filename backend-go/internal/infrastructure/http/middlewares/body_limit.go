package middlewares

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes <= 0 || c.Request.Body == nil {
			c.Next()
			return
		}
		if c.Request.ContentLength > maxBytes {
			AbortWithError(c, PayloadTooLarge("request body is too large", fmt.Errorf("content length %d exceeds %d bytes", c.Request.ContentLength, maxBytes)))
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
