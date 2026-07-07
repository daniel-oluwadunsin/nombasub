package nomba

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"resty.dev/v3"
)

// detectNombaBusinessError catches responses where Nomba returned HTTP 200 but
// embedded a business-level failure (e.g. `{"status": false, "message": "..."}`).
// Without this check, callers would receive a zero-value success struct and
// silently proceed with empty IDs.
func detectNombaBusinessError(op string, res *resty.Response) error {
	body := strings.TrimSpace(res.String())
	if body == "" {
		return nil
	}

	var envelope struct {
		Status  *bool  `json:"status"`
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		return nil
	}
	if envelope.Status == nil || *envelope.Status {
		return nil
	}

	message := strings.TrimSpace(envelope.Message)
	if message == "" {
		message = fmt.Sprintf("Nomba %s returned status:false with no message", op)
	}
	log.Printf("nomba %s business failure: code=%q message=%q body=%s", op, envelope.Code, message, body)

	return &responses.AppError{
		StatusCode: 422,
		Message:    message,
		Data:       body,
		Err:        errors.New(message),
	}
}

// logNombaRequest prints the outgoing request payload so we can compare what
// we sent against what Nomba rejected. Only active in debug builds; noisy but
// invaluable for diagnosing 4xx responses with empty bodies.
func logNombaRequest(op string, body any) {
	payload, err := json.Marshal(body)
	if err != nil {
		log.Printf("nomba %s request: (marshal error: %v)", op, err)
		return
	}
	log.Printf("nomba %s request: %s", op, string(payload))
}

type Response[T any] struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Data        T      `json:"data"`
}

type errorResponse = Response[struct{}]

// buildNombaError constructs a rich AppError from a failed Nomba HTTP response,
// falling back to the raw body when the response is not shaped like errorResponse
// (e.g. Nomba returned an HTML 502 page or an unexpected schema). It also logs
// the full status + body so the failure is visible in server logs.
func buildNombaError(op string, res *resty.Response) error {
	status := res.StatusCode()
	body := strings.TrimSpace(res.String())

	description := ""
	code := ""
	if parsed, ok := res.ResultError().(*errorResponse); ok && parsed != nil {
		description = strings.TrimSpace(parsed.Description)
		code = strings.TrimSpace(parsed.Code)
	}

	message := description
	if message == "" {
		if body != "" {
			message = fmt.Sprintf("Nomba %s failed with %d: %s", op, status, body)
		} else {
			message = fmt.Sprintf("Nomba %s failed with %d (empty response body)", op, status)
		}
	}

	headerParts := []string{}
	for _, h := range []string{"X-Request-Id", "X-Correlation-Id", "X-Nomba-Request-Id", "X-Nomba-Error", "Content-Type"} {
		if v := res.Header().Get(h); v != "" {
			headerParts = append(headerParts, fmt.Sprintf("%s=%q", h, v))
		}
	}
	log.Printf("nomba %s failed: status=%d code=%q description=%q headers=[%s] body=%s",
		op, status, code, description, strings.Join(headerParts, " "), body)

	return &responses.AppError{
		StatusCode: status,
		Message:    message,
		Data:       body,
		Err:        errors.New(message),
	}
}

type GetAccessTokenResponse = Response[struct {
	BusinessID   string `json:"businessId"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expiresAt"`
}]

type CreateCheckoutOrderResponse = Response[struct {
	CheckoutLink   string `json:"checkoutLink"`
	OrderReference string `json:"orderReference"`
}]

type ChargeCardResponse = Response[struct{}]

type CreateDirectDebitManadateResponse = Response[struct {
	MandateID           string `json:"mandateId"`
	MerchantReference   string `json:"merchantReference"`
	CustomerPhoneNumber string `json:"customerPhoneNumber"`
	Description         string `json:"description"`
}]

type GetDirectDebitManadateResponse = Response[struct {
	MandateId             string        `json:"mandateId"`
	CustomerAccountName   string        `json:"customerAccountName"`
	CustomerAccountNumber string        `json:"customerAccountNumber"`
	MandateStatus         MandateStatus `json:"mandateStatus"`
	RejectionReason       string        `json:"rejectionReason"`
	MandateAdviceStatus   string        `json:"mandateAdviceStatus"`
}]

type RequestRefundResponse = Response[struct {
}]

type TransferToAccountResponse = Response[struct {
	ID               *string        `json:"id"`
	Status           TransferStatus `json:"status"`
	Type             *string        `json:"type"`
	Amount           *float64       `json:"amount"`
	Source           *string        `json:"source"`
	SourceUserId     *string        `json:"sourceUserId"`
	CustomerBillerId *string        `json:"customerBillerId"`
	ProductId        *string        `json:"productId"`
	Meta             *struct {
		BankCode            *string `json:"bankCode"`
		ApiClientId         *string `json:"api_client_id"`
		ApiRrn              *string `json:"api_rrn"`
		ApiAccountId        *string `json:"api_account_id"`
		SenderName          *string `json:"sender_name"`
		BankName            *string `json:"bankName"`
		SessionId           *string `json:"sessionId"`
		UserName            *string `json:"userName"`
		AccountNumber       *string `json:"accountNumber"`
		Rrn                 *string `json:"rrn"`
		HooksEligible       *string `json:"hooksEligible"`
		MerchantTxRef       *string `json:"merchantTxRef"`
		BankingEntityType   *string `json:"banking_entity_type"`
		UserId              *string `json:"user_id"`
		IsCorporate         *string `json:"isCorporate"`
		Narration           *string `json:"narration"`
		TransactionCategory *string `json:"transactionCategory"`
		RecipientName       *string `json:"recipientName"`
		Currency            *string `json:"currency"`
	} `json:"meta"`
	UserId      *string `json:"userId"`
	TimeCreated *string `json:"timeCreated"`
}]

type DebitMandateResponse = Response[struct {
	MandateId string  `json:"mandateId"`
	Status    string  `json:"status"`
	Amount    float64 `json:"amount"`
	Message   string  `json:"message"`
}]

type DirectDebitStatusItem struct {
	Status                string    `json:"status"`
	CustomerAccountNumber string    `json:"customerAccountNumber"`
	CustomerAccountName   string    `json:"customerAccountName"`
	BankCode              string    `json:"bankCode"`
	Amount                float64   `json:"amount"`
	CustomerName          string    `json:"customerName"`
	CustomerAddress       string    `json:"customerAddress"`
	CustomerEmail         string    `json:"customerEmail"`
	CustomerPhoneNumber   string    `json:"customerPhoneNumber"`
	MerchantReference     string    `json:"merchantReference"`
	Frequency             Frequency `json:"frequency"`
	StartDate             []int     `json:"startDate"`
	EndDate               []int     `json:"endDate"`
	MandateAdviceStatus   string    `json:"mandateAdviceStatus"`
	MandateId             string    `json:"mandateId"`
}

type UpdateDirectDebitStatusResponse = Response[struct {
	Items []DirectDebitStatusItem `json:"items"`
}]

type VerifyCheckoutOrderResponse = Response[struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Order   struct {
		OrderID        string  `json:"orderId"`
		OrderReference string  `json:"orderReference"`
		Amount         float64 `json:"amount,string"`
		Currency       string  `json:"currency"`
		CustomerEmail  string  `json:"customerEmail"`
	} `json:"order"`
	TransactionDetails struct {
		TransactionDate      string `json:"transactionDate"`
		PaymentReference     string `json:"paymentReference"`
		StatusCode           string `json:"statusCode"`
		TokenizedCardPayment string `json:"tokenizedCardPayment"`
	} `json:"transactionDetails"`
	CardDetails struct {
		CardPan      string `json:"cardPan"`
		CardType     string `json:"cardType"`
		CardCurrency string `json:"cardCurrency"`
	} `json:"cardDetails"`
}]
