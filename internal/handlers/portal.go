package handlers

import (
	"net/http"

	"github.com/daniel-oluwadunsin/nombasub/internal/middleware"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/gin-gonic/gin"
)

func (h *Handler) InitiatePortalSession(ctx *gin.Context) {
	var body requests.InitiatePortalSessionRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	data, err := h.sc.PortalService.InitiateSession(body)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Portal session code sent successfully", data)
}

func (h *Handler) VerifyPortalSession(ctx *gin.Context) {
	var body requests.VerifyPortalSessionRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	data, err := h.sc.PortalService.VerifySession(body)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Portal session verified successfully", data)
}

func (h *Handler) GetPortalSession(ctx *gin.Context) {
	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	customerID := ctx.GetString(middleware.PortalCustomerIdCtxKey)

	data, err := h.sc.PortalService.CurrentSession(tenantID, customerID)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Portal session retrieved successfully", data)
}

func (h *Handler) UpdatePortalProfile(ctx *gin.Context) {
	var body requests.UpdatePortalProfileRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	customerID := ctx.GetString(middleware.PortalCustomerIdCtxKey)

	data, err := h.sc.PortalService.UpdateProfile(tenantID, customerID, body)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Portal profile updated successfully", data)
}

func (h *Handler) GetPortalAnalytics(ctx *gin.Context) {
	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	customerID := ctx.GetString(middleware.PortalCustomerIdCtxKey)

	data, err := h.sc.PortalService.Analytics(tenantID, customerID)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Portal analytics retrieved successfully", data)
}

func (h *Handler) GetPortalPaymentSources(ctx *gin.Context) {
	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	customerID := ctx.GetString(middleware.PortalCustomerIdCtxKey)

	data, err := h.sc.PortalService.PaymentSources(tenantID, customerID)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Portal payment sources retrieved successfully", data)
}

func (h *Handler) InitiatePortalCardUpdate(ctx *gin.Context) {
	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	customerID := ctx.GetString(middleware.PortalCustomerIdCtxKey)

	data, err := h.sc.PortalService.InitiateCardUpdate(tenantID, customerID, ctx.Param("paymentSourceID"))
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Card update checkout created successfully", data)
}

func (h *Handler) DisablePortalPaymentSource(ctx *gin.Context) {
	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	customerID := ctx.GetString(middleware.PortalCustomerIdCtxKey)

	if err := h.sc.PortalService.DisablePaymentSource(tenantID, customerID, ctx.Param("paymentSourceID")); err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.SuccessEmpty(ctx, http.StatusOK, "Payment method disabled successfully")
}

func (h *Handler) GetPortalSubscriptions(ctx *gin.Context) {
	var query requests.GetSubscriptionQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	customerID := ctx.GetString(middleware.PortalCustomerIdCtxKey)

	data, err := h.sc.PortalService.Subscriptions(tenantID, customerID, query)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Portal subscriptions retrieved successfully", data)
}

func (h *Handler) GetPortalSubscription(ctx *gin.Context) {
	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	customerID := ctx.GetString(middleware.PortalCustomerIdCtxKey)

	data, err := h.sc.PortalService.Subscription(tenantID, customerID, ctx.Param("idOrCode"))
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Portal subscription retrieved successfully", data)
}

func (h *Handler) CancelPortalSubscription(ctx *gin.Context) {
	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	customerID := ctx.GetString(middleware.PortalCustomerIdCtxKey)

	if err := h.sc.PortalService.CancelSubscription(tenantID, customerID, ctx.Param("idOrCode")); err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.SuccessEmpty(ctx, http.StatusOK, "Subscription canceled successfully")
}

func (h *Handler) UpdatePortalSubscriptionPaymentMethod(ctx *gin.Context) {
	var body requests.UpdatePortalSubscriptionPaymentMethodRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	customerID := ctx.GetString(middleware.PortalCustomerIdCtxKey)

	data, err := h.sc.PortalService.UpdateSubscriptionPaymentMethod(tenantID, customerID, ctx.Param("idOrCode"), body)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Subscription payment method updated successfully", data)
}

func (h *Handler) GetPortalInvoices(ctx *gin.Context) {
	var query requests.GetInvoiceQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	customerID := ctx.GetString(middleware.PortalCustomerIdCtxKey)

	data, err := h.sc.PortalService.Invoices(tenantID, customerID, query)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Portal invoices retrieved successfully", data)
}

func (h *Handler) GetPortalInvoice(ctx *gin.Context) {
	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	customerID := ctx.GetString(middleware.PortalCustomerIdCtxKey)

	data, err := h.sc.PortalService.Invoice(tenantID, customerID, ctx.Param("idOrCode"))
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Portal invoice retrieved successfully", data)
}

func (h *Handler) RetryPortalInvoicePayment(ctx *gin.Context) {
	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	customerID := ctx.GetString(middleware.PortalCustomerIdCtxKey)

	data, err := h.sc.PortalService.RetryInvoicePayment(tenantID, customerID, ctx.Param("idOrCode"))
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Invoice payment retry initiated", data)
}

func (h *Handler) GetPortalRefunds(ctx *gin.Context) {
	var query requests.RefundsQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	customerID := ctx.GetString(middleware.PortalCustomerIdCtxKey)

	data, err := h.sc.PortalService.Refunds(tenantID, customerID, query)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Portal refunds retrieved successfully", data)
}
