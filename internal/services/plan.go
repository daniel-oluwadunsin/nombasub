package services

import (
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PlanService struct {
	rc *repositories.Container
}

func NewPlanService(rc *repositories.Container) *PlanService {
	return &PlanService{rc: rc}
}

func (s *PlanService) CreatePlan(tenantId string, body requests.CreatePlanRequest) (*models.Plan, error) {
	db := s.rc.DB

	code, err := utils.GenerateCode("PLN")

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	var plan *models.Plan

	err = db.Transaction(func(trx *gorm.DB) error {
		plan = &models.Plan{
			Name:            body.Name,
			Description:     body.Description,
			TenantID:        tenantId,
			Code:            code,
			Amount:          body.Amount,
			Currency:        body.Currency,
			Interval:        body.Interval,
			IntervalCount:   body.IntervalCount,
			TrialPeriodDays: body.TrialPeriodDays,
			InvoiceLimit:    body.InvoiceLimit,
		}

		if err := trx.Create(plan).Error; err != nil {
			return err
		}

		planVersion := &models.PlanVersion{
			PlanID:          plan.ID,
			Index:           1,
			Name:            plan.Name,
			Description:     plan.Description,
			Code:            plan.Code,
			Amount:          plan.Amount,
			Interval:        plan.Interval,
			IntervalCount:   plan.IntervalCount,
			TrialPeriodDays: plan.TrialPeriodDays,
			InvoiceLimit:    plan.InvoiceLimit,
		}

		if err := trx.Create(planVersion).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return plan, nil
}

func (s *PlanService) GetPlans(tenantId string, query requests.GetPlansQuery) (*responses.PaginatedResponse[models.Plan], error) {
	planRepository := s.rc.PlanRepository

	filter := &models.Plan{TenantID: tenantId}

	if query.Status != nil {
		filter.Status = *query.Status
	}
	if query.Interval != nil {
		filter.Interval = *query.Interval
	}
	if query.Amount != nil {
		filter.Amount = *query.Amount
	}

	plans, err := planRepository.FindManyPaginated(
		filter,
		&repositories.FindArgs{
			OrderBy: []repositories.OrderBy{
				{Column: "created_at", Desc: true},
			},
		},
		&query.PaginationQuery,
	)

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return plans, nil
}

func (s *PlanService) GetPlan(tenantId string, planCode string) (*models.Plan, error) {
	planRepository := s.rc.PlanRepository

	plan, err := planRepository.Find(&models.Plan{TenantID: tenantId, Code: planCode}, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if plan == nil {
		return nil, responses.NotFound("Plan not found")
	}

	return plan, nil
}

func (s *PlanService) UpdatePlan(tenantId string, planCode string, body requests.UpdatePlanRequest) (*models.Plan, error) {
	planRepository := s.rc.PlanRepository
	db := s.rc.DB

	plan, err := planRepository.Find(&models.Plan{TenantID: tenantId, Code: planCode}, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if plan == nil {
		return nil, responses.NotFound("Plan not found")
	}

	err = db.Transaction(func(trx *gorm.DB) error {
		plan.Name = *utils.Or(body.Name, &plan.Name)
		plan.Description = utils.Or(body.Description, plan.Description)
		plan.Amount = *utils.Or(body.Amount, &plan.Amount)
		plan.Interval = *utils.Or(body.Interval, &plan.Interval)
		plan.IntervalCount = utils.Or(body.IntervalCount, plan.IntervalCount)
		plan.TrialPeriodDays = *utils.Or(body.TrialPeriodDays, &plan.TrialPeriodDays)
		plan.InvoiceLimit = utils.Or(body.InvoiceLimit, plan.InvoiceLimit)
		plan.Status = *utils.Or(body.Status, &plan.Status)
		if plan.Status == models.PlanStatusInactive && plan.ArchivedAt == nil {
			plan.ArchivedAt = utils.ToPtr(time.Now())
		}

		if err := trx.Updates(plan).Error; err != nil {
			return err
		}

		var lastPlanVersion models.PlanVersion

		if err := trx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Order("index DESC").
			Limit(1).
			Find(&lastPlanVersion).Error; err != nil {
			return err
		}

		newPlanVersion := &models.PlanVersion{
			PlanID:          plan.ID,
			Index:           lastPlanVersion.Index + 1,
			Name:            plan.Name,
			Description:     plan.Description,
			Code:            plan.Code,
			Amount:          plan.Amount,
			Interval:        plan.Interval,
			IntervalCount:   plan.IntervalCount,
			TrialPeriodDays: plan.TrialPeriodDays,
			InvoiceLimit:    plan.InvoiceLimit,
		}

		if err := trx.Create(newPlanVersion).Error; err != nil {
			return err
		}

		return nil
	})

	return plan, nil
}
