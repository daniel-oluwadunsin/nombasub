package tools

import (
	"context"
	"encoding/json"
	"fmt"

	nsmcp "github.com/daniel-oluwadunsin/nombasub/internal/mcp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Action struct {
	Engine *nsmcp.EngineClient
}

func (a *Action) Register(s *server.MCPServer) {
	s.AddTool(a.retryPaymentTool(), a.handleRetryPayment)
	s.AddTool(a.cancelSubscriptionTool(), a.handleCancelSubscription)
	s.AddTool(a.createPortalLinkTool(), a.handleCreatePortalLink)
	s.AddTool(a.sendDunningReminderTool(), a.handleSendDunningReminder)
}

func destructiveAnnotations(title string) mcp.ToolAnnotation {
	return mcp.ToolAnnotation{
		Title:           title,
		ReadOnlyHint:    mcp.ToBoolPtr(false),
		DestructiveHint: mcp.ToBoolPtr(true),
		IdempotentHint:  mcp.ToBoolPtr(false),
		OpenWorldHint:   mcp.ToBoolPtr(true),
	}
}

func writeAnnotations(title string) mcp.ToolAnnotation {
	return mcp.ToolAnnotation{
		Title:           title,
		ReadOnlyHint:    mcp.ToBoolPtr(false),
		DestructiveHint: mcp.ToBoolPtr(false),
		IdempotentHint:  mcp.ToBoolPtr(false),
		OpenWorldHint:   mcp.ToBoolPtr(true),
	}
}

func dryRunResult(action string, entity json.RawMessage) (*mcp.CallToolResult, error) {
	out := map[string]any{
		"dry_run": true,
		"action":  action,
		"note":    "no changes were made; entity below shows the current state that would be affected",
		"entity":  json.RawMessage(entity),
	}
	pretty, _ := json.MarshalIndent(out, "", "  ")
	return mcp.NewToolResultText(string(pretty)), nil
}

// retry_payment

func (a *Action) retryPaymentTool() mcp.Tool {
	return mcp.NewTool("retry_payment",
		mcp.WithDescription("Retry a failed or open invoice payment using the customer's saved payment source. If the invoice was previously marked failed, it is re-opened and re-processed."),
		mcp.WithToolAnnotation(destructiveAnnotations("Retry invoice payment")),
		mcp.WithString("invoice_id_or_code", mcp.Required(), mcp.Description("Invoice UUID or code (e.g. INV_xxx)")),
		mcp.WithBoolean("dry_run", mcp.Description("If true, do not perform the retry — return the current invoice state instead.")),
	)
}

func (a *Action) handleRetryPayment(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idOrCode, err := req.RequireString("invoice_id_or_code")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if req.GetBool("dry_run", false) {
		entity, err := a.Engine.Get(ctx, "/v1/invoice/"+idOrCode, nil)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return dryRunResult("retry_payment", entity)
	}

	data, err := a.Engine.Post(ctx, "/v1/invoice/"+idOrCode+"/retry", nil, map[string]any{})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return prettyJSON(data)
}

// cancel_subscription

func (a *Action) cancelSubscriptionTool() mcp.Tool {
	return mcp.NewTool("cancel_subscription",
		mcp.WithDescription("Cancel a subscription. This stops future billing cycles and closes any open invoice for the subscription. This action cannot be undone."),
		mcp.WithToolAnnotation(destructiveAnnotations("Cancel subscription")),
		mcp.WithString("subscription_id_or_code", mcp.Required(), mcp.Description("Subscription UUID or code (e.g. SUB_xxx)")),
		mcp.WithBoolean("dry_run", mcp.Description("If true, do not cancel — return the current subscription state instead.")),
	)
}

func (a *Action) handleCancelSubscription(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idOrCode, err := req.RequireString("subscription_id_or_code")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if req.GetBool("dry_run", false) {
		entity, err := a.Engine.Get(ctx, "/v1/subscription/"+idOrCode, nil)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return dryRunResult("cancel_subscription", entity)
	}

	data, err := a.Engine.Post(ctx, "/v1/subscription/"+idOrCode+"/cancel", nil, map[string]any{})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(data) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf(`{"canceled": true, "subscription": %q}`, idOrCode)), nil
	}
	return prettyJSON(data)
}

// create_portal_link

func (a *Action) createPortalLinkTool() mcp.Tool {
	return mcp.NewTool("create_portal_link",
		mcp.WithDescription("Generate a hosted checkout link a customer can use to complete or update payment on a subscription. Optionally email the link directly to the customer."),
		mcp.WithToolAnnotation(writeAnnotations("Create portal link")),
		mcp.WithString("subscription_id_or_code", mcp.Required(), mcp.Description("Subscription UUID or code")),
		mcp.WithBoolean("send_email", mcp.Description("If true, email the checkout link to the customer as well.")),
	)
}

func (a *Action) handleCreatePortalLink(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idOrCode, err := req.RequireString("subscription_id_or_code")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	body := map[string]any{"sendEmail": req.GetBool("send_email", false)}

	data, err := a.Engine.Post(ctx, "/v1/subscription/"+idOrCode+"/checkout-link", nil, body)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return prettyJSON(data)
}

// send_dunning_reminder

func (a *Action) sendDunningReminderTool() mcp.Tool {
	return mcp.NewTool("send_dunning_reminder",
		mcp.WithDescription("Send a payment reminder email to the customer for an unpaid invoice, including a checkout link they can use to complete payment."),
		mcp.WithToolAnnotation(writeAnnotations("Send dunning reminder")),
		mcp.WithString("invoice_id_or_code", mcp.Required(), mcp.Description("Invoice UUID or code (e.g. INV_xxx)")),
	)
}

func (a *Action) handleSendDunningReminder(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idOrCode, err := req.RequireString("invoice_id_or_code")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	data, err := a.Engine.Post(ctx, "/v1/invoice/"+idOrCode+"/send-reminder", nil, map[string]any{})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return prettyJSON(data)
}
