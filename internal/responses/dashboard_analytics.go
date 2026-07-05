package responses

type DashboardAnalytics struct {
	Period               AnalyticsPeriod        `json:"period"`
	TopCards             DashboardTopCards      `json:"top_cards"`
	Summary              AnalyticsSummary       `json:"summary"`
	Revenue              AnalyticsRevenue       `json:"revenue"`
	Subscriptions        AnalyticsSubscriptions `json:"subscriptions"`
	Payments             AnalyticsPayments      `json:"payments"`
	Plans                AnalyticsPlans         `json:"plans"`
	RevenueTrend         RevenueTrend           `json:"revenue_trend"`
	UpcomingRenewals     UpcomingRenewals       `json:"upcoming_renewals"`
	DirectDebitMandates  DirectDebitMandates    `json:"direct_debit_mandates"`
	Webhooks             WebhookAnalytics       `json:"webhooks"`
	RecentFailedInvoices []RecentFailedInvoice  `json:"recent_failed_invoices"`
	AtRiskCustomers      []AtRiskCustomer       `json:"at_risk_customers"`
	TopCustomers         []TopCustomer          `json:"top_customers"`
	AIInsights           []DashboardInsight     `json:"ai_insights"`
}

type DashboardTopCards struct {
	MonthlyRecurringRevenue AmountMetric  `json:"monthly_recurring_revenue"`
	TotalRevenue            RevenueMetric `json:"total_revenue"`
	ActiveSubscriptions     CountMetric   `json:"active_subscriptions"`
	TotalCustomers          CountMetric   `json:"total_customers"`
	ARPU                    AmountMetric  `json:"arpu"`
	PaymentSuccessRate      RateMetric    `json:"payment_success_rate"`
	OutstandingInvoices     AmountMetric  `json:"outstanding_invoices"`
}

type AmountMetric struct {
	Amount int64  `json:"amount"`
	Helper string `json:"helper"`
}

type RevenueMetric struct {
	Amount           int64  `json:"amount"`
	Last30DaysAmount int64  `json:"last_30_days_amount"`
	Helper           string `json:"helper"`
}

type CountMetric struct {
	Count  int64  `json:"count"`
	Total  int64  `json:"total"`
	Helper string `json:"helper"`
}

type RateMetric struct {
	Rate       float64 `json:"rate"`
	Successful int64   `json:"successful"`
	Failed     int64   `json:"failed"`
	Helper     string  `json:"helper"`
}

type AnalyticsPeriod struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Currency string `json:"currency"`
	Timezone string `json:"timezone"`
}

type AnalyticsSummary struct {
	GrossRevenue          int64   `json:"gross_revenue"`
	NetRevenue            int64   `json:"net_revenue"`
	PlatformFees          int64   `json:"platform_fees"`
	MRR                   int64   `json:"mrr"`
	ARR                   int64   `json:"arr"`
	RevenueGrowthPercent  float64 `json:"revenue_growth_percent"`
	ActiveSubscriptions   int64   `json:"active_subscriptions"`
	NewSubscriptions      int64   `json:"new_subscriptions"`
	CanceledSubscriptions int64   `json:"canceled_subscriptions"`
	PastDueSubscriptions  int64   `json:"past_due_subscriptions"`
	PaymentSuccessRate    float64 `json:"payment_success_rate"`
	PaymentFailureRate    float64 `json:"payment_failure_rate"`
	ChurnRate             float64 `json:"churn_rate"`
	RetentionRate         float64 `json:"retention_rate"`
}

type AnalyticsRevenue struct {
	GrossRevenue                  int64                 `json:"gross_revenue"`
	NetRevenue                    int64                 `json:"net_revenue"`
	PlatformFees                  int64                 `json:"platform_fees"`
	MRR                           int64                 `json:"mrr"`
	ARR                           int64                 `json:"arr"`
	AverageRevenuePerCustomer     int64                 `json:"average_revenue_per_customer"`
	AverageRevenuePerSubscription int64                 `json:"average_revenue_per_subscription"`
	RevenueGrowthPercent          float64               `json:"revenue_growth_percent"`
	PreviousPeriod                PreviousPeriodRevenue `json:"previous_period"`
}

type PreviousPeriodRevenue struct {
	GrossRevenue int64 `json:"gross_revenue"`
	NetRevenue   int64 `json:"net_revenue"`
	PlatformFees int64 `json:"platform_fees"`
	MRR          int64 `json:"mrr"`
}

type AnalyticsSubscriptions struct {
	TotalSubscriptions        int64            `json:"total_subscriptions"`
	Active                    int64            `json:"active"`
	Trialing                  int64            `json:"trialing"`
	PastDue                   int64            `json:"past_due"`
	Paused                    int64            `json:"paused"`
	Canceled                  int64            `json:"canceled"`
	New                       int64            `json:"new"`
	Churned                   int64            `json:"churned"`
	NonRenewing               int64            `json:"non_renewing"`
	SubscriptionGrowthPercent float64          `json:"subscription_growth_percent"`
	ChurnRate                 float64          `json:"churn_rate"`
	RetentionRate             float64          `json:"retention_rate"`
	BreakdownByStatus         []CountBreakdown `json:"breakdown_by_status"`
	ByPaymentSourceType       []CountBreakdown `json:"by_payment_source_type"`
}

type CountBreakdown struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type AnalyticsPayments struct {
	TotalPaymentAttempts    int64                    `json:"total_payment_attempts"`
	SuccessfulPayments      int64                    `json:"successful_payments"`
	FailedPayments          int64                    `json:"failed_payments"`
	SuccessRate             float64                  `json:"success_rate"`
	FailureRate             float64                  `json:"failure_rate"`
	TotalSuccessfulAmount   int64                    `json:"total_successful_amount"`
	TotalFailedAmount       int64                    `json:"total_failed_amount"`
	MostCommonFailureReason string                   `json:"most_common_failure_reason"`
	ByPaymentMethod         []PaymentMethodAnalytics `json:"by_payment_method"`
}

type PaymentMethodAnalytics struct {
	Method           string  `json:"method"`
	Attempts         int64   `json:"attempts"`
	Successful       int64   `json:"successful"`
	Failed           int64   `json:"failed"`
	SuccessRate      float64 `json:"success_rate"`
	FailureRate      float64 `json:"failure_rate"`
	SuccessfulAmount int64   `json:"successful_amount"`
	FailedAmount     int64   `json:"failed_amount"`
}

type AnalyticsPlans struct {
	TotalPlans       int64                `json:"total_plans"`
	TopPlanByRevenue *TopPlanByRevenue    `json:"top_plan_by_revenue"`
	RevenueByPlan    []PlanRevenueSummary `json:"revenue_by_plan"`
}

type TopPlanByRevenue struct {
	PlanKey             string `json:"plan_key"`
	Name                string `json:"name"`
	GrossRevenue        int64  `json:"gross_revenue"`
	ActiveSubscriptions int64  `json:"active_subscriptions"`
}

type PlanRevenueSummary struct {
	PlanKey               string  `json:"plan_key"`
	Name                  string  `json:"name"`
	Interval              string  `json:"interval"`
	Amount                int64   `json:"amount"`
	ActiveSubscriptions   int64   `json:"active_subscriptions"`
	NewSubscriptions      int64   `json:"new_subscriptions"`
	CanceledSubscriptions int64   `json:"canceled_subscriptions"`
	GrossRevenue          int64   `json:"gross_revenue"`
	NetRevenue            int64   `json:"net_revenue"`
	MRR                   int64   `json:"mrr"`
	ChurnRate             float64 `json:"churn_rate"`
	PaymentSuccessRate    float64 `json:"payment_success_rate"`
}

type RevenueTrend struct {
	Interval string              `json:"interval"`
	Data     []RevenueTrendPoint `json:"data"`
}

type RevenueTrendPoint struct {
	Date                  string `json:"date"`
	GrossRevenue          int64  `json:"gross_revenue"`
	NetRevenue            int64  `json:"net_revenue"`
	PlatformFees          int64  `json:"platform_fees"`
	SuccessfulPayments    int64  `json:"successful_payments"`
	FailedPayments        int64  `json:"failed_payments"`
	NewSubscriptions      int64  `json:"new_subscriptions"`
	CanceledSubscriptions int64  `json:"canceled_subscriptions"`
}

type UpcomingRenewals struct {
	Next7Days     RenewalWindow          `json:"next_7_days"`
	Next30Days    RenewalWindow          `json:"next_30_days"`
	Subscriptions []UpcomingSubscription `json:"subscriptions"`
}

type RenewalWindow struct {
	Count                int64 `json:"count"`
	ExpectedGrossRevenue int64 `json:"expected_gross_revenue"`
	ExpectedNetRevenue   int64 `json:"expected_net_revenue"`
}

type UpcomingSubscription struct {
	SubscriptionKey string `json:"subscription_key"`
	CustomerKey     string `json:"customer_key"`
	CustomerEmail   string `json:"customer_email"`
	PlanName        string `json:"plan_name"`
	Amount          int64  `json:"amount"`
	PaymentMethod   string `json:"payment_method"`
	NextBillingAt   string `json:"next_billing_at"`
	Status          string `json:"status"`
}

type DirectDebitMandates struct {
	MandatesCreated            int64            `json:"mandates_created"`
	ActiveMandates             int64            `json:"active_mandates"`
	PendingMandates            int64            `json:"pending_mandates"`
	FailedMandates             int64            `json:"failed_mandates"`
	MandateActivationRate      float64          `json:"mandate_activation_rate"`
	AverageActivationTimeHours float64          `json:"average_activation_time_hours"`
	Pending                    []PendingMandate `json:"pending"`
}

type PendingMandate struct {
	CustomerKey      string `json:"customer_key"`
	CustomerEmail    string `json:"customer_email"`
	MandateReference string `json:"mandate_reference"`
	Status           string `json:"status"`
	CreatedAt        string `json:"created_at"`
}

type WebhookAnalytics struct {
	EventsSent             int64                   `json:"events_sent"`
	SuccessfulDeliveries   int64                   `json:"successful_deliveries"`
	FailedDeliveries       int64                   `json:"failed_deliveries"`
	SuccessRate            float64                 `json:"success_rate"`
	PendingRetries         int64                   `json:"pending_retries"`
	MostFailedEndpoint     string                  `json:"most_failed_endpoint"`
	RecentFailedDeliveries []FailedWebhookDelivery `json:"recent_failed_deliveries"`
}

type FailedWebhookDelivery struct {
	EventType      string `json:"event_type"`
	EndpointURL    string `json:"endpoint_url"`
	ResponseStatus int    `json:"response_status"`
	AttemptCount   int    `json:"attempt_count"`
	NextRetryAt    string `json:"next_retry_at"`
}

type RecentFailedInvoice struct {
	InvoiceKey      string `json:"invoice_key"`
	SubscriptionKey string `json:"subscription_key"`
	CustomerKey     string `json:"customer_key"`
	CustomerEmail   string `json:"customer_email"`
	PlanName        string `json:"plan_name"`
	AmountDue       int64  `json:"amount_due"`
	Currency        string `json:"currency"`
	PaymentMethod   string `json:"payment_method"`
	FailureReason   string `json:"failure_reason"`
	FailedAt        string `json:"failed_at"`
	Status          string `json:"status"`
}

type AtRiskCustomer struct {
	CustomerKey     string `json:"customer_key"`
	Email           string `json:"email"`
	SubscriptionKey string `json:"subscription_key"`
	PlanName        string `json:"plan_name"`
	RiskReason      string `json:"risk_reason"`
	AmountAtRisk    int64  `json:"amount_at_risk"`
	PaymentMethod   string `json:"payment_method"`
}

type TopCustomer struct {
	CustomerKey         string `json:"customer_key"`
	Name                string `json:"name"`
	Email               string `json:"email"`
	TotalRevenue        int64  `json:"total_revenue"`
	ActiveSubscriptions int64  `json:"active_subscriptions"`
	InvoiceCount        int64  `json:"invoice_count"`
}

type DashboardInsight struct {
	Type              string `json:"type"`
	Title             string `json:"title"`
	Message           string `json:"message"`
	Metric            string `json:"metric"`
	Severity          string `json:"severity"`
	RecommendedAction string `json:"recommended_action"`
}
