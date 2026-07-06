package handlers

import (
	"net/http"

	"github.com/daniel-oluwadunsin/nombasub/internal/middleware"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetInvoices(ctx *gin.Context) {
	var query requests.GetInvoiceQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	data, err := h.sc.InvoiceService.GetInvoices(tenantID, query)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Invoices retrieved successfully", data)
}

func (h *Handler) GetInvoice(ctx *gin.Context) {
	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	idOrCode := ctx.Param("idOrCode")

	data, err := h.sc.InvoiceService.GetInvoice(tenantID, idOrCode)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Invoice retrieved successfully", data)
}

func (h *Handler) GenerateInvoiceCheckoutLink(ctx *gin.Context) {
	var body requests.GenerateInvoiceCheckoutLinkRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	idOrCode := ctx.Param("idOrCode")

	data, err := h.sc.InvoiceService.GenerateCheckoutLink(tenantID, idOrCode, body.SendEmail)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Checkout link generated successfully", data)
}
