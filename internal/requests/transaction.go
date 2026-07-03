package requests

import "github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"

type CreateCheckoutOrderRequest struct {
	nomba.CreateCheckoutOrderRequest
	PlanCode string `json:"planCode" binding:"required"`
}

type InitializeDirectDebitRequest struct {
	PlanCode              string           `json:"planCode" binding:"required"`
	CustomerEmail         string           `json:"customerEmail" binding:"required,email"`
	CustomerAccountNumber string           `json:"customerAccountNumber" binding:"required"`
	CustomerAccountName   string           `json:"customerAccountName" binding:"required"`
	CustomerName          string           `json:"customerName" binding:"required"`
	CustomerAddress       string           `json:"customerAddress" binding:"required"`
	CustomerPhoneNumber   string           `json:"customerPhoneNumber" binding:"required"`
	BankCode              string           `json:"bankCode" binding:"required"`
	Narration             string           `json:"narration" binding:"required"`
	Frequency             nomba.Frequency  `json:"frequency" binding:"required,oneof=VARIABLE WEEKLY MONTHLY QUARTERLY EVERY_TWO_MONTHS EVERY_THREE_MONTHS EVERY_FOUR_MONTHS EVERY_FIVE_MONTHS EVERY_SIX_MONTHS EVERY_SEVEN_MONTHS EVERY_EIGHT_MONTHS EVERY_NINE_MONTHS EVERY_TEN_MONTHS EVERY_ELEVEN_MONTHS EVERY_TWELVE_MONTHS"`
	StartDate             string           `json:"startDate" binding:"required"`
	EndDate               string           `json:"endDate" binding:"required"`
	StartImmediately      bool             `json:"startImmediately"`
	OrderReference        *string          `json:"orderReference"`
}
