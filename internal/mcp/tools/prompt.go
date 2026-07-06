package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Prompts struct{}

func (p *Prompts) Register(s *server.MCPServer) {
	s.AddPrompt(mcp.NewPrompt("monthly_business_review",
		mcp.WithPromptDescription("Walk through this month's business performance: revenue, MRR, churn, payment failures, and recommended actions. Uses the merchant's live data via the MCP tools."),
		mcp.WithArgument("focus", mcp.ArgumentDescription("Optional focus area, e.g. \"revenue\", \"churn\", \"payments\"")),
	), p.monthlyBusinessReview)

	s.AddPrompt(mcp.NewPrompt("subscription_health_check",
		mcp.WithPromptDescription("Analyze subscription health: active vs past-due, churn drivers, at-risk customers, and specific next actions per customer."),
		mcp.WithArgument("customer", mcp.ArgumentDescription("Optional customer email or code to focus the check on")),
	), p.subscriptionHealthCheck)
}

func (p *Prompts) monthlyBusinessReview(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	focus := ""
	if req.Params.Arguments != nil {
		focus = req.Params.Arguments["focus"]
	}

	systemMsg := "You are a subscription revenue analyst reviewing a merchant's month with the MCP tools available. " +
		"Always ground your analysis in the tool responses — never invent numbers. " +
		"Be concise and end with a short list of concrete next actions."

	userMsg := "Please produce a monthly business review for this merchant.\n\n" +
		"Steps:\n" +
		"1. Call `generate_business_report` first for the top-level view.\n" +
		"2. For each finding worth explaining, call `explain_metric_change` on the relevant metric (e.g. mrr, churn_rate, payment_failure_rate).\n" +
		"3. Call `compare_periods` on revenue and mrr comparing this_month vs last_month.\n" +
		"4. If failed payments look high, call `generate_dunning_report`.\n" +
		"5. Summarize in Markdown with sections: Highlights, Concerns, Recommended actions.\n"
	if focus != "" {
		userMsg += "\nFocus your review on: " + focus + ".\n"
	}

	return mcp.NewGetPromptResult(
		"Monthly business review",
		[]mcp.PromptMessage{
			mcp.NewPromptMessage(mcp.RoleAssistant, mcp.NewTextContent(systemMsg)),
			mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(userMsg)),
		},
	), nil
}

func (p *Prompts) subscriptionHealthCheck(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	customer := ""
	if req.Params.Arguments != nil {
		customer = req.Params.Arguments["customer"]
	}

	systemMsg := "You are a customer success analyst diagnosing subscription health. " +
		"Ground every conclusion in tool output. When you recommend an action, name the exact MCP tool that would execute it."

	userMsg := "Assess the health of the merchant's subscription portfolio.\n\n" +
		"Steps:\n" +
		"1. Call `compute_metric` for churn_rate, retention_rate, past_due_subscriptions, and payment_failure_rate.\n" +
		"2. Call `get_subscriptions` filtered on any concerning state (past-due, canceled) and inspect the top offenders.\n" +
		"3. Call `explain_metric_change` on churn_rate to surface drivers.\n" +
		"4. Call `generate_dunning_report` to see who is at risk.\n" +
		"5. For each at-risk customer, propose a specific action and name the tool: `send_dunning_reminder`, `retry_payment`, or `create_portal_link`.\n" +
		"6. Return a Markdown summary with sections: Overall health, At-risk customers, Recommended actions.\n"
	if customer != "" {
		userMsg += "\nFocus on customer: " + customer + ". Use `get_customers` / `customers://" + customer + "` first to pull their profile, then apply the steps above to just their subscriptions.\n"
	}

	return mcp.NewGetPromptResult(
		"Subscription health check",
		[]mcp.PromptMessage{
			mcp.NewPromptMessage(mcp.RoleAssistant, mcp.NewTextContent(systemMsg)),
			mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(userMsg)),
		},
	), nil
}
