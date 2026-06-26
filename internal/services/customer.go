package services

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"gorm.io/gorm"
)

type CustomerService struct {
	rc *repositories.Container
}

func NewCustomerService(rc *repositories.Container) *CustomerService {
	return &CustomerService{rc: rc}
}

func (s *CustomerService) CreateCustomer(tenantId string, body requests.CreateCustomerRequest) (*models.Customer, error) {
	customerRepository := s.rc.CustomerRepository

	existingCustomer, err := customerRepository.ExistsRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where("tenant_id = ? AND email ILIKE ?", tenantId, body.Email),
	})
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if existingCustomer {
		return nil, responses.Conflict("A customer with this email already exists")
	}

	code, err := utils.GenerateCode("CUST")
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	customer, err := customerRepository.Create(&models.Customer{
		TenantID:    tenantId,
		Name:        body.Name,
		Email:       body.Email,
		PhoneNumber: body.PhoneNumber,
		ExternalRef: body.ExternalRef,
		Code:        code,
	}, nil)

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return customer, nil
}

func (s *CustomerService) GetCustomer(tenantId string, emailOrCode string) (*models.Customer, error) {
	customerRepository := s.rc.CustomerRepository

	customer, err := customerRepository.FindRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"tenant_id = ? AND (email ILIKE ? OR code = ?)",
			tenantId,
			emailOrCode,
			emailOrCode,
		),
		Preloads: []repositories.Preload{
			{Association: "Subscriptions"},
			{Association: "PaymentSources"},
			{Association: "Payments"},
		},
	})

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if customer == nil {
		return nil, responses.NotFound("Customer not found")
	}

	return customer, nil
}

func (s *CustomerService) GetCustomers(tenantId string, query requests.GetCustomersRequest) (*responses.PaginatedResponse[models.Customer], error) {
	customerRepository := s.rc.CustomerRepository

	response, err := customerRepository.FindManyPaginated(
		&models.Customer{TenantID: tenantId},
		&repositories.FindArgs{
			Preloads: []repositories.Preload{
				{Association: "Subscriptions"},
				{Association: "PaymentSources"},
				{Association: "Payments"},
			},
			OrderBy: []repositories.OrderBy{{Column: "created_at", Desc: true}},
		},
		&query.PaginationQuery,
	)

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return response, nil
}

func (s *CustomerService) UpdateCustomer(tenantId string, emailOrCode string, body requests.UpdateCustomerRequest) (*models.Customer, error) {
	customerRepository := s.rc.CustomerRepository

	customer, err := customerRepository.FindRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"tenant_id = ? AND (email ILIKE ? OR code = ?)",
			tenantId,
			emailOrCode,
			emailOrCode,
		),
	})

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if customer == nil {
		return nil, responses.NotFound("Customer not found")
	}

	if body.Name != nil {
		customer.Name = body.Name
	}

	if body.PhoneNumber != nil {
		customer.PhoneNumber = body.PhoneNumber
	}

	if body.ExternalRef != nil {
		customer.ExternalRef = body.ExternalRef
	}

	updatedCustomer, err := customerRepository.Update(customer, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return updatedCustomer, nil
}

func (s *CustomerService) GetOrCreateCustomer(tenantId string, customer models.Customer, trx *gorm.DB) (*models.Customer, error) {
	if customer.Email == "" {
		return nil, responses.BadRequest("Customer email is required")
	}

	customerRepository := s.rc.CustomerRepository

	customerDetails, err := customerRepository.FindRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"tenant_id = ? AND email ILIKE ?",
			tenantId,
			customer.Email,
		),
		Trx: trx,
	})
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if customerDetails == nil {
		code, err := utils.GenerateCode("CUST")
		if err != nil {
			return nil, responses.InternalServerError(err)
		}

		customer.Code = code
		customerDetails, err = customerRepository.Create(&customer, trx)
		if err != nil {
			return nil, responses.InternalServerError(err)
		}
	}

	return customerDetails, nil
}
