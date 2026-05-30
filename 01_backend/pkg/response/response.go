package response

import "github.com/gin-gonic/gin"

type Body struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(200, Body{Code: 0, Message: "ok", Data: data, RequestID: c.GetString("request_id")})
}

func Error(c *gin.Context, httpCode int, code int, message string) {
	c.JSON(httpCode, Body{Code: code, Message: message, RequestID: c.GetString("request_id")})
}
