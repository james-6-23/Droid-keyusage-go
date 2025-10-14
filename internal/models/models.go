package models

import "time"

// APIKey represents a stored API key
type APIKey struct {
	ID        string    `json:"id"`
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// APIKeyMasked represents an API key with masked value for display
type APIKeyMasked struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Masked    string    `json:"masked"`
	CreatedAt time.Time `json:"created_at"`
}

// Usage represents API key usage information
type Usage struct {
	ID               string    `json:"id"`
	Key              string    `json:"key,omitempty"`
	StartDate        string    `json:"start_date"`
	EndDate          string    `json:"end_date"`
	TotalAllowance   float64   `json:"total_allowance"`
	OrgTotalUsed     float64   `json:"org_total_tokens_used"`
	Remaining        float64   `json:"remaining"`
	UsedRatio        float64   `json:"used_ratio"`
	LastUpdated      time.Time `json:"last_updated"`
	Error            string    `json:"error,omitempty"`
}

// FactoryAPIResponse represents the response from Factory.ai API
type FactoryAPIResponse struct {
	Usage struct {
		StartDate int64 `json:"startDate"`
		EndDate   int64 `json:"endDate"`
		Standard  struct {
			OrgTotalTokensUsed float64 `json:"orgTotalTokensUsed"`
			TotalAllowance     float64 `json:"totalAllowance"`
			UsedRatio          float64 `json:"usedRatio"`
		} `json:"standard"`
	} `json:"usage"`
}

// AggregatedData represents the aggregated usage data
type AggregatedData struct {
	UpdateTime  string   `json:"update_time"`
	TotalCount  int      `json:"total_count"`
	Totals      Totals   `json:"totals"`
	Data        []*Usage `json:"data"`
}

// Totals represents the total usage statistics
type Totals struct {
	TotalOrgTotalTokensUsed float64 `json:"total_orgTotalTokensUsed"`
	TotalAllowance          float64 `json:"total_totalAllowance"`
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Password string `json:"password"`
}

// ImportRequest represents batch import request
type ImportRequest struct {
	Keys []string `json:"keys"`
}

// ImportResult represents batch import result
type ImportResult struct {
	Success    int `json:"success"`
	Failed     int `json:"failed"`
	Duplicates int `json:"duplicates"`
}

// BatchDeleteRequest represents batch delete request
type BatchDeleteRequest struct {
	IDs []string `json:"ids"`
}

// BatchDeleteResult represents batch delete result
type BatchDeleteResult struct {
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}
