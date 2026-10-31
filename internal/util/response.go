package util

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Response defines the standard envelope for all API responses.
type Response struct {
	Success bool        `json:"success"`           // true for 2xx responses, false otherwise
	Message string      `json:"message,omitempty"` // optional human‑readable message
	Data    interface{} `json:"data,omitempty"`    // payload for successful calls
	Error   string      `json:"error,omitempty"`   // error description for failed calls
}

// Success writes a 200 OK response with the supplied payload.
func Success(c *gin.Context, data interface{}) {
	writeResponse(c, http.StatusOK, true, "", data, "")
}

// Created writes a 201 Created response with the supplied payload.
func Created(c *gin.Context, data interface{}) {
	writeResponse(c, http.StatusCreated, true, "", data, "")
}

// NoContent writes a 204 No Content response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// BadRequest writes a 400 Bad Request response with an error message.
func BadRequest(c *gin.Context, err error) {
	writeResponse(c, http.StatusBadRequest, false, "", nil, err.Error())
}

// Unauthorized writes a 401 Unauthorized response with an error message.
func Unauthorized(c *gin.Context, err error) {
	writeResponse(c, http.StatusUnauthorized, false, "", nil, err.Error())
}

// Forbidden writes a 403 Forbidden response with an error message.
func Forbidden(c *gin.Context, err error) {
	writeResponse(c, http.StatusForbidden, false, "", nil, err.Error())
}

// NotFound writes a 404 Not Found response with an error message.
func NotFound(c *gin.Context, err error) {
	writeResponse(c, http.StatusNotFound, false, "", nil, err.Error())
}

// ValidationError writes a 422 Unprocessable Entity response containing field‑level validation errors.
func ValidationError(c *gin.Context, fieldErrors map[string]string) {
	resp := Response{
		Success: false,
		Message: "validation error",
		Data:    fieldErrors,
	}
	c.JSON(http.StatusUnprocessableEntity, resp)
}

// InternalError writes a 500 Internal Server Error response.
// The original error is logged (if a logger is provided) but not exposed to the client.
func InternalError(c *gin.Context, err error, logger *zap.Logger) {
	if logger != nil {
		logger.Error("internal server error",
			zap.Error(err),
			zap.String("path", c.FullPath()),
			zap.String("method", c.Request.Method),
		)
	}
	// Do not leak internal details; return a generic message.
	writeResponse(c, http.StatusInternalServerError, false, "internal server error", nil, "")
}

// writeResponse builds the envelope and writes JSON with the given HTTP status.
func writeResponse(c *gin.Context, status int, success bool, message string, data interface{}, errMsg string) {
	resp := Response{
		Success: success,
	}
	if message != "" {
		resp.Message = message
	}
	if data != nil {
		resp.Data = data
	}
	if errMsg != "" {
		resp.Error = errMsg
	}
	c.JSON(status, resp)
}