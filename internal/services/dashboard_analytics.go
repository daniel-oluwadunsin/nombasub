package services

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
)

const dashboardTimezone = "Africa/Lagos"

type DashboardAnalyticsService struct {
	rc *repositories.Container
}

func NewDashboardAnalyticsService(rc *repositories.Container) *DashboardAnalyticsService {
	return &DashboardAnalyticsService{rc: rc}
}

func (s *DashboardAnalyticsService) GetAnalytics(tenantId string, requestedFrom, requestedTo *time.Time) (*responses.DashboardAnalytics, error) {
	location, err := time.LoadLocation(dashboardTimezone)
	if err != nil {
		location = time.Local
	}

	now := time.Now().In(location)
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, location)
	to := from.AddDate(0, 1, 0).Add(-time.Nanosecond)
	if requestedFrom != nil {
		from = time.Date(requestedFrom.Year(), requestedFrom.Month(), requestedFrom.Day(), 0, 0, 0, 0, location)
	}
	if requestedTo != nil {
		to = time.Date(requestedTo.Year(), requestedTo.Month(), requestedTo.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), location)
	}
	if to.Before(from) {
		return nil, responses.BadRequest("to must be greater than or equal to from")
	}
	last30From := now.AddDate(0, 0, -30)
	periodLength := to.Sub(from)
	previousFrom := from.Add(-periodLength)
	previousTo := from.Add(-time.Nanosecond)

	var invoices []models.Invoice
	if err := s.rc.DB.Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantId, from, to).Find(&invoices).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	var previousInvoices []models.Invoice
	if err := s.rc.DB.Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantId, previousFrom, previousTo).Find(&previousInvoices).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	var allInvoices []models.Invoice
	if err := s.rc.DB.Where("tenant_id = ?", tenantId).Find(&allInvoices).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	var customers []models.Customer
	if err := s.rc.DB.Where("tenant_id = ?", tenantId).Find(&customers).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	var subscriptions []models.Subscription
	if err := s.rc.DB.Preload("Customer").Preload("Plan").Where("tenant_id = ?", tenantId).Find(&subscriptions).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	var plans []models.Plan
	if err := s.rc.DB.Where("tenant_id = ?", tenantId).Find(&plans).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	var payments []models.PaymentIntent
	if err := s.rc.DB.Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantId, from, to).Find(&payments).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	var paymentSources []models.PaymentSource
	if err := s.rc.DB.Preload("Customer").Where("tenant_id = ?", tenantId).Find(&paymentSources).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	var webhooks []models.WebhookDelivery
	if err := s.rc.DB.Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantId, from, to).Find(&webhooks).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	currency := resolveCurrency(invoices, subscriptions, plans)
	grossRevenue := sumPaidInvoices(invoices)
	previousGrossRevenue := sumPaidInvoices(previousInvoices)
	totalRevenue := grossRevenue
	last30Revenue := sumPaidInvoices(filterInvoicesByCreatedAt(allInvoices, last30From, now))
	platformFees := fee(grossRevenue)
	netRevenue := grossRevenue - platformFees
	previousPlatformFees := fee(previousGrossRevenue)
	previousNetRevenue := previousGrossRevenue - previousPlatformFees
	mrr := calculateMRR(subscriptions)
	revenueGrowth := growthPercent(grossRevenue, previousGrossRevenue)

	subscriptionStats := buildSubscriptionStats(subscriptions, from, to, previousFrom, previousTo)
	subscriptionStats.ByPaymentSourceType = buildSubscriptionPaymentSourceBreakdown(subscriptions)
	paymentStats := buildPaymentStats(payments)
	planStats := buildPlanStats(plans, subscriptions, payments)
	trend := buildRevenueTrend(from, to, invoices, payments, subscriptions)
	renewals := buildUpcomingRenewals(subscriptions, now)
	mandates := buildMandates(paymentSources)
	webhookStats := buildWebhookStats(webhooks)
	failedInvoices, atRisk := buildFailedInvoiceViews(invoices, subscriptions)
	outstandingInvoiceCount, outstandingInvoiceAmount := outstandingInvoices(invoices)
	topCustomers := buildTopCustomers(customers, subscriptions, invoices)
	insights := buildInsights(revenueGrowth, subscriptionStats, paymentStats, planStats, webhookStats)

	customerCount := int64(len(customers))
	newCustomersLast30 := countNewCustomers(customers, last30From, now)
	activeSubscriptionCount := maxInt64(subscriptionStats.Active, 1)
	arpu := divideInt64(totalRevenue, activeSubscriptionCount)

	return &responses.DashboardAnalytics{
		Period: responses.AnalyticsPeriod{
			From:     from.Format("2006-01-02"),
			To:       to.Format("2006-01-02"),
			Currency: currency,
			Timezone: dashboardTimezone,
		},
		TopCards: responses.DashboardTopCards{
			MonthlyRecurringRevenue: responses.AmountMetric{
				Amount: mrr,
				Helper: "MRR across all active plans",
			},
			TotalRevenue: responses.RevenueMetric{
				Amount:           totalRevenue,
				Last30DaysAmount: last30Revenue,
				Helper:           "in last 30 days",
			},
			ActiveSubscriptions: responses.CountMetric{
				Count:  subscriptionStats.Active,
				Total:  subscriptionStats.TotalSubscriptions,
				Helper: "total across all statuses",
			},
			TotalCustomers: responses.CountMetric{
				Count:  customerCount,
				Total:  newCustomersLast30,
				Helper: "new in last 30 days",
			},
			ARPU: responses.AmountMetric{
				Amount: arpu,
				Helper: "Average revenue per active subscriber",
			},
			PaymentSuccessRate: responses.RateMetric{
				Rate:       paymentStats.SuccessRate,
				Successful: paymentStats.SuccessfulPayments,
				Failed:     paymentStats.FailedPayments,
				Helper:     "payment attempts",
			},
			OutstandingInvoices: responses.AmountMetric{
				Amount: outstandingInvoiceAmount,
				Helper: fmt.Sprintf("%d open or failed invoices", outstandingInvoiceCount),
			},
		},
		Summary: responses.AnalyticsSummary{
			GrossRevenue:          grossRevenue,
			NetRevenue:            netRevenue,
			PlatformFees:          platformFees,
			MRR:                   mrr,
			ARR:                   mrr * 12,
			RevenueGrowthPercent:  revenueGrowth,
			ActiveSubscriptions:   subscriptionStats.Active,
			NewSubscriptions:      subscriptionStats.New,
			CanceledSubscriptions: subscriptionStats.Canceled,
			PastDueSubscriptions:  subscriptionStats.PastDue,
			PaymentSuccessRate:    paymentStats.SuccessRate,
			PaymentFailureRate:    paymentStats.FailureRate,
			ChurnRate:             subscriptionStats.ChurnRate,
			RetentionRate:         subscriptionStats.RetentionRate,
		},
		Revenue: responses.AnalyticsRevenue{
			GrossRevenue:                  grossRevenue,
			NetRevenue:                    netRevenue,
			PlatformFees:                  platformFees,
			MRR:                           mrr,
			ARR:                           mrr * 12,
			AverageRevenuePerCustomer:     divideInt64(totalRevenue, maxInt64(customerCount, 1)),
			AverageRevenuePerSubscription: arpu,
			RevenueGrowthPercent:          revenueGrowth,
			PreviousPeriod: responses.PreviousPeriodRevenue{
				GrossRevenue: previousGrossRevenue,
				NetRevenue:   previousNetRevenue,
				PlatformFees: previousPlatformFees,
				MRR:          mrr,
			},
		},
		Subscriptions:        subscriptionStats,
		Payments:             paymentStats,
		Plans:                planStats,
		RevenueTrend:         trend,
		UpcomingRenewals:     renewals,
		DirectDebitMandates:  mandates,
		Webhooks:             webhookStats,
		RecentFailedInvoices: failedInvoices,
		AtRiskCustomers:      atRisk,
		TopCustomers:         topCustomers,
		AIInsights:           insights,
	}, nil
}

func resolveCurrency(invoices []models.Invoice, subscriptions []models.Subscription, plans []models.Plan) string {
	if len(invoices) > 0 && invoices[0].Currency != "" {
		return invoices[0].Currency
	}
	if len(subscriptions) > 0 && subscriptions[0].Currency != "" {
		return subscriptions[0].Currency
	}
	if len(plans) > 0 && plans[0].Currency != "" {
		return plans[0].Currency
	}
	return "NGN"
}

func sumPaidInvoices(invoices []models.Invoice) int64 {
	var total int64
	for _, invoice := range invoices {
		if invoice.Status == models.InvoiceStatusPaid {
			if invoice.AmountPaid > 0 {
				total += invoice.AmountPaid
			} else {
				total += invoice.AmountDue
			}
		}
	}
	return total
}

func filterInvoicesByCreatedAt(invoices []models.Invoice, from, to time.Time) []models.Invoice {
	var filtered []models.Invoice
	for _, invoice := range invoices {
		if between(invoice.CreatedAt, from, to) {
			filtered = append(filtered, invoice)
		}
	}
	return filtered
}

func outstandingInvoices(invoices []models.Invoice) (int64, int64) {
	var count int64
	var amount int64
	for _, invoice := range invoices {
		if invoice.Status == models.InvoiceStatusOpen || invoice.Status == models.InvoiceStatusFailed {
			count++
			if invoice.AmountRemaining > 0 {
				amount += invoice.AmountRemaining
			} else {
				amount += invoice.AmountDue
			}
		}
	}
	return count, amount
}

func fee(amount int64) int64 {
	return int64(math.Round(float64(amount) * 0.05))
}

func growthPercent(current, previous int64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100
		}
		return 0
	}
	return round1((float64(current-previous) / float64(previous)) * 100)
}

func rate(part, total int64) float64 {
	if total == 0 {
		return 0
	}
	return round1((float64(part) / float64(total)) * 100)
}

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}

func divideInt64(value, divisor int64) int64 {
	if divisor == 0 {
		return 0
	}
	return value / divisor
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func calculateMRR(subscriptions []models.Subscription) int64 {
	var mrr int64
	for _, subscription := range subscriptions {
		if subscription.Status != models.SubscriptionStatusActive {
			continue
		}
		switch subscription.Interval {
		case models.PlanIntervalMonthly:
			mrr += subscription.Amount
		case models.PlanIntervalYearly:
			mrr += subscription.Amount / 12
		case models.PlanIntervalQuarterly:
			mrr += subscription.Amount / 3
		case models.PlanIntervalWeekly:
			mrr += subscription.Amount * 4
		case models.PlanIntervalBiWeekly:
			mrr += subscription.Amount * 2
		case models.PlanIntervalDaily:
			mrr += subscription.Amount * 30
		default:
			mrr += subscription.Amount
		}
	}
	return mrr
}

func buildSubscriptionStats(subscriptions []models.Subscription, from, to, previousFrom, previousTo time.Time) responses.AnalyticsSubscriptions {
	var stats responses.AnalyticsSubscriptions
	stats.TotalSubscriptions = int64(len(subscriptions))

	var previousNew int64
	statusCounts := map[string]int64{}
	for _, subscription := range subscriptions {
		statusCounts[string(subscription.Status)]++
		switch subscription.Status {
		case models.SubscriptionStatusActive:
			stats.Active++
		case models.SubscriptionStatusPastDue, models.SubscriptionStatusAttention:
			stats.PastDue++
		case models.SubscriptionStatusPaused:
			stats.Paused++
		case models.SubscriptionStatusCanceled, models.SubscriptionStatusExpired:
			stats.Canceled++
		}
		if subscription.TrialEndDate != nil && subscription.TrialEndDate.After(time.Now()) {
			stats.Trialing++
		}
		if subscription.CancelledAtEndOfBillingCycle {
			stats.NonRenewing++
		}
		if between(subscription.CreatedAt, from, to) {
			stats.New++
		}
		if between(subscription.CreatedAt, previousFrom, previousTo) {
			previousNew++
		}
		if subscription.CancelledAt != nil && between(*subscription.CancelledAt, from, to) {
			stats.Churned++
		}
	}

	stats.SubscriptionGrowthPercent = growthPercent(stats.New, previousNew)
	stats.ChurnRate = rate(stats.Churned, maxInt64(stats.TotalSubscriptions, 1))
	stats.RetentionRate = round1(100 - stats.ChurnRate)
	stats.BreakdownByStatus = buildOrderedStatusBreakdown(statusCounts)
	return stats
}

func buildOrderedStatusBreakdown(statusCounts map[string]int64) []responses.CountBreakdown {
	order := []string{
		string(models.SubscriptionStatusActive),
		"trialing",
		string(models.SubscriptionStatusPastDue),
		string(models.SubscriptionStatusPaused),
		string(models.SubscriptionStatusAttention),
		string(models.SubscriptionStatusCanceled),
		string(models.SubscriptionStatusExpired),
	}
	breakdown := make([]responses.CountBreakdown, 0, len(statusCounts))
	seen := map[string]bool{}
	for _, status := range order {
		if count := statusCounts[status]; count > 0 {
			breakdown = append(breakdown, responses.CountBreakdown{Name: status, Count: count})
			seen[status] = true
		}
	}
	for status, count := range statusCounts {
		if !seen[status] {
			breakdown = append(breakdown, responses.CountBreakdown{Name: status, Count: count})
		}
	}
	return breakdown
}

func buildSubscriptionPaymentSourceBreakdown(subscriptions []models.Subscription) []responses.CountBreakdown {
	counts := map[string]int64{
		"card": 0,
		"bank": 0,
	}
	for _, subscription := range subscriptions {
		if subscription.PaymentSourceType == nil {
			counts["unknown"]++
			continue
		}
		sourceType := string(*subscription.PaymentSourceType)
		if sourceType == string(models.PaymentSourceTypeBank) {
			sourceType = "bank"
		}
		counts[sourceType]++
	}

	breakdown := []responses.CountBreakdown{}
	for _, sourceType := range []string{"card", "bank", "unknown"} {
		if count := counts[sourceType]; count > 0 || sourceType != "unknown" {
			breakdown = append(breakdown, responses.CountBreakdown{Name: sourceType, Count: count})
		}
	}
	return breakdown
}

func buildPaymentStats(payments []models.PaymentIntent) responses.AnalyticsPayments {
	methods := map[string]*responses.PaymentMethodAnalytics{}
	failureReasons := map[string]int64{}
	var stats responses.AnalyticsPayments

	for _, payment := range payments {
		stats.TotalPaymentAttempts++
		method := "unknown"
		if payment.PaymentSourceType != nil {
			method = string(*payment.PaymentSourceType)
			if method == string(models.PaymentSourceTypeBank) {
				method = "direct_debit"
			}
		}

		if methods[method] == nil {
			methods[method] = &responses.PaymentMethodAnalytics{Method: method}
		}
		methods[method].Attempts++

		switch payment.Status {
		case models.PaymentIntentStatusSuccess:
			stats.SuccessfulPayments++
			stats.TotalSuccessfulAmount += payment.Amount
			methods[method].Successful++
			methods[method].SuccessfulAmount += payment.Amount
		case models.PaymentIntentStatusFailed:
			stats.FailedPayments++
			stats.TotalFailedAmount += payment.Amount
			methods[method].Failed++
			methods[method].FailedAmount += payment.Amount
			if payment.FailureReason != nil && *payment.FailureReason != "" {
				failureReasons[*payment.FailureReason]++
			}
		}
	}

	stats.SuccessRate = rate(stats.SuccessfulPayments, stats.TotalPaymentAttempts)
	stats.FailureRate = rate(stats.FailedPayments, stats.TotalPaymentAttempts)
	stats.MostCommonFailureReason = mostCommon(failureReasons)

	for _, method := range methods {
		method.SuccessRate = rate(method.Successful, method.Attempts)
		method.FailureRate = rate(method.Failed, method.Attempts)
		stats.ByPaymentMethod = append(stats.ByPaymentMethod, *method)
	}
	sort.Slice(stats.ByPaymentMethod, func(i, j int) bool {
		return stats.ByPaymentMethod[i].Attempts > stats.ByPaymentMethod[j].Attempts
	})
	return stats
}

func buildPlanStats(plans []models.Plan, subscriptions []models.Subscription, payments []models.PaymentIntent) responses.AnalyticsPlans {
	stats := responses.AnalyticsPlans{TotalPlans: int64(len(plans))}
	planMap := map[string]models.Plan{}
	revenueByPlan := map[string]int64{}
	paymentsByPlan := map[string][2]int64{}

	for _, plan := range plans {
		planMap[plan.ID] = plan
	}
	for _, payment := range payments {
		if payment.Status == models.PaymentIntentStatusSuccess {
			revenueByPlan[payment.PlanID] += payment.Amount
		}
		counts := paymentsByPlan[payment.PlanID]
		if payment.Status == models.PaymentIntentStatusSuccess {
			counts[0]++
		}
		if payment.Status == models.PaymentIntentStatusFailed {
			counts[1]++
		}
		paymentsByPlan[payment.PlanID] = counts
	}

	for _, plan := range plans {
		summary := responses.PlanRevenueSummary{
			PlanKey:      plan.Code,
			Name:         plan.Name,
			Interval:     string(plan.Interval),
			Amount:       plan.Amount,
			GrossRevenue: revenueByPlan[plan.ID],
		}

		for _, subscription := range subscriptions {
			if subscription.PlanID != plan.ID {
				continue
			}
			if subscription.Status == models.SubscriptionStatusActive {
				summary.ActiveSubscriptions++
			}
			if subscription.Status == models.SubscriptionStatusCanceled || subscription.Status == models.SubscriptionStatusExpired {
				summary.CanceledSubscriptions++
			}
			if subscription.CreatedAt.After(time.Now().AddDate(0, -1, 0)) {
				summary.NewSubscriptions++
			}
		}
		applyPlanRevenueCalculations(&summary)
		summary.ChurnRate = rate(summary.CanceledSubscriptions, maxInt64(summary.ActiveSubscriptions+summary.CanceledSubscriptions, 1))
		counts := paymentsByPlan[plan.ID]
		summary.PaymentSuccessRate = rate(counts[0], counts[0]+counts[1])
		stats.RevenueByPlan = append(stats.RevenueByPlan, summary)
	}

	sort.Slice(stats.RevenueByPlan, func(i, j int) bool {
		return stats.RevenueByPlan[i].GrossRevenue > stats.RevenueByPlan[j].GrossRevenue
	})
	if len(stats.RevenueByPlan) > 0 {
		top := stats.RevenueByPlan[0]
		stats.TopPlanByRevenue = &responses.TopPlanByRevenue{
			PlanKey:             top.PlanKey,
			Name:                top.Name,
			GrossRevenue:        top.GrossRevenue,
			ActiveSubscriptions: top.ActiveSubscriptions,
		}
	}
	_ = planMap
	return stats
}

func applyPlanRevenueCalculations(p *responses.PlanRevenueSummary) {
	p.NetRevenue = p.GrossRevenue - fee(p.GrossRevenue)
	switch p.Interval {
	case string(models.PlanIntervalMonthly):
		p.MRR = p.Amount * p.ActiveSubscriptions
	case string(models.PlanIntervalYearly):
		p.MRR = (p.Amount * p.ActiveSubscriptions) / 12
	default:
		p.MRR = p.Amount * p.ActiveSubscriptions
	}
}

func buildRevenueTrend(from, to time.Time, invoices []models.Invoice, payments []models.PaymentIntent, subscriptions []models.Subscription) responses.RevenueTrend {
	points := map[string]*responses.RevenueTrendPoint{}
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		points[key] = &responses.RevenueTrendPoint{Date: key}
	}
	for _, invoice := range invoices {
		key := invoice.CreatedAt.Format("2006-01-02")
		point := points[key]
		if point == nil {
			continue
		}
		if invoice.Status == models.InvoiceStatusPaid {
			amount := invoice.AmountPaid
			if amount == 0 {
				amount = invoice.AmountDue
			}
			point.GrossRevenue += amount
			point.PlatformFees += fee(amount)
			point.NetRevenue = point.GrossRevenue - point.PlatformFees
		}
	}
	for _, payment := range payments {
		key := payment.CreatedAt.Format("2006-01-02")
		point := points[key]
		if point == nil {
			continue
		}
		if payment.Status == models.PaymentIntentStatusSuccess {
			point.SuccessfulPayments++
		}
		if payment.Status == models.PaymentIntentStatusFailed {
			point.FailedPayments++
		}
	}
	for _, subscription := range subscriptions {
		key := subscription.CreatedAt.Format("2006-01-02")
		point := points[key]
		if point == nil {
			continue
		}
		point.NewSubscriptions++
		if subscription.CancelledAt != nil {
			cancelKey := subscription.CancelledAt.Format("2006-01-02")
			if cancelPoint := points[cancelKey]; cancelPoint != nil {
				cancelPoint.CanceledSubscriptions++
			}
		}
	}

	var data []responses.RevenueTrendPoint
	for _, point := range points {
		data = append(data, *point)
	}
	sort.Slice(data, func(i, j int) bool { return data[i].Date < data[j].Date })
	return responses.RevenueTrend{Interval: "daily", Data: data}
}

func buildUpcomingRenewals(subscriptions []models.Subscription, now time.Time) responses.UpcomingRenewals {
	var renewals responses.UpcomingRenewals
	for _, subscription := range subscriptions {
		if subscription.CurrentBillingCycleEnd == nil || subscription.Status != models.SubscriptionStatusActive {
			continue
		}
		days := subscription.CurrentBillingCycleEnd.Sub(now).Hours() / 24
		if days < 0 || days > 30 {
			continue
		}
		renewals.Next30Days.Count++
		renewals.Next30Days.ExpectedGrossRevenue += subscription.Amount
		if days <= 7 {
			renewals.Next7Days.Count++
			renewals.Next7Days.ExpectedGrossRevenue += subscription.Amount
		}
		if len(renewals.Subscriptions) < 8 {
			renewals.Subscriptions = append(renewals.Subscriptions, upcomingSubscription(subscription))
		}
	}
	renewals.Next7Days.ExpectedNetRevenue = renewals.Next7Days.ExpectedGrossRevenue - fee(renewals.Next7Days.ExpectedGrossRevenue)
	renewals.Next30Days.ExpectedNetRevenue = renewals.Next30Days.ExpectedGrossRevenue - fee(renewals.Next30Days.ExpectedGrossRevenue)
	return renewals
}

func upcomingSubscription(subscription models.Subscription) responses.UpcomingSubscription {
	customerKey, customerEmail := "", ""
	if subscription.Customer != nil {
		customerKey = subscription.Customer.Code
		customerEmail = subscription.Customer.Email
	}
	planName := ""
	if subscription.Plan != nil {
		planName = subscription.Plan.Name
	}
	paymentMethod := ""
	if subscription.PaymentSourceType != nil {
		paymentMethod = string(*subscription.PaymentSourceType)
	}
	nextBillingAt := ""
	if subscription.CurrentBillingCycleEnd != nil {
		nextBillingAt = subscription.CurrentBillingCycleEnd.Format(time.RFC3339)
	}
	return responses.UpcomingSubscription{
		SubscriptionKey: subscription.Code,
		CustomerKey:     customerKey,
		CustomerEmail:   customerEmail,
		PlanName:        planName,
		Amount:          subscription.Amount,
		PaymentMethod:   paymentMethod,
		NextBillingAt:   nextBillingAt,
		Status:          string(subscription.Status),
	}
}

func buildMandates(paymentSources []models.PaymentSource) responses.DirectDebitMandates {
	var mandates responses.DirectDebitMandates
	for _, source := range paymentSources {
		if source.Type != models.PaymentSourceTypeBank {
			continue
		}
		mandates.MandatesCreated++
		if source.Status == models.PaymentSourceStatusActive {
			mandates.ActiveMandates++
		}
	}
	mandates.MandateActivationRate = rate(mandates.ActiveMandates, mandates.MandatesCreated)
	return mandates
}

func buildWebhookStats(webhooks []models.WebhookDelivery) responses.WebhookAnalytics {
	var stats responses.WebhookAnalytics
	failuresByEndpoint := map[string]int64{}
	for _, webhook := range webhooks {
		stats.EventsSent++
		switch webhook.Status {
		case models.WebhookDeliveryStatusDelivered:
			stats.SuccessfulDeliveries++
		case models.WebhookDeliveryStatusFailed:
			stats.FailedDeliveries++
			failuresByEndpoint[webhook.EndpointURL]++
			if len(stats.RecentFailedDeliveries) < 6 {
				stats.RecentFailedDeliveries = append(stats.RecentFailedDeliveries, responses.FailedWebhookDelivery{
					EventType:    string(webhook.EventType),
					EndpointURL:  webhook.EndpointURL,
					AttemptCount: webhook.AttempsCount,
				})
			}
		case models.WebhookDeliveryStatusPending:
			stats.PendingRetries++
		}
	}
	stats.SuccessRate = rate(stats.SuccessfulDeliveries, stats.EventsSent)
	stats.MostFailedEndpoint = mostCommon(failuresByEndpoint)
	return stats
}

func buildFailedInvoiceViews(invoices []models.Invoice, subscriptions []models.Subscription) ([]responses.RecentFailedInvoice, []responses.AtRiskCustomer) {
	subscriptionMap := map[string]models.Subscription{}
	for _, subscription := range subscriptions {
		subscriptionMap[subscription.ID] = subscription
	}

	var failed []responses.RecentFailedInvoice
	var atRisk []responses.AtRiskCustomer
	for _, invoice := range invoices {
		if invoice.Status != models.InvoiceStatusFailed && invoice.Status != models.InvoiceStatusOpen {
			continue
		}
		subscription := subscriptionMap[invoice.SubscriptionID]
		customerKey, email, planName, paymentMethod := "", "", "", ""
		if subscription.Customer != nil {
			customerKey = subscription.Customer.Code
			email = subscription.Customer.Email
		}
		if subscription.Plan != nil {
			planName = subscription.Plan.Name
		}
		if subscription.PaymentSourceType != nil {
			paymentMethod = string(*subscription.PaymentSourceType)
		}
		failedAt := ""
		if invoice.FailedAt != nil {
			failedAt = invoice.FailedAt.Format(time.RFC3339)
		}
		reason := ""
		if invoice.FailureReason != nil {
			reason = *invoice.FailureReason
		}
		if len(failed) < 8 {
			failed = append(failed, responses.RecentFailedInvoice{
				InvoiceKey:      invoice.Code,
				SubscriptionKey: subscription.Code,
				CustomerKey:     customerKey,
				CustomerEmail:   email,
				PlanName:        planName,
				AmountDue:       invoice.AmountDue,
				Currency:        invoice.Currency,
				PaymentMethod:   paymentMethod,
				FailureReason:   reason,
				FailedAt:        failedAt,
				Status:          string(invoice.Status),
			})
		}
		if email != "" && len(atRisk) < 8 {
			atRisk = append(atRisk, responses.AtRiskCustomer{
				CustomerKey:     customerKey,
				Email:           email,
				SubscriptionKey: subscription.Code,
				PlanName:        planName,
				RiskReason:      "payment_failed",
				AmountAtRisk:    invoice.AmountDue,
				PaymentMethod:   paymentMethod,
			})
		}
	}
	return failed, atRisk
}

func buildTopCustomers(customers []models.Customer, subscriptions []models.Subscription, invoices []models.Invoice) []responses.TopCustomer {
	customerMap := map[string]models.Customer{}
	topByCustomer := map[string]*responses.TopCustomer{}

	for _, customer := range customers {
		customerMap[customer.ID] = customer
		name := ""
		if customer.Name != nil {
			name = *customer.Name
		}
		topByCustomer[customer.ID] = &responses.TopCustomer{
			CustomerKey: customer.Code,
			Name:        name,
			Email:       customer.Email,
		}
	}

	for _, subscription := range subscriptions {
		topCustomer := topByCustomer[subscription.CustomerID]
		if topCustomer == nil {
			continue
		}
		if subscription.Status == models.SubscriptionStatusActive {
			topCustomer.ActiveSubscriptions++
		}
	}

	for _, invoice := range invoices {
		if invoice.Status != models.InvoiceStatusPaid {
			continue
		}
		topCustomer := topByCustomer[invoice.CustomerID]
		if topCustomer == nil {
			if customer, ok := customerMap[invoice.CustomerID]; ok {
				name := ""
				if customer.Name != nil {
					name = *customer.Name
				}
				topCustomer = &responses.TopCustomer{
					CustomerKey: customer.Code,
					Name:        name,
					Email:       customer.Email,
				}
				topByCustomer[invoice.CustomerID] = topCustomer
			} else {
				continue
			}
		}
		amount := invoice.AmountPaid
		if amount == 0 {
			amount = invoice.AmountDue
		}
		topCustomer.TotalRevenue += amount
		topCustomer.InvoiceCount++
	}

	var topCustomers []responses.TopCustomer
	for _, customer := range topByCustomer {
		if customer.TotalRevenue > 0 || customer.ActiveSubscriptions > 0 {
			topCustomers = append(topCustomers, *customer)
		}
	}
	sort.Slice(topCustomers, func(i, j int) bool {
		if topCustomers[i].TotalRevenue == topCustomers[j].TotalRevenue {
			return topCustomers[i].ActiveSubscriptions > topCustomers[j].ActiveSubscriptions
		}
		return topCustomers[i].TotalRevenue > topCustomers[j].TotalRevenue
	})
	if len(topCustomers) > 8 {
		topCustomers = topCustomers[:8]
	}
	return topCustomers
}

func buildInsights(revenueGrowth float64, subscriptions responses.AnalyticsSubscriptions, payments responses.AnalyticsPayments, plans responses.AnalyticsPlans, webhooks responses.WebhookAnalytics) []responses.DashboardInsight {
	var insights []responses.DashboardInsight
	if revenueGrowth > 0 {
		insights = append(insights, responses.DashboardInsight{
			Type:              "positive",
			Title:             "Revenue is growing",
			Message:           fmt.Sprintf("Revenue increased by %.1f%% compared to the previous period.", revenueGrowth),
			Metric:            "revenue_growth_percent",
			Severity:          "low",
			RecommendedAction: "Review the plans and channels contributing most to the growth.",
		})
	} else if revenueGrowth < 0 {
		insights = append(insights, responses.DashboardInsight{
			Type:              "warning",
			Title:             "Revenue declined this period",
			Message:           fmt.Sprintf("Revenue changed by %.1f%% compared to the previous period.", revenueGrowth),
			Metric:            "revenue_growth_percent",
			Severity:          "medium",
			RecommendedAction: "Review failed payments, churn, and upcoming renewals for recovery opportunities.",
		})
	}

	var card, debit *responses.PaymentMethodAnalytics
	for i := range payments.ByPaymentMethod {
		method := &payments.ByPaymentMethod[i]
		if method.Method == "card" {
			card = method
		}
		if method.Method == "direct_debit" || method.Method == "bank" {
			debit = method
		}
	}
	if card != nil && debit != nil && card.SuccessRate < debit.SuccessRate {
		insights = append(insights, responses.DashboardInsight{
			Type:              "opportunity",
			Title:             "Direct debit performs better",
			Message:           fmt.Sprintf("Direct debit has a %.1f%% success rate compared to %.1f%% for cards.", debit.SuccessRate, card.SuccessRate),
			Metric:            "payment_success_rate",
			Severity:          "medium",
			RecommendedAction: "Encourage customers with repeated card failures to switch to direct debit.",
		})
	}
	if subscriptions.PastDue > 0 {
		insights = append(insights, responses.DashboardInsight{
			Type:              "warning",
			Title:             "Past due subscriptions need attention",
			Message:           fmt.Sprintf("There are %d past due subscriptions with potential revenue at risk.", subscriptions.PastDue),
			Metric:            "past_due_subscriptions",
			Severity:          "high",
			RecommendedAction: "Send customer portal links so users can retry payment or update their payment method.",
		})
	}
	if plans.TopPlanByRevenue != nil && plans.TopPlanByRevenue.GrossRevenue > 0 {
		insights = append(insights, responses.DashboardInsight{
			Type:              "opportunity",
			Title:             fmt.Sprintf("%s is your strongest plan", plans.TopPlanByRevenue.Name),
			Message:           fmt.Sprintf("%s generated %d and has %d active subscriptions.", plans.TopPlanByRevenue.Name, plans.TopPlanByRevenue.GrossRevenue, plans.TopPlanByRevenue.ActiveSubscriptions),
			Metric:            "revenue_by_plan",
			Severity:          "low",
			RecommendedAction: "Promote this plan more aggressively or create an annual version.",
		})
	}
	if webhooks.EventsSent > 0 && webhooks.SuccessRate < 95 {
		insights = append(insights, responses.DashboardInsight{
			Type:              "warning",
			Title:             "Webhook delivery reliability is below target",
			Message:           fmt.Sprintf("Webhook success rate is %.1f%% this period.", webhooks.SuccessRate),
			Metric:            "webhook_success_rate",
			Severity:          "medium",
			RecommendedAction: "Check the merchant webhook endpoint reliability and retry response handling.",
		})
	}
	return insights
}

func between(value, from, to time.Time) bool {
	return !value.Before(from) && !value.After(to)
}

func uniqueCustomerCount(subscriptions []models.Subscription) int64 {
	seen := map[string]bool{}
	for _, subscription := range subscriptions {
		if subscription.CustomerID != "" {
			seen[subscription.CustomerID] = true
		}
	}
	return int64(len(seen))
}

func countNewCustomers(customers []models.Customer, from, to time.Time) int64 {
	var count int64
	for _, customer := range customers {
		if between(customer.CreatedAt, from, to) {
			count++
		}
	}
	return count
}

func mostCommon(values map[string]int64) string {
	var selected string
	var count int64
	for value, valueCount := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if valueCount > count {
			selected = value
			count = valueCount
		}
	}
	return selected
}
