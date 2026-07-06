package tools

import (
	"context"
	"fmt"
	"strings"

	nsmcp "github.com/daniel-oluwadunsin/nombasub/internal/mcp"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Report struct {
	Engine *nsmcp.EngineClient
}

func (r *Report) Register(s *server.MCPServer) {
	s.AddTool(r.businessReportTool(), r.handleBusinessReport)
	s.AddTool(r.dunningReportTool(), r.handleDunningReport)
}

func money(currency string, minorUnits int64) string {
	major := float64(minorUnits) / 100.0
	if currency == "" {
		currency = "NGN"
	}
	return fmt.Sprintf("%s %s", currency, humanNumber(major))
}

func humanNumber(v float64) string {
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	whole := int64(v)
	frac := v - float64(whole)

	s := fmt.Sprintf("%d", whole)
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	if frac > 0 {
		return fmt.Sprintf("%s%s.%02d", sign, string(out), int(frac*100+0.5))
	}
	return fmt.Sprintf("%s%s.00", sign, string(out))
}

func pct(v float64) string {
	return fmt.Sprintf("%.1f%%", v)
}

// generate_business_report

func (r *Report) businessReportTool() mcp.Tool {
	return mcp.NewTool("generate_business_report",
		mcp.WithDescription("Generate a Markdown-formatted executive business report for the current merchant: revenue, MRR, subscription health, payment performance, top plan, upcoming renewals, and AI-generated recommendations. Great for board updates or Slack-style summaries."),
		mcp.WithToolAnnotation(readOnlyAnnotations("Generate business report")),
		mcp.WithString("from", mcp.Description("Start of reporting period (YYYY-MM-DD). Defaults to start of current month.")),
		mcp.WithString("to", mcp.Description("End of reporting period (YYYY-MM-DD). Defaults to end of current month.")),
	)
}

func (r *Report) handleBusinessReport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	a := &Analytics{Engine: r.Engine}
	an, err := a.fetchAnalytics(ctx, req.GetString("from", ""), req.GetString("to", ""))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var b strings.Builder
	currency := an.Period.Currency

	fmt.Fprintf(&b, "# Business Report\n\n")
	fmt.Fprintf(&b, "**Period:** %s → %s (%s)\n\n", an.Period.From, an.Period.To, an.Period.Timezone)

	fmt.Fprintf(&b, "## Revenue\n\n")
	fmt.Fprintf(&b, "- **Gross revenue:** %s\n", money(currency, an.Summary.GrossRevenue))
	fmt.Fprintf(&b, "- **Net revenue:** %s (after %s in fees)\n", money(currency, an.Summary.NetRevenue), money(currency, an.Summary.PlatformFees))
	fmt.Fprintf(&b, "- **MRR:** %s   **ARR:** %s\n", money(currency, an.Summary.MRR), money(currency, an.Summary.ARR))
	fmt.Fprintf(&b, "- **Revenue growth vs. previous period:** %s\n", pct(an.Summary.RevenueGrowthPercent))
	fmt.Fprintf(&b, "- **ARPU:** %s   **ARPS:** %s\n\n",
		money(currency, an.Revenue.AverageRevenuePerCustomer),
		money(currency, an.Revenue.AverageRevenuePerSubscription))

	fmt.Fprintf(&b, "## Subscriptions\n\n")
	fmt.Fprintf(&b, "- **Active:** %d   **New:** %d   **Canceled:** %d   **Past due:** %d\n",
		an.Summary.ActiveSubscriptions, an.Summary.NewSubscriptions,
		an.Summary.CanceledSubscriptions, an.Summary.PastDueSubscriptions)
	fmt.Fprintf(&b, "- **Churn rate:** %s   **Retention rate:** %s\n\n",
		pct(an.Summary.ChurnRate), pct(an.Summary.RetentionRate))

	fmt.Fprintf(&b, "## Payments\n\n")
	fmt.Fprintf(&b, "- **Attempts:** %d   **Succeeded:** %d   **Failed:** %d\n",
		an.Payments.TotalPaymentAttempts, an.Payments.SuccessfulPayments, an.Payments.FailedPayments)
	fmt.Fprintf(&b, "- **Success rate:** %s\n", pct(an.Payments.SuccessRate))
	if an.Payments.MostCommonFailureReason != "" {
		fmt.Fprintf(&b, "- **Top failure reason:** %s\n", an.Payments.MostCommonFailureReason)
	}
	b.WriteString("\n")

	if top := an.Plans.TopPlanByRevenue; top != nil {
		fmt.Fprintf(&b, "## Top plan\n\n")
		fmt.Fprintf(&b, "- **%s** — %s in gross revenue across %d active subscribers\n\n",
			top.Name, money(currency, top.GrossRevenue), top.ActiveSubscriptions)
	}

	if an.UpcomingRenewals.Next7Days.Count > 0 || an.UpcomingRenewals.Next30Days.Count > 0 {
		fmt.Fprintf(&b, "## Upcoming renewals\n\n")
		fmt.Fprintf(&b, "- **Next 7 days:** %d renewals, expected %s gross\n",
			an.UpcomingRenewals.Next7Days.Count, money(currency, an.UpcomingRenewals.Next7Days.ExpectedGrossRevenue))
		fmt.Fprintf(&b, "- **Next 30 days:** %d renewals, expected %s gross\n\n",
			an.UpcomingRenewals.Next30Days.Count, money(currency, an.UpcomingRenewals.Next30Days.ExpectedGrossRevenue))
	}

	if len(an.TopCustomers) > 0 {
		fmt.Fprintf(&b, "## Top customers\n\n")
		fmt.Fprintf(&b, "| Customer | Revenue | Active subs | Invoices |\n")
		fmt.Fprintf(&b, "|---|---:|---:|---:|\n")
		for _, c := range truncate(an.TopCustomers, 5) {
			label := c.Name
			if label == "" {
				label = c.Email
			}
			fmt.Fprintf(&b, "| %s | %s | %d | %d |\n",
				label, money(currency, c.TotalRevenue), c.ActiveSubscriptions, c.InvoiceCount)
		}
		b.WriteString("\n")
	}

	if len(an.AIInsights) > 0 {
		fmt.Fprintf(&b, "## Insights\n\n")
		for _, ins := range an.AIInsights {
			fmt.Fprintf(&b, "- **%s** — %s\n", ins.Title, ins.Message)
			if ins.RecommendedAction != "" {
				fmt.Fprintf(&b, "  - _Recommended:_ %s\n", ins.RecommendedAction)
			}
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "## Recommendations\n\n")
	writeRecommendations(&b, an)

	return mcp.NewToolResultText(b.String()), nil
}

func writeRecommendations(b *strings.Builder, an *responses.DashboardAnalytics) {
	any := false
	if an.Summary.PastDueSubscriptions > 0 {
		fmt.Fprintf(b, "- Retry past-due invoices (%d subscriptions in past-due state).\n", an.Summary.PastDueSubscriptions)
		any = true
	}
	if len(an.AtRiskCustomers) > 0 {
		fmt.Fprintf(b, "- Reach out to %d at-risk customers before their next billing cycle.\n", len(an.AtRiskCustomers))
		any = true
	}
	if an.Summary.PaymentFailureRate > 10 {
		fmt.Fprintf(b, "- Investigate payment failure rate (%s) — top reason: %s.\n",
			pct(an.Summary.PaymentFailureRate), an.Payments.MostCommonFailureReason)
		any = true
	}
	if an.Webhooks.SuccessRate < 95 && an.Webhooks.EventsSent > 0 {
		fmt.Fprintf(b, "- Webhook delivery success rate is %s — check %s.\n",
			pct(an.Webhooks.SuccessRate), an.Webhooks.MostFailedEndpoint)
		any = true
	}
	if !any {
		b.WriteString("- Metrics look healthy for this period. Keep monitoring churn and renewals.\n")
	}
}

// generate_dunning_report

func (r *Report) dunningReportTool() mcp.Tool {
	return mcp.NewTool("generate_dunning_report",
		mcp.WithDescription("Generate a Markdown-formatted dunning report: failed invoices, at-risk customers, and recommended next steps for revenue recovery. Ideal for finance or success teams."),
		mcp.WithToolAnnotation(readOnlyAnnotations("Generate dunning report")),
		mcp.WithString("from", mcp.Description("Start of reporting period (YYYY-MM-DD). Defaults to start of current month.")),
		mcp.WithString("to", mcp.Description("End of reporting period (YYYY-MM-DD). Defaults to end of current month.")),
	)
}

func (r *Report) handleDunningReport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	a := &Analytics{Engine: r.Engine}
	an, err := a.fetchAnalytics(ctx, req.GetString("from", ""), req.GetString("to", ""))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var b strings.Builder
	currency := an.Period.Currency

	fmt.Fprintf(&b, "# Dunning Report\n\n")
	fmt.Fprintf(&b, "**Period:** %s → %s\n\n", an.Period.From, an.Period.To)

	fmt.Fprintf(&b, "## Summary\n\n")
	fmt.Fprintf(&b, "- **Failed payments:** %d (%s value)\n",
		an.Payments.FailedPayments, money(currency, an.Payments.TotalFailedAmount))
	fmt.Fprintf(&b, "- **Payment failure rate:** %s\n", pct(an.Summary.PaymentFailureRate))
	fmt.Fprintf(&b, "- **Past-due subscriptions:** %d\n", an.Summary.PastDueSubscriptions)
	fmt.Fprintf(&b, "- **At-risk customers:** %d\n\n", len(an.AtRiskCustomers))
	if an.Payments.MostCommonFailureReason != "" {
		fmt.Fprintf(&b, "**Top failure reason:** %s\n\n", an.Payments.MostCommonFailureReason)
	}

	if len(an.RecentFailedInvoices) > 0 {
		fmt.Fprintf(&b, "## Recent failed invoices\n\n")
		fmt.Fprintf(&b, "| Invoice | Customer | Plan | Amount | Method | Failed at | Reason |\n")
		fmt.Fprintf(&b, "|---|---|---|---:|---|---|---|\n")
		for _, inv := range truncate(an.RecentFailedInvoices, 20) {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n",
				inv.InvoiceKey, inv.CustomerEmail, inv.PlanName,
				money(inv.Currency, inv.AmountDue), inv.PaymentMethod, inv.FailedAt, inv.FailureReason)
		}
		b.WriteString("\n")
	}

	if len(an.AtRiskCustomers) > 0 {
		fmt.Fprintf(&b, "## At-risk customers\n\n")
		fmt.Fprintf(&b, "| Customer | Plan | Reason | Amount at risk | Method |\n")
		fmt.Fprintf(&b, "|---|---|---|---:|---|\n")
		for _, c := range truncate(an.AtRiskCustomers, 20) {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				c.Email, c.PlanName, c.RiskReason, money(currency, c.AmountAtRisk), c.PaymentMethod)
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "## Recommended actions\n\n")
	if an.Summary.PastDueSubscriptions == 0 && len(an.AtRiskCustomers) == 0 {
		b.WriteString("- No past-due invoices or at-risk customers in this period. Dunning queue is clear.\n")
	} else {
		if an.Summary.PastDueSubscriptions > 0 {
			fmt.Fprintf(&b, "- Retry the %d past-due invoices via `retry_payment`.\n", an.Summary.PastDueSubscriptions)
			b.WriteString("- Send `send_dunning_reminder` for invoices older than 3 days with no attempt.\n")
		}
		if len(an.AtRiskCustomers) > 0 {
			b.WriteString("- Contact at-risk customers with a checkout link (`create_portal_link`).\n")
		}
		if an.Payments.MostCommonFailureReason != "" {
			fmt.Fprintf(&b, "- Investigate the recurring failure reason: %s.\n", an.Payments.MostCommonFailureReason)
		}
	}

	return mcp.NewToolResultText(b.String()), nil
}
