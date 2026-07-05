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

	tenant, err := h.sc.AuthService.LoginTenant(body)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Tenant logged in successfully", gin.H{
		"accessToken": tenant.AccessToken,
		"expiresAt":   tenant.AccessTokenExpiresAt,
		"tenant": gin.H{
			"accountId":    tenant.AccountID,
			"businessName": tenant.BusinessName,
		},
	})
}

func (h *Handler) SignOutTenant(ctx *gin.Context) {
	tenantId := ctx.GetString(middleware.TenantIdCtxKey)

	if err := h.sc.AuthService.SignOutTenant(tenantId); err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.SuccessEmpty(ctx, http.StatusOK, "Signed out successfully")
}

func (h *Handler) GetTenantSettings(ctx *gin.Context) {
	tenantId := ctx.GetString(middleware.TenantIdCtxKey)

	settings, err := h.sc.AuthService.GetTenantSettings(tenantId)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Tenant settings retrieved successfully", settings)
}

func (h *Handler) UpdateTenantSettings(ctx *gin.Context) {
	var body requests.UpdateTenantSettingsRequest

	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantId := ctx.GetString(middleware.TenantIdCtxKey)
	settings, err := h.sc.AuthService.UpdateTenantSettings(tenantId, body)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Tenant settings updated successfully", settings)
}

func (h *Handler) RotateTenantApiKey(ctx *gin.Context) {
	tenantId := ctx.GetString(middleware.TenantIdCtxKey)

	settings, err := h.sc.AuthService.RotateTenantApiKey(tenantId)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "API key rotated successfully", settings)
}

func (h *Handler) RotateTenantWebhookSecret(ctx *gin.Context) {
	tenantId := ctx.GetString(middleware.TenantIdCtxKey)

	settings, err := h.sc.AuthService.RotateTenantWebhookSecret(tenantId)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Webhook secret rotated successfully", settings)
}

func (h *Handler) ChangeTenantPassword(ctx *gin.Context) {
	var body requests.ChangePasswordRequest

	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantId := ctx.GetString(middleware.TenantIdCtxKey)
	if err := h.sc.AuthService.ChangeTenantPassword(tenantId, body); err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.SuccessEmpty(ctx, http.StatusOK, "Password changed successfully")
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
