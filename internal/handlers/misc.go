package handlers

import (
	"net/http"

	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/gin-gonic/gin"
)

func (h *Handler) Health(ctx *gin.Context) {
	responses.SuccessEmpty(ctx, http.StatusOK, "ok")

}

func (h *Handler) NoRoute(ctx *gin.Context) {
	responses.Error(ctx, responses.NotFound("requested resource does not exist"))
}
