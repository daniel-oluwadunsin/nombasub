package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	nsmcp "github.com/daniel-oluwadunsin/nombasub/internal/mcp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Resources struct {
	Engine *nsmcp.EngineClient
}

func (r *Resources) Register(s *server.MCPServer) {
	s.AddResource(mcp.NewResource(
		"plans://all",
		"All plans",
		mcp.WithResourceDescription("List of every billing plan on the merchant account."),
		mcp.WithMIMEType("application/json"),
	), r.readPlans)

	s.AddResource(mcp.NewResource(
		"subscriptions://active",
		"Active subscriptions",
		mcp.WithResourceDescription("All subscriptions currently in the active state."),
		mcp.WithMIMEType("application/json"),
	), r.readActiveSubscriptions)

	s.AddResource(mcp.NewResource(
		"analytics://summary",
		"Dashboard analytics summary",
		mcp.WithResourceDescription("Full dashboard analytics response for the current billing month."),
		mcp.WithMIMEType("application/json"),
	), r.readAnalyticsSummary)

	s.AddResource(mcp.NewResource(
		"events://recent",
		"Recent webhook deliveries",
		mcp.WithResourceDescription("Most recent outbound webhook deliveries sent to the merchant."),
		mcp.WithMIMEType("application/json"),
	), r.readRecentEvents)

	s.AddResourceTemplate(mcp.NewResourceTemplate(
		"customers://{emailOrCode}",
		"Customer detail",
		mcp.WithTemplateDescription("Full profile, subscriptions, and payment sources for a specific customer, by email or customer code."),
		mcp.WithTemplateMIMEType("application/json"),
	), r.readCustomer)
}

func jsonResource(uri string, data json.RawMessage) []mcp.ResourceContents {
	pretty, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		pretty = data
	}
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(pretty),
		},
	}
}

func errorResource(uri string, err error) ([]mcp.ResourceContents, error) {
	payload := map[string]string{"error": err.Error()}
	body, _ := json.MarshalIndent(payload, "", "  ")
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(body),
		},
	}, nil
}

func (r *Resources) readPlans(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	data, err := r.Engine.Get(ctx, "/v1/plan/", map[string]string{"limit": "100"})
	if err != nil {
		return errorResource(req.Params.URI, err)
	}
	return jsonResource(req.Params.URI, data), nil
}

func (r *Resources) readActiveSubscriptions(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	data, err := r.Engine.Get(ctx, "/v1/subscription/", map[string]string{"limit": "100"})
	if err != nil {
		return errorResource(req.Params.URI, err)
	}
	return jsonResource(req.Params.URI, data), nil
}

func (r *Resources) readAnalyticsSummary(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	data, err := r.Engine.Get(ctx, "/v1/dashboard/analytics", nil)
	if err != nil {
		return errorResource(req.Params.URI, err)
	}
	return jsonResource(req.Params.URI, data), nil
}

func (r *Resources) readRecentEvents(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	data, err := r.Engine.Get(ctx, "/v1/webhook-deliveries/", map[string]string{"limit": "50"})
	if err != nil {
		return errorResource(req.Params.URI, err)
	}
	return jsonResource(req.Params.URI, data), nil
}

func (r *Resources) readCustomer(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	uri := req.Params.URI
	prefix := "customers://"
	if !strings.HasPrefix(uri, prefix) {
		return errorResource(uri, fmt.Errorf("expected customers://{emailOrCode}"))
	}
	emailOrCode := strings.TrimPrefix(uri, prefix)
	if emailOrCode == "" {
		return errorResource(uri, fmt.Errorf("missing customer identifier"))
	}

	data, err := r.Engine.Get(ctx, "/v1/customer/"+emailOrCode, nil)
	if err != nil {
		return errorResource(uri, err)
	}
	return jsonResource(uri, data), nil
}
