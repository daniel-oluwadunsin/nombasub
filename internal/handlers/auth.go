package handlers

import (
	"net/http"

	"github.com/daniel-oluwadunsin/nombasub/internal/middleware"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterTenant(ctx *gin.Context) {
	var body requests.SignUpTenantRequest

	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantApiKey, err := h.sc.AuthService.RegisterTenant(body)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Tenant registered successfully", gin.H{"apiKey": tenantApiKey})
}

func (h *Handler) LoginTenant(ctx *gin.Context) {
	var body requests.LoginTenantRequest

	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	apiKey, err := h.sc.AuthService.LoginTenant(body)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Tenant logged in successfully", gin.H{"apiKey": apiKey})
}

func (h *Handler) SetWebhookUrl(ctx *gin.Context) {
	var body requests.SetWebhookUrlRequest

	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantId := ctx.GetString(middleware.TenantIdCtxKey)

	secret, err := h.sc.AuthService.SetWebhookUrl(tenantId, body.WebhookUrl)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Webhook URL set successfully", gin.H{"webhookSecret": *secret})
}
