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
	s.AddTool(mcp.NewTool(
		"get_customers",
		mcp.WithDescription("List customers for the current merchant. Supports pagination and date filtering. Returns customer profiles with lifetime value and subscription counts."),
		mcp.WithNumber("limit", mcp.Description("Max customers per page (default 20, max 100)")),
		mcp.WithNumber("page", mcp.Description("Page number, 1-indexed")),
		mcp.WithString("search", mcp.Description("Search by name, email, or customer code")),
		mcp.WithString("from", mcp.Description("Filter customers created on/after this date (YYYY-MM-DD)")),
		mcp.WithString("to", mcp.Description("Filter customers created on/before this date (YYYY-MM-DD)")),
	), q.handleGetCustomers)
}

func (q *Query) handleGetCustomers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := map[string]string{}
	if v := req.GetString("search", ""); v != "" {
		query["search"] = v
	}
	if v := req.GetString("from", ""); v != "" {
		query["from"] = v
	}
	if v := req.GetString("to", ""); v != "" {
		query["to"] = v
	}
	if v := req.GetInt("limit", 0); v > 0 {
		query["limit"] = fmt.Sprintf("%d", v)
	}
	if v := req.GetInt("page", 0); v > 0 {
		query["page"] = fmt.Sprintf("%d", v)
	}

	data, err := q.Engine.Get(ctx, "/v1/customer/", query)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	pretty, err := json.MarshalIndent(json.RawMessage(data), "", "  ")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(pretty)), nil
}
