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

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// CursorWebError represents an error from the Cursor Web API.
type CursorWebError struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

func (e *CursorWebError) Error() string { return e.Message }

// NewCursorWebError creates a new CursorWebError.
func NewCursorWebError(statusCode int, message string) *CursorWebError {
	return &CursorWebError{StatusCode: statusCode, Message: message}
}

// RequestValidationError represents a request validation error.
type RequestValidationError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func (e *RequestValidationError) Error() string { return e.Message }

// NewRequestValidationError creates a new RequestValidationError.
func NewRequestValidationError(message, code string) *RequestValidationError {
	return &RequestValidationError{Message: message, Code: code}
}

// ErrorHandler creates a global error handling middleware.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			handleError(c, c.Errors.Last().Err)
		}
	}
}

// HandleError dispatches an error to the appropriate HTTP response.
func HandleError(c *gin.Context, err error) {
	handleError(c, err)
}

func handleError(c *gin.Context, err error) {
	if c.Writer.Written() {
		return
	}

	logrus.WithError(err).Error("API error")

	var statusCode int
	var errorType, code string

	switch e := err.(type) {
	case *CursorWebError:
		statusCode = e.StatusCode
		errorType = "cursor_web_error"
		code = ""
	case *RequestValidationError:
		statusCode = http.StatusBadRequest
		errorType = "invalid_request_error"
		code = e.Code
	default:
		statusCode = http.StatusBadGateway
		errorType = "internal_error"
		code = ""
	}

	c.JSON(statusCode, models.NewErrorResponse(err.Error(), errorType, code))
}

// RecoveryHandler creates a custom panic recovery middleware.
func RecoveryHandler() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logrus.WithField("panic", recovered).Error("Panic recovered")
		if !c.Writer.Written() {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
				"Internal server error", "panic_error", "",
			))
		}
	})
}
