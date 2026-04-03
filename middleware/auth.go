// Copyright (c) 2025-2026 libaxuan
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package middleware

import (
	"cursor2api-go/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthRequired creates a Bearer token authentication middleware.
// It accepts the expected API key directly, avoiding os.Getenv on every request.
func AuthRequired(apiKey string) gin.HandlerFunc {
	expected := "Bearer " + apiKey
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
				"Missing authorization header",
				"authentication_error",
				"missing_auth",
			))
			c.Abort()
			return
		}
		if authHeader != expected {
			// Determine the specific error type for better client feedback
			if !strings.HasPrefix(authHeader, "Bearer ") {
				c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
					"Invalid authorization format. Expected 'Bearer <token>'",
					"authentication_error",
					"invalid_auth_format",
				))
			} else {
				c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
					"Invalid API key",
					"authentication_error",
					"invalid_api_key",
				))
			}
			c.Abort()
			return
		}
		c.Next()
	}
}
