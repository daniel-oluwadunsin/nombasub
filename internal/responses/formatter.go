package responses

import (
	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func Format(response *Response) gin.H {
	status := "success"
	if !response.Success {
		status = "error"
	}

	return gin.H{
		"status":  status,
		"message": response.Message,
		"data":    response.Data,
	}
}

func respond(ctx *gin.Context, statusCode int, response *Response) {
	ctx.JSON(statusCode, Format(response))
}

func Error(ctx *gin.Context, err error) {
	appError := AppErrorFromError(err)

	respond(ctx, appError.StatusCode, &Response{
		Success: false,
		Message: appError.Message,
		Data:    appError.Data,
	})
}

func Success(ctx *gin.Context, code int, message string, data interface{}) {
	respond(ctx, code, &Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func SuccessEmpty(ctx *gin.Context, code int, message string) {
	respond(ctx, code, &Response{
		Success: true,
		Message: message,
	})
}
