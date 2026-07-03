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
