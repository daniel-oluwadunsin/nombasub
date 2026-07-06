package handlers

import (
	"net/http"

	"github.com/daniel-oluwadunsin/nombasub/internal/middleware"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/gin-gonic/gin"
)

func (h *Handler) InitializeDirectDebitTransaction(ctx *gin.Context) {
	var body requests.InitializeDirectDebitRequest

	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantId := ctx.GetString(middleware.TenantIdCtxKey)

	response, err := h.sc.TransactionService.InitializeDirectDebitSubscription(tenantId, body)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Direct debit mandate created successfully", response)
}

func (h *Handler) InitializeCardTransaction(ctx *gin.Context) {
	var body requests.CreateCheckoutOrderRequest

	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantId := ctx.GetString(middleware.TenantIdCtxKey)
	accountId := ctx.GetString(middleware.AccountIdCtxKey)

	response, err := h.sc.TransactionService.InitializeCardTransaction(tenantId, accountId, body)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Transaction initialized successfully", response.Data)
}

func (h *Handler) RefundPaymentOrInvoice(ctx *gin.Context) {
	var body requests.RefundPaymentOrInvoiceRequest

	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantId := ctx.GetString(middleware.TenantIdCtxKey)

	if err := h.sc.TransactionService.RefundPaymentOrInvoice(tenantId, body); err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.SuccessEmpty(ctx, http.StatusOK, "Refund processed successfully")
}

func (h *Handler) GetPaymentIntents(ctx *gin.Context) {
	var query requests.PaymentIntentsQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantId := ctx.GetString(middleware.TenantIdCtxKey)
	result, err := h.sc.TransactionService.ListPaymentIntents(tenantId, query)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Payment attempts retrieved successfully", result)
}

func (h *Handler) GetRefunds(ctx *gin.Context) {
	tenantId := ctx.GetString(middleware.TenantIdCtxKey)
	var query requests.RefundsQuery

	if err := ctx.ShouldBindQuery(&query); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	refunds, err := h.sc.TransactionService.GetRefunds(tenantId, query)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Refunds retrieved successfully", refunds)
}
