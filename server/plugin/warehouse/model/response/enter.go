// Package response defines warehouse API response values.
package response

// BusinessError is the stable business-error payload nested in the framework
// response data. Message is needed for per-row import failures.
type BusinessError struct {
	ErrorCode      string          `json:"errorCode"`
	Message        string          `json:"message"`
	Field          string          `json:"field,omitempty"`
	Sheet          string          `json:"sheet,omitempty"`
	LineNo         int             `json:"lineNo,omitempty"`
	Column         string          `json:"column,omitempty"`
	CurrentVersion uint64          `json:"currentVersion,omitempty"`
	Errors         []BusinessError `json:"errors,omitempty"`
}

// Page is the uniform paginated response envelope.
type Page[T any] struct {
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
	List     []T   `json:"list"`
}
