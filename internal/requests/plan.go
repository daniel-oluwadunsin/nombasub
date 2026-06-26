package requests

import "github.com/daniel-oluwadunsin/nombasub/internal/models"

type CreatePlanRequest struct {
	Name            string              `json:"name" binding:"required"`
	Description     *string             `json:"description"`
	Amount          int64               `json:"amount" binding:"required"`
	Interval        models.PlanInterval `json:"interval" binding:"required"`
	IntervalCount   *int                `json:"intervalCount"`
	TrialPeriodDays int                 `json:"trialPeriodDays"`
	InvoiceLimit    *int                `json:"invoiceLimit"`
	Currency        string              `json:"currency" binding:"required,default=NGN,oneof=NGN"`
}

type GetPlansQuery struct {
	PaginationQuery
	Status   *models.PlanStatus   `form:"status" binding:"omitempty,oneof=active inactive"`
	Interval *models.PlanInterval `form:"interval" binding:"omitempty,oneof=daily weekly bi-weekly monthly quarterly yearly"`
	Amount   *int64               `form:"amount" binding:"omitempty,gt=0"`
}

type UpdatePlanRequest struct {
	Name            *string              `json:"name"`
	Description     *string              `json:"description"`
	Amount          *int64               `json:"amount" binding:"omitempty,gt=0"`
	Interval        *models.PlanInterval `json:"interval" binding:"omitempty,oneof=daily weekly bi-weekly monthly quarterly yearly"`
	IntervalCount   *int                 `json:"intervalCount" binding:"omitempty,gt=0"`
	TrialPeriodDays *int                 `json:"trialPeriodDays" binding:"omitempty,gte=0"`
	InvoiceLimit    *int                 `json:"invoiceLimit" binding:"omitempty,gt=0"`
	Status          *models.PlanStatus   `json:"status" binding:"omitempty,oneof=active inactive"`
}
