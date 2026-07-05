package requests

type CreateCustomerRequest struct {
	Name        *string `json:"name"`
	Email       string  `json:"email" binding:"required,email"`
	PhoneNumber *string `json:"phoneNumber"`
	ExternalRef *string `json:"externalRef"`
}

type GetCustomersRequest struct {
	PaginationQuery
	From *string `form:"from"`
	To   *string `form:"to"`
}

type UpdateCustomerRequest struct {
	Name        *string `json:"name"`
	Email       *string `json:"email" binding:"omitempty,email"`
	PhoneNumber *string `json:"phoneNumber"`
	ExternalRef *string `json:"externalRef"`
}
