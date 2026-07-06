package tools

import (
	"context"
	"encoding/json"
	"fmt"

	nsmcp "github.com/daniel-oluwadunsin/nombasub/internal/mcp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Query struct {
	Engine *nsmcp.EngineClient
}

func (q *Query) Register(s *server.MCPServer) {
	s.AddTool(q.getCustomersTool(), q.handleGetCustomers)
	s.AddTool(q.getSubscriptionsTool(), q.handleGetSubscriptions)
	s.AddTool(q.getInvoicesTool(), q.handleGetInvoices)
	s.AddTool(q.getPlansTool(), q.handleGetPlans)
	s.AddTool(q.getPaymentAttemptsTool(), q.handleGetPaymentAttempts)
	s.AddTool(q.getWebhookDeliveriesTool(), q.handleGetWebhookDeliveries)
}

func readOnlyAnnotations(title string) mcp.ToolAnnotation {
	return mcp.ToolAnnotation{
		Title:           title,
		ReadOnlyHint:    mcp.ToBoolPtr(true),
		DestructiveHint: mcp.ToBoolPtr(false),
		IdempotentHint:  mcp.ToBoolPtr(true),
		OpenWorldHint:   mcp.ToBoolPtr(true),
	}
}

func paginationOptions() []mcp.ToolOption {
	return []mcp.ToolOption{
		mcp.WithNumber("limit", mcp.Description("Max items per page (default 20, max 100)")),
		mcp.WithNumber("page", mcp.Description("Page number, 1-indexed")),
	}
}

func applyPagination(req mcp.CallToolRequest, query map[string]string) {
	if v := req.GetInt("limit", 0); v > 0 {
		query["limit"] = fmt.Sprintf("%d", v)
	}
	if v := req.GetInt("page", 0); v > 0 {
		query["page"] = fmt.Sprintf("%d", v)
	}
}

func setIfNotEmpty(query map[string]string, req mcp.CallToolRequest, keys ...string) {
	for _, k := range keys {
		if v := req.GetString(k, ""); v != "" {
			query[k] = v
		}
	}
}

func prettyJSON(data json.RawMessage) (*mcp.CallToolResult, error) {
	pretty, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(pretty)), nil
}

// get_customers

func (q *Query) getCustomersTool() mcp.Tool {
	opts := append([]mcp.ToolOption{
		mcp.WithDescription("List customers for the current merchant. Supports pagination, search, and date filtering. Returns customer profiles with lifetime value and subscription counts."),
		mcp.WithToolAnnotation(readOnlyAnnotations("Get customers")),
		mcp.WithString("search", mcp.Description("Search by name, email, or customer code")),
		mcp.WithString("from", mcp.Description("Filter customers created on/after this date (YYYY-MM-DD)")),
		mcp.WithString("to", mcp.Description("Filter customers created on/before this date (YYYY-MM-DD)")),
	}, paginationOptions()...)
	return mcp.NewTool("get_customers", opts...)
}

func (q *Query) handleGetCustomers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := map[string]string{}
	setIfNotEmpty(query, req, "search", "from", "to")
	applyPagination(req, query)

	data, err := q.Engine.Get(ctx, "/v1/customer/", query)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return prettyJSON(data)
}

// get_subscriptions

func (q *Query) getSubscriptionsTool() mcp.Tool {
	opts := append([]mcp.ToolOption{
		mcp.WithDescription("List subscriptions for the current merchant. Filter by customer, plan, or free-text search. Returns subscription details, plan, next billing date, and status."),
		mcp.WithToolAnnotation(readOnlyAnnotations("Get subscriptions")),
		mcp.WithString("search", mcp.Description("Search by subscription code or customer email")),
		mcp.WithString("customer", mcp.Description("Customer email, code, or ID to filter by")),
		mcp.WithString("plan", mcp.Description("Plan code to filter by")),
	}, paginationOptions()...)
	return mcp.NewTool("get_subscriptions", opts...)
}

func (q *Query) handleGetSubscriptions(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := map[string]string{}
	setIfNotEmpty(query, req, "search", "customer", "plan")
	applyPagination(req, query)

	data, err := q.Engine.Get(ctx, "/v1/subscription/", query)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return prettyJSON(data)
}

// get_invoices

func (q *Query) getInvoicesTool() mcp.Tool {
	opts := append([]mcp.ToolOption{
		mcp.WithDescription("List invoices for the current merchant. Filter by status (draft, open, paid, failed, refunded). Use search to find by invoice code, plan name, or subscription code."),
		mcp.WithToolAnnotation(readOnlyAnnotations("Get invoices")),
		mcp.WithString("status", mcp.Description("Filter by invoice status: draft, open, paid, failed, or refunded")),
		mcp.WithString("search", mcp.Description("Search by invoice code, plan name, or subscription code")),
	}, paginationOptions()...)
	return mcp.NewTool("get_invoices", opts...)
}

func (q *Query) handleGetInvoices(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := map[string]string{}
	setIfNotEmpty(query, req, "status", "search")
	applyPagination(req, query)

	data, err := q.Engine.Get(ctx, "/v1/invoice/", query)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return prettyJSON(data)
}

// get_plans

func (q *Query) getPlansTool() mcp.Tool {
	opts := append([]mcp.ToolOption{
		mcp.WithDescription("List billing plans for the current merchant. Filter by status, interval, or amount."),
		mcp.WithToolAnnotation(readOnlyAnnotations("Get plans")),
		mcp.WithString("status", mcp.Description("Filter by status: active or inactive")),
		mcp.WithString("interval", mcp.Description("Filter by billing interval: daily, weekly, bi-weekly, monthly, quarterly, yearly")),
		mcp.WithNumber("amount", mcp.Description("Filter by exact plan amount in minor units (e.g. kobo)")),
		mcp.WithString("search", mcp.Description("Search by plan name or code")),
		mcp.WithString("from", mcp.Description("Filter plans created on/after this date (YYYY-MM-DD)")),
		mcp.WithString("to", mcp.Description("Filter plans created on/before this date (YYYY-MM-DD)")),
	}, paginationOptions()...)
	return mcp.NewTool("get_plans", opts...)
}

func (q *Query) handleGetPlans(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := map[string]string{}
	setIfNotEmpty(query, req, "status", "interval", "search", "from", "to")
	if v := req.GetInt("amount", 0); v > 0 {
		query["amount"] = fmt.Sprintf("%d", v)
	}
	applyPagination(req, query)

	data, err := q.Engine.Get(ctx, "/v1/plan/", query)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return prettyJSON(data)
}

// get_payment_attempts

func (q *Query) getPaymentAttemptsTool() mcp.Tool {
	opts := append([]mcp.ToolOption{
		mcp.WithDescription("List payment attempts (payment intents) for the current merchant. Each entry is one attempt to charge a customer, whether success, failed, pending, or refund. Includes provider reference and failure reason."),
		mcp.WithToolAnnotation(readOnlyAnnotations("Get payment attempts")),
		mcp.WithString("status", mcp.Description("Filter by status: PENDING_BILLING, SUCCESS, PAYMENT_FAILED, REFUND, CANCELLED")),
		mcp.WithString("customerId", mcp.Description("Filter by customer UUID")),
		mcp.WithString("subscriptionId", mcp.Description("Filter by subscription UUID")),
		mcp.WithString("invoiceId", mcp.Description("Filter by invoice UUID")),
		mcp.WithString("search", mcp.Description("Search by payment code or provider reference")),
		mcp.WithString("from", mcp.Description("Filter attempts created on/after this date (YYYY-MM-DD)")),
		mcp.WithString("to", mcp.Description("Filter attempts created on/before this date (YYYY-MM-DD)")),
	}, paginationOptions()...)
	return mcp.NewTool("get_payment_attempts", opts...)
}

func (q *Query) handleGetPaymentAttempts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := map[string]string{}
	setIfNotEmpty(query, req, "status", "customerId", "subscriptionId", "invoiceId", "search", "from", "to")
	applyPagination(req, query)

	data, err := q.Engine.Get(ctx, "/v1/checkout/payment-attempts", query)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return prettyJSON(data)
}

// get_webhook_deliveries

func (q *Query) getWebhookDeliveriesTool() mcp.Tool {
	opts := append([]mcp.ToolOption{
		mcp.WithDescription("List outbound webhook deliveries sent from the merchant. Filter by status, event type, or date range. Useful for support and debugging."),
		mcp.WithToolAnnotation(readOnlyAnnotations("Get webhook deliveries")),
		mcp.WithString("status", mcp.Description("Filter by delivery status")),
		mcp.WithString("eventType", mcp.Description("Filter by webhook event type (e.g. invoice.paid, subscription.created)")),
		mcp.WithString("from", mcp.Description("Filter deliveries on/after this date (YYYY-MM-DD)")),
		mcp.WithString("to", mcp.Description("Filter deliveries on/before this date (YYYY-MM-DD)")),
	}, paginationOptions()...)
	return mcp.NewTool("get_webhook_deliveries", opts...)
}

func (q *Query) handleGetWebhookDeliveries(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := map[string]string{}
	setIfNotEmpty(query, req, "status", "eventType", "from", "to")
	applyPagination(req, query)

	data, err := q.Engine.Get(ctx, "/v1/webhook-deliveries/", query)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return prettyJSON(data)
}
