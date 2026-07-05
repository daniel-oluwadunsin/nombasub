package services

import (
	"fmt"
	"strings"
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
			TenantID:        tenantId,
			PlanID:          plan.ID,
			Index:           1,
			Name:            plan.Name,
			Description:     plan.Description,
			Currency:        plan.Currency,
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

	queryFilter, err := buildPlanQueryFilter(query)
	if err != nil {
		return nil, err
	}

	plans, err := planRepository.FindManyPaginated(
		filter,
		&repositories.FindArgs{
			Filter: queryFilter,
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
			Where("tenant_id = ? AND plan_id = ?", tenantId, plan.ID).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Order("index DESC").
			Limit(1).
			Find(&lastPlanVersion).Error; err != nil {
			return err
		}

		newPlanVersion := &models.PlanVersion{
			TenantID:        tenantId,
			PlanID:          plan.ID,
			Index:           lastPlanVersion.Index + 1,
			Name:            plan.Name,
			Description:     plan.Description,
			Code:            plan.Code,
			Currency:        plan.Currency,
			Amount:          plan.Amount,
			Interval:        plan.Interval,
			IntervalCount:   plan.IntervalCount,
			TrialPeriodDays: plan.TrialPeriodDays,
			InvoiceLimit:    plan.InvoiceLimit,
			Status:          plan.Status,
		}

		if err := trx.Create(newPlanVersion).Error; err != nil {
			return err
		}

		if body.UpdateExistingSubscriptions {
			if err := trx.Model(&models.Subscription{}).
				Where("tenant_id = ? and plan_id = ?", tenantId, plan.ID).
				Select("NextBillingCyclePlanVersion").
				Updates(models.Subscription{NextBillingCyclePlanVersion: &newPlanVersion.ID}).Error; err != nil {
				return err
			}
		}

		return nil
	})

	return plan, nil
}

func (s *PlanService) GetPlanLatestVersion(tenantId string, planCode string) (*models.PlanVersion, error) {
	planVersionRepository := s.rc.PlanVersionRepository

	planVersion, err := planVersionRepository.Find(
		&models.PlanVersion{
			TenantID: tenantId,
			Code:     planCode,
		},
		&repositories.FindArgs{
			OrderBy: []repositories.OrderBy{{Column: "index", Desc: true}},
		},
	)

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if planVersion == nil {
		return nil, responses.NotFound("Plan not found")
	}

	return planVersion, nil
}

func buildPlanQueryFilter(query requests.GetPlansQuery) (*repositories.QueryFilter, error) {
	clauses := []string{}
	args := []interface{}{}

	if query.Search != nil && strings.TrimSpace(*query.Search) != "" {
		search := "%" + strings.TrimSpace(*query.Search) + "%"
		clauses = append(clauses, "(name ILIKE ? OR code ILIKE ? OR description ILIKE ?)")
		args = append(args, search, search, search)
	}

	if query.From != nil && strings.TrimSpace(*query.From) != "" {
		from, err := parsePlanDate(*query.From, false)
		if err != nil {
			return nil, responses.BadRequest("from must use YYYY-MM-DD format")
		}
		clauses = append(clauses, "created_at >= ?")
		args = append(args, *from)
	}

	if query.To != nil && strings.TrimSpace(*query.To) != "" {
		to, err := parsePlanDate(*query.To, true)
		if err != nil {
			return nil, responses.BadRequest("to must use YYYY-MM-DD format")
		}
		clauses = append(clauses, "created_at <= ?")
		args = append(args, *to)
	}

	if len(clauses) == 0 {
		return nil, nil
	}

	return repositories.NewQueryFilter().Where(strings.Join(clauses, " AND "), args...), nil
}

func parsePlanDate(value string, endOfDay bool) (*time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return nil, fmt.Errorf("date must use YYYY-MM-DD format")
	}
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return &parsed, nil
}
