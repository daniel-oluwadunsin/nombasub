package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	nsmcp "github.com/daniel-oluwadunsin/nombasub/internal/mcp"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Analytics struct {
	Engine *nsmcp.EngineClient
}

func (a *Analytics) Register(s *server.MCPServer) {
	s.AddTool(a.computeMetricTool(), a.handleComputeMetric)
	s.AddTool(a.comparePeriodsTool(), a.handleComparePeriods)
	s.AddTool(a.explainMetricChangeTool(), a.handleExplainMetricChange)
}

type metricValue struct {
	Name        string  `json:"metric"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit"`
	Currency    string  `json:"currency,omitempty"`
	AsInt       *int64  `json:"as_int,omitempty"`
	Description string  `json:"description"`
}

const (
	unitMinorUnits = "minor_units"
	unitCount      = "count"
	unitPercent    = "percent"
)

func supportedMetrics() []string {
	return []string{
		"mrr", "arr",
		"gross_revenue", "net_revenue", "revenue", "platform_fees",
		"arpu", "arps",
		"active_subscriptions", "new_subscriptions", "canceled_subscriptions", "past_due_subscriptions", "total_subscriptions",
		"churn_rate", "retention_rate",
		"payment_success_rate", "payment_failure_rate", "revenue_growth_percent",
		"total_payment_attempts", "successful_payments", "failed_payments",
		"total_customers",
	}
}

func extractMetric(name string, an *responses.DashboardAnalytics) (*metricValue, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	currency := an.Period.Currency

	amount := func(v int64, desc string) *metricValue {
		i := v
		return &metricValue{Name: name, Value: float64(v), Unit: unitMinorUnits, Currency: currency, AsInt: &i, Description: desc}
	}
	count := func(v int64, desc string) *metricValue {
		i := v
		return &metricValue{Name: name, Value: float64(v), Unit: unitCount, AsInt: &i, Description: desc}
	}
	pct := func(v float64, desc string) *metricValue {
		return &metricValue{Name: name, Value: v, Unit: unitPercent, Description: desc}
	}

	switch name {
	case "mrr":
		return amount(an.Summary.MRR, "Monthly recurring revenue"), nil
	case "arr":
		return amount(an.Summary.ARR, "Annual recurring revenue"), nil
	case "gross_revenue", "revenue":
		return amount(an.Summary.GrossRevenue, "Gross revenue for the period"), nil
	case "net_revenue":
		return amount(an.Summary.NetRevenue, "Net revenue after platform fees"), nil
	case "platform_fees":
		return amount(an.Summary.PlatformFees, "Platform fees deducted from gross revenue"), nil
	case "arpu":
		return amount(an.Revenue.AverageRevenuePerCustomer, "Average revenue per customer"), nil
	case "arps":
		return amount(an.Revenue.AverageRevenuePerSubscription, "Average revenue per active subscription"), nil
	case "active_subscriptions":
		return count(an.Summary.ActiveSubscriptions, "Currently active subscriptions"), nil
	case "new_subscriptions":
		return count(an.Summary.NewSubscriptions, "Subscriptions created during the period"), nil
	case "canceled_subscriptions":
		return count(an.Summary.CanceledSubscriptions, "Subscriptions canceled during the period"), nil
	case "past_due_subscriptions":
		return count(an.Summary.PastDueSubscriptions, "Subscriptions currently in past-due state"), nil
	case "total_subscriptions":
		return count(an.Subscriptions.TotalSubscriptions, "Total subscriptions on record"), nil
	case "total_customers":
		return count(an.TopCards.TotalCustomers.Count, "Total customers on record"), nil
	case "total_payment_attempts":
		return count(an.Payments.TotalPaymentAttempts, "Total payment attempts in the period"), nil
	case "successful_payments":
		return count(an.Payments.SuccessfulPayments, "Successful payments in the period"), nil
	case "failed_payments":
		return count(an.Payments.FailedPayments, "Failed payments in the period"), nil
	case "churn_rate":
		return pct(an.Summary.ChurnRate, "Percentage of active subscriptions that churned"), nil
	case "retention_rate":
		return pct(an.Summary.RetentionRate, "Percentage of subscriptions retained"), nil
	case "payment_success_rate":
		return pct(an.Summary.PaymentSuccessRate, "Percentage of payment attempts that succeeded"), nil
	case "payment_failure_rate":
		return pct(an.Summary.PaymentFailureRate, "Percentage of payment attempts that failed"), nil
	case "revenue_growth_percent":
		return pct(an.Summary.RevenueGrowthPercent, "Revenue change vs. previous period"), nil
	}

	return nil, fmt.Errorf("unknown metric %q; supported: %s", name, strings.Join(supportedMetrics(), ", "))
}

func (a *Analytics) fetchAnalytics(ctx context.Context, from, to string) (*responses.DashboardAnalytics, error) {
	query := map[string]string{}
	if from != "" {
		query["from"] = from
	}
	if to != "" {
		query["to"] = to
	}
	raw, err := a.Engine.Get(ctx, "/v1/dashboard/analytics", query)
	if err != nil {
		return nil, err
	}
	var out responses.DashboardAnalytics
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode analytics response: %w", err)
	}
	return &out, nil
}

// compute_metric

func (a *Analytics) computeMetricTool() mcp.Tool {
	return mcp.NewTool("compute_metric",
		mcp.WithDescription("Compute a single business metric (MRR, ARR, revenue, churn rate, etc.) for the current merchant over an optional date range. Returns the numeric value with unit and currency."),
		mcp.WithToolAnnotation(readOnlyAnnotations("Compute metric")),
		mcp.WithString("metric", mcp.Required(), mcp.Description("Metric name. Supported: "+strings.Join(supportedMetrics(), ", "))),
		mcp.WithString("from", mcp.Description("Start of period (YYYY-MM-DD). Defaults to the start of the current month.")),
		mcp.WithString("to", mcp.Description("End of period (YYYY-MM-DD). Defaults to end of the current month.")),
	)
}

func (a *Analytics) handleComputeMetric(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	metric, err := req.RequireString("metric")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	from := req.GetString("from", "")
	to := req.GetString("to", "")

	an, err := a.fetchAnalytics(ctx, from, to)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	value, err := extractMetric(metric, an)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	out := map[string]any{
		"metric":      value.Name,
		"value":       value.Value,
		"unit":        value.Unit,
		"currency":    value.Currency,
		"description": value.Description,
		"period": map[string]string{
			"from": an.Period.From,
			"to":   an.Period.To,
		},
	}
	if value.AsInt != nil {
		out["as_int"] = *value.AsInt
	}
	pretty, _ := json.MarshalIndent(out, "", "  ")
	return mcp.NewToolResultText(string(pretty)), nil
}

// compare_periods

var periodPresets = []string{"this_month", "last_month", "last_7_days", "last_30_days", "last_90_days", "this_year", "last_year"}

func resolvePeriod(preset string) (from, to string, ok bool) {
	now := time.Now().UTC()
	fmtDate := func(t time.Time) string { return t.Format("2006-01-02") }
	firstOfMonth := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case "this_month":
		start := firstOfMonth(now)
		end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
		return fmtDate(start), fmtDate(end), true
	case "last_month":
		startThis := firstOfMonth(now)
		start := startThis.AddDate(0, -1, 0)
		end := startThis.Add(-time.Nanosecond)
		return fmtDate(start), fmtDate(end), true
	case "last_7_days":
		return fmtDate(now.AddDate(0, 0, -7)), fmtDate(now), true
	case "last_30_days":
		return fmtDate(now.AddDate(0, 0, -30)), fmtDate(now), true
	case "last_90_days":
		return fmtDate(now.AddDate(0, 0, -90)), fmtDate(now), true
	case "this_year":
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		return fmtDate(start), fmtDate(now), true
	case "last_year":
		start := time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(now.Year()-1, 12, 31, 0, 0, 0, 0, time.UTC)
		return fmtDate(start), fmtDate(end), true
	}
	return "", "", false
}

func (a *Analytics) comparePeriodsTool() mcp.Tool {
	return mcp.NewTool("compare_periods",
		mcp.WithDescription("Compare a metric across two time periods. Returns current value, previous value, absolute delta, and percent change. Periods can be preset names or explicit date ranges."),
		mcp.WithToolAnnotation(readOnlyAnnotations("Compare periods")),
		mcp.WithString("metric", mcp.Required(), mcp.Description("Metric name. Supported: "+strings.Join(supportedMetrics(), ", "))),
		mcp.WithString("current", mcp.Description("Preset for the current window: "+strings.Join(periodPresets, ", ")+". Ignored if current_from/current_to are set.")),
		mcp.WithString("previous", mcp.Description("Preset for the previous window. Ignored if previous_from/previous_to are set.")),
		mcp.WithString("current_from", mcp.Description("Explicit start of current period (YYYY-MM-DD)")),
		mcp.WithString("current_to", mcp.Description("Explicit end of current period (YYYY-MM-DD)")),
		mcp.WithString("previous_from", mcp.Description("Explicit start of previous period (YYYY-MM-DD)")),
		mcp.WithString("previous_to", mcp.Description("Explicit end of previous period (YYYY-MM-DD)")),
	)
}

func (a *Analytics) handleComparePeriods(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	metric, err := req.RequireString("metric")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	curFrom := req.GetString("current_from", "")
	curTo := req.GetString("current_to", "")
	prevFrom := req.GetString("previous_from", "")
	prevTo := req.GetString("previous_to", "")

	if curFrom == "" || curTo == "" {
		if preset := req.GetString("current", "this_month"); preset != "" {
			if f, t, ok := resolvePeriod(preset); ok {
				curFrom, curTo = f, t
			} else {
				return mcp.NewToolResultError(fmt.Sprintf("unknown current preset %q", preset)), nil
			}
		}
	}
	if prevFrom == "" || prevTo == "" {
		if preset := req.GetString("previous", "last_month"); preset != "" {
			if f, t, ok := resolvePeriod(preset); ok {
				prevFrom, prevTo = f, t
			} else {
				return mcp.NewToolResultError(fmt.Sprintf("unknown previous preset %q", preset)), nil
			}
		}
	}

	curAn, err := a.fetchAnalytics(ctx, curFrom, curTo)
	if err != nil {
		return mcp.NewToolResultError("current period: " + err.Error()), nil
	}
	prevAn, err := a.fetchAnalytics(ctx, prevFrom, prevTo)
	if err != nil {
		return mcp.NewToolResultError("previous period: " + err.Error()), nil
	}
	curVal, err := extractMetric(metric, curAn)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	prevVal, err := extractMetric(metric, prevAn)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	delta := curVal.Value - prevVal.Value
	var changePct float64
	if prevVal.Value != 0 {
		changePct = round2((delta / math.Abs(prevVal.Value)) * 100)
	}

	out := map[string]any{
		"metric":         curVal.Name,
		"unit":           curVal.Unit,
		"currency":       curVal.Currency,
		"current":        map[string]any{"value": curVal.Value, "from": curAn.Period.From, "to": curAn.Period.To},
		"previous":       map[string]any{"value": prevVal.Value, "from": prevAn.Period.From, "to": prevAn.Period.To},
		"absolute_delta": round2(delta),
		"percent_change": changePct,
		"direction":      direction(delta),
	}
	pretty, _ := json.MarshalIndent(out, "", "  ")
	return mcp.NewToolResultText(string(pretty)), nil
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func direction(delta float64) string {
	switch {
	case delta > 0:
		return "up"
	case delta < 0:
		return "down"
	default:
		return "flat"
	}
}

// explain_metric_change

func (a *Analytics) explainMetricChangeTool() mcp.Tool {
	return mcp.NewTool("explain_metric_change",
		mcp.WithDescription("Explain what drove a metric's change. Returns the current value, previous-period value, delta, and any dashboard AI insights that mention this metric. Use it to answer questions like 'why did MRR drop' or 'what's happening with churn'."),
		mcp.WithToolAnnotation(readOnlyAnnotations("Explain metric change")),
		mcp.WithString("metric", mcp.Required(), mcp.Description("Metric name. Supported: "+strings.Join(supportedMetrics(), ", "))),
		mcp.WithString("from", mcp.Description("Start of current period (YYYY-MM-DD). Defaults to the start of the current month.")),
		mcp.WithString("to", mcp.Description("End of current period (YYYY-MM-DD). Defaults to the end of the current month.")),
	)
}

func (a *Analytics) handleExplainMetricChange(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	metric, err := req.RequireString("metric")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	from := req.GetString("from", "")
	to := req.GetString("to", "")

	an, err := a.fetchAnalytics(ctx, from, to)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	curVal, err := extractMetric(metric, an)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	prevVal, prevPeriod := previousPeriodMetric(metric, an)

	insights := filterInsights(metric, an.AIInsights)

	var delta float64
	var changePct float64
	if prevVal != nil {
		delta = curVal.Value - *prevVal
		if *prevVal != 0 {
			changePct = round2((delta / math.Abs(*prevVal)) * 100)
		}
	}

	supporting := map[string]any{
		"failed_payments":          an.Summary.PastDueSubscriptions,
		"past_due_subscriptions":   an.Summary.PastDueSubscriptions,
		"canceled_subscriptions":   an.Summary.CanceledSubscriptions,
		"new_subscriptions":        an.Summary.NewSubscriptions,
		"payment_failure_rate":     an.Summary.PaymentFailureRate,
		"most_common_failure":      an.Payments.MostCommonFailureReason,
		"webhook_success_rate":     an.Webhooks.SuccessRate,
		"top_plan_by_revenue":      an.Plans.TopPlanByRevenue,
		"recent_failed_invoices":   truncate(an.RecentFailedInvoices, 5),
		"at_risk_customers":        truncate(an.AtRiskCustomers, 5),
	}

	out := map[string]any{
		"metric":         curVal.Name,
		"description":    curVal.Description,
		"unit":           curVal.Unit,
		"currency":       curVal.Currency,
		"current_value":  curVal.Value,
		"period":         map[string]string{"from": an.Period.From, "to": an.Period.To},
		"insights":       insights,
		"supporting_signals": supporting,
	}
	if prevVal != nil {
		out["previous_value"] = *prevVal
		out["previous_period"] = prevPeriod
		out["absolute_delta"] = round2(delta)
		out["percent_change"] = changePct
		out["direction"] = direction(delta)
	}

	pretty, _ := json.MarshalIndent(out, "", "  ")
	return mcp.NewToolResultText(string(pretty)), nil
}

func previousPeriodMetric(metric string, an *responses.DashboardAnalytics) (*float64, string) {
	m := strings.ToLower(strings.TrimSpace(metric))
	label := "previous period (same length, immediately before current)"
	switch m {
	case "gross_revenue", "revenue":
		v := float64(an.Revenue.PreviousPeriod.GrossRevenue)
		return &v, label
	case "net_revenue":
		v := float64(an.Revenue.PreviousPeriod.NetRevenue)
		return &v, label
	case "platform_fees":
		v := float64(an.Revenue.PreviousPeriod.PlatformFees)
		return &v, label
	case "mrr":
		v := float64(an.Revenue.PreviousPeriod.MRR)
		return &v, label
	}
	return nil, ""
}

func filterInsights(metric string, insights []responses.DashboardInsight) []responses.DashboardInsight {
	m := strings.ToLower(strings.TrimSpace(metric))
	filtered := make([]responses.DashboardInsight, 0, len(insights))
	for _, ins := range insights {
		if strings.EqualFold(ins.Metric, m) || strings.Contains(strings.ToLower(ins.Message), m) {
			filtered = append(filtered, ins)
		}
	}
	if len(filtered) == 0 {
		return insights
	}
	return filtered
}

func truncate[T any](items []T, max int) []T {
	if len(items) <= max {
		return items
	}
	return items[:max]
}
