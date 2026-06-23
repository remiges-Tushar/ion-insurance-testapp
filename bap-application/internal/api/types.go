package api

// SearchRequest is sent by the frontend to trigger a Beckn search.
type SearchRequest struct {
	Query string `json:"query"`
}

// SelectRequest wraps the message body for a Beckn select call.
type SelectRequest struct {
	Message map[string]any `json:"message"`
}

// InitRequest wraps the message body for a Beckn init call.
type InitRequest struct {
	Message       map[string]any `json:"message"`
	TransactionID string         `json:"transaction_id,omitempty"`
}

// ConfirmRequest wraps the message body for a Beckn confirm call.
type ConfirmRequest struct {
	Message       map[string]any `json:"message"`
	TransactionID string         `json:"transaction_id,omitempty"`
}

// StatusRequest carries a transaction_id for a Beckn status enquiry.
type StatusRequest struct {
	TransactionID string `json:"transaction_id" binding:"required"`
}

// CancelRequest carries a transaction_id for a Beckn cancel call.
type CancelRequest struct {
	TransactionID string `json:"transaction_id" binding:"required"`
}

// RateRequest wraps a rating message.
type RateRequest struct {
	Message       map[string]any `json:"message"`
	TransactionID string         `json:"transaction_id,omitempty"`
}

// SupportRequest wraps a support message.
type SupportRequest struct {
	Message       map[string]any `json:"message"`
	TransactionID string         `json:"transaction_id,omitempty"`
}

// Problem is an RFC 7807-style error response.
type Problem struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Status int    `json:"status"`
}
