package handlers

import (
	"net/http"

	"github.com/daniel-oluwadunsin/nombasub/internal/middleware"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/gin-gonic/gin"
)

func (h *Handler) CreatePlan(ctx *gin.Context) {
	var body requests.CreatePlanRequest

	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantId := ctx.GetString(middleware.TenantIdCtxKey)

	plan, err := h.sc.PlanService.CreatePlan(tenantId, body)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusCreated, "Plan created successfully", plan)
}

func (h *Handler) GetPlan(ctx *gin.Context) {
	planCode := ctx.Param("planCode")
	tenantId := ctx.GetString(middleware.TenantIdCtxKey)

	plan, err := h.sc.PlanService.GetPlan(tenantId, planCode)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Plan retrieved successfully", plan)
}

func (h *Handler) UpdatePlan(ctx *gin.Context) {
	var body requests.UpdatePlanRequest

	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	planCode := ctx.Param("planCode")
	tenantId := ctx.GetString(middleware.TenantIdCtxKey)

	plan, err := h.sc.PlanService.UpdatePlan(tenantId, planCode, body)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Plan updated successfully", plan)
}

func (h *Handler) GetPlans(ctx *gin.Context) {
	tenantId := ctx.GetString(middleware.TenantIdCtxKey)
	var query requests.GetPlansQuery

	if err := ctx.ShouldBindQuery(&query); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	plans, err := h.sc.PlanService.GetPlans(tenantId, query)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Plans retrieved successfully", plans)
}
