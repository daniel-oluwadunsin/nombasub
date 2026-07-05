package handlers

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/middleware"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/gin-gonic/gin"
)

func (h *Handler) CreateCustomer(ctx *gin.Context) {
	tenantId := ctx.GetString(middleware.TenantIdCtxKey)
	var body requests.CreateCustomerRequest

	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	customer, err := h.sc.CustomerService.CreateCustomer(tenantId, body)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, 200, "Customer created successfully", customer)
}

func (h *Handler) GetCustomer(ctx *gin.Context) {
	tenantId := ctx.GetString(middleware.TenantIdCtxKey)
	emailOrCode := ctx.Param("emailOrCode")

	customer, err := h.sc.CustomerService.GetCustomerDetails(tenantId, emailOrCode)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, 200, "Customer retrieved successfully", customer)
}

func (h *Handler) UpdateCustomer(ctx *gin.Context) {
	tenantId := ctx.GetString(middleware.TenantIdCtxKey)
	emailOrCode := ctx.Param("emailOrCode")
	var body requests.UpdateCustomerRequest

	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	customer, err := h.sc.CustomerService.UpdateCustomer(tenantId, emailOrCode, body)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, 200, "Customer updated successfully", customer)
}

func (h *Handler) RemindCustomerCardExpiring(ctx *gin.Context) {
	tenantId := ctx.GetString(middleware.TenantIdCtxKey)
	emailOrCode := ctx.Param("emailOrCode")
	paymentSourceID := ctx.Param("paymentSourceID")

	if err := h.sc.CustomerService.RemindCustomerCardExpiring(tenantId, emailOrCode, paymentSourceID); err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, 200, "Card expiry reminder queued successfully", gin.H{"queued": true})
}

func (h *Handler) GetCustomers(ctx *gin.Context) {
	tenantId := ctx.GetString(middleware.TenantIdCtxKey)
	var query requests.GetCustomersRequest

	if err := ctx.ShouldBindQuery(&query); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	customers, err := h.sc.CustomerService.GetCustomers(tenantId, query)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, 200, "Customers retrieved successfully", customers)
}
