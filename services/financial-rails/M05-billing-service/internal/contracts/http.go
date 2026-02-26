package contracts

import "time"

type CreateInvoiceRequest struct {
	CustomerID     string        `json:"customer_id"`
	CustomerName   string        `json:"customer_name"`
	CustomerEmail  string        `json:"customer_email"`
	BillingAddress AddressDTO    `json:"billing_address"`
	InvoiceType    string        `json:"invoice_type"`
	LineItems      []LineItemDTO `json:"line_items"`
	DueDate        time.Time     `json:"due_date"`
	Notes          string        `json:"notes"`
}

type AddressDTO struct {
	Line1      string `json:"line1"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type LineItemDTO struct {
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	SourceType  string  `json:"source_type"`
	SourceID    string  `json:"source_id"`
}

type SendInvoiceRequest struct {
	RecipientEmail string `json:"recipient_email"`
}

type VoidInvoiceRequest struct {
	Reason string `json:"reason"`
}

type RefundRequest struct {
	InvoiceID  string  `json:"invoice_id"`
	LineItemID string  `json:"line_item_id"`
	Amount     float64 `json:"amount"`
	Reason     string  `json:"reason"`
}

type DeleteRequest struct {
	Reason string `json:"reason"`
}

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Status string       `json:"status"`
	Error  ErrorPayload `json:"error"`
}

type ErrorPayload struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	RequestID string      `json:"request_id,omitempty"`
	Details   interface{} `json:"details,omitempty"`
}
