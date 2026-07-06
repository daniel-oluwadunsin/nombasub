package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

var dateLayout = "2006-01-02"

func validateDate(field, value string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse(dateLayout, value); err != nil {
		return fmt.Errorf("%s must be a date in YYYY-MM-DD format", field)
	}
	return nil
}

func validateDates(fields map[string]string) error {
	for name, val := range fields {
		if err := validateDate(name, val); err != nil {
			return err
		}
	}
	return nil
}

func validateEnum(field, value string, allowed ...string) error {
	if value == "" {
		return nil
	}
	v := strings.ToLower(strings.TrimSpace(value))
	for _, opt := range allowed {
		if strings.EqualFold(v, opt) {
			return nil
		}
	}
	return fmt.Errorf("%s must be one of: %s", field, strings.Join(allowed, ", "))
}

func validateIdentifier(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s cannot be empty", field)
	}
	if len(value) > 200 {
		return fmt.Errorf("%s exceeds maximum length of 200 characters", field)
	}
	return nil
}

func toolError(code, message string, retryable bool) (*mcp.CallToolResult, error) {
	payload := map[string]any{
		"error": map[string]any{
			"code":      code,
			"message":   message,
			"retryable": retryable,
		},
	}
	body, _ := json.Marshal(payload)
	return mcp.NewToolResultError(string(body)), nil
}

func validationError(err error) (*mcp.CallToolResult, error) {
	return toolError("invalid_input", err.Error(), false)
}

func upstreamError(err error) (*mcp.CallToolResult, error) {
	return toolError("engine_error", err.Error(), true)
}
