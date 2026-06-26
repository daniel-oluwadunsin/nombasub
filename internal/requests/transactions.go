package requests

import "github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"

type CreateCheckoutOrderRequest struct {
	nomba.CreateCheckoutOrderRequest
	PlanCode string `json:"planCode" binding:"required"`
}
