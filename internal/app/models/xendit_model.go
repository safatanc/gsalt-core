package models

import (
	"time"
)

// Supported payment methods
type PaymentMethod string

const (
	PaymentMethodQRIS           PaymentMethod = "QRIS"
	PaymentMethodVirtualAccount PaymentMethod = "VIRTUAL_ACCOUNT"
	PaymentMethodEWallet        PaymentMethod = "EWALLET"
	PaymentMethodRetailOutlet   PaymentMethod = "RETAIL_OUTLET"
	PaymentMethodCardless       PaymentMethod = "CARDLESS_CREDIT"
	PaymentMethodCard           PaymentMethod = "CARD"
)

// E-wallet providers
type EWalletProvider string

const (
	EWalletOVO       EWalletProvider = "OVO"
	EWalletDANA      EWalletProvider = "DANA"
	EWalletGOPAY     EWalletProvider = "GOPAY"
	EWalletLINKAJA   EWalletProvider = "LINKAJA"
	EWalletSHOPEEPAY EWalletProvider = "SHOPEEPAY"
)

// Virtual Account banks
type VirtualAccountBank string

const (
	VABankBCA     VirtualAccountBank = "BCA"
	VABankBNI     VirtualAccountBank = "BNI"
	VABankBRI     VirtualAccountBank = "BRI"
	VABankMANDIRI VirtualAccountBank = "MANDIRI"
	VABankBSI     VirtualAccountBank = "BSI"
	VABankPERMATA VirtualAccountBank = "PERMATA"
)

// Payment Request structures for Xendit API
type CreatePaymentRequestBody struct {
	ReferenceID   string               `json:"reference_id"`
	Amount        float64              `json:"amount"`
	Currency      string               `json:"currency"`
	Country       string               `json:"country"`
	PaymentMethod PaymentMethodRequest `json:"payment_method"`
	Description   string               `json:"description"`
	Customer      *CustomerRequest     `json:"customer,omitempty"`
	CallbackURLs  *CallbackURLs        `json:"callback_urls,omitempty"`
	ExpiresAt     *time.Time           `json:"expires_at,omitempty"`
}

type PaymentMethodRequest struct {
	Type           string                 `json:"type"`
	Reusability    string                 `json:"reusability"`
	QRCode         *QRCodeRequest         `json:"qr_code,omitempty"`
	VirtualAccount *VirtualAccountRequest `json:"virtual_account,omitempty"`
	EWallet        *EWalletRequest        `json:"ewallet,omitempty"`
}

type QRCodeRequest struct {
	ChannelCode string `json:"channel_code"`
}

type VirtualAccountRequest struct {
	ChannelCode       string                 `json:"channel_code"`
	ChannelProperties map[string]interface{} `json:"channel_properties,omitempty"`
}

type EWalletRequest struct {
	ChannelCode       string                 `json:"channel_code"`
	ChannelProperties map[string]interface{} `json:"channel_properties,omitempty"`
}

type CustomerRequest struct {
	ReferenceID  string `json:"reference_id,omitempty"`
	GivenNames   string `json:"given_names,omitempty"`
	Email        string `json:"email,omitempty"`
	MobileNumber string `json:"mobile_number,omitempty"`
}

type CallbackURLs struct {
	Success string `json:"success,omitempty"`
	Failure string `json:"failure,omitempty"`
}

// Payment Response structures
type PaymentRequestResponse struct {
	ID            string                `json:"id"`
	ReferenceID   string                `json:"reference_id"`
	BusinessID    string                `json:"business_id"`
	CustomerID    string                `json:"customer_id"`
	Amount        float64               `json:"amount"`
	Currency      string                `json:"currency"`
	Country       string                `json:"country"`
	Status        string                `json:"status"`
	PaymentMethod PaymentMethodResponse `json:"payment_method"`
	Description   string                `json:"description"`
	CallbackURLs  *CallbackURLs         `json:"callback_urls,omitempty"`
	Actions       []ActionResponse      `json:"actions,omitempty"`
	Created       time.Time             `json:"created"`
	Updated       time.Time             `json:"updated"`
	ExpiresAt     *time.Time            `json:"expires_at,omitempty"`
}

type PaymentMethodResponse struct {
	ID             string                  `json:"id"`
	Type           string                  `json:"type"`
	Status         string                  `json:"status"`
	QRCode         *QRCodeResponse         `json:"qr_code,omitempty"`
	VirtualAccount *VirtualAccountResponse `json:"virtual_account,omitempty"`
	EWallet        *EWalletResponse        `json:"ewallet,omitempty"`
	Created        time.Time               `json:"created"`
	Updated        time.Time               `json:"updated"`
}

type QRCodeResponse struct {
	ChannelCode string `json:"channel_code"`
	QRString    string `json:"qr_string"`
}

type VirtualAccountResponse struct {
	ChannelCode       string                 `json:"channel_code"`
	ChannelProperties map[string]interface{} `json:"channel_properties"`
}

type EWalletResponse struct {
	ChannelCode       string                 `json:"channel_code"`
	ChannelProperties map[string]interface{} `json:"channel_properties"`
}

type ActionResponse struct {
	Action string `json:"action"`
	URL    string `json:"url"`
	Method string `json:"method"`
}

// Webhook structures
type WebhookEventType string

const (
	WebhookPaymentCompleted WebhookEventType = "payment.completed"
	WebhookPaymentFailed    WebhookEventType = "payment.failed"
	WebhookPaymentPending   WebhookEventType = "payment.pending"
	WebhookPaymentExpired   WebhookEventType = "payment.expired"
)

type WebhookPayload struct {
	ID               string           `json:"id"`
	Event            WebhookEventType `json:"event"`
	PaymentRequestID string           `json:"payment_request_id"`
	TransactionID    string           `json:"transaction_id"`
	Status           string           `json:"status"`
	Amount           float64          `json:"amount"`
	Currency         string           `json:"currency"`
	PaymentMethod    string           `json:"payment_method"`
	PaidAt           *time.Time       `json:"paid_at,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
}

// Our simplified response format
type PaymentResponse struct {
	PaymentRequestID string                 `json:"payment_request_id"`
	Status           string                 `json:"status"`
	Amount           int64                  `json:"amount"`
	Currency         string                 `json:"currency"`
	PaymentMethod    string                 `json:"payment_method"`
	CheckoutURL      *string                `json:"checkout_url,omitempty"`
	QRCode           *string                `json:"qr_code,omitempty"`
	VirtualAccount   *VirtualAccountDetails `json:"virtual_account,omitempty"`
	EWallet          *EWalletDetails        `json:"ewallet,omitempty"`
	Actions          []PaymentAction        `json:"actions,omitempty"`
	ExpiryTime       *time.Time             `json:"expiry_time,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
}

type VirtualAccountDetails struct {
	BankCode      string `json:"bank_code"`
	AccountNumber string `json:"account_number"`
}

type EWalletDetails struct {
	Provider    string  `json:"provider"`
	CheckoutURL *string `json:"checkout_url,omitempty"`
	QRCode      *string `json:"qr_code,omitempty"`
}

type PaymentAction struct {
	Action string `json:"action"`
	URL    string `json:"url"`
	Method string `json:"method"`
}

type PaymentRequestData struct {
	TransactionID   string              `json:"transaction_id"`
	Amount          int64               `json:"amount"`
	Currency        string              `json:"currency"`
	PaymentMethod   PaymentMethod       `json:"payment_method"`
	Description     string              `json:"description"`
	CustomerName    *string             `json:"customer_name,omitempty"`
	CustomerEmail   *string             `json:"customer_email,omitempty"`
	CustomerPhone   *string             `json:"customer_phone,omitempty"`
	EWalletProvider *EWalletProvider    `json:"ewallet_provider,omitempty"`
	VABank          *VirtualAccountBank `json:"va_bank,omitempty"`
	ExpiryTime      *time.Time          `json:"expiry_time,omitempty"`
}

// XenditWebhookPayload represents the webhook payload from Xendit for handlers
type XenditWebhookPayload struct {
	ID               string  `json:"id"`
	Event            string  `json:"event"`
	PaymentRequestID string  `json:"payment_request_id"`
	TransactionID    string  `json:"transaction_id"`
	Status           string  `json:"status"`
	Amount           float64 `json:"amount"`
	Currency         string  `json:"currency"`
	PaymentMethod    string  `json:"payment_method"`
	PaidAt           *string `json:"paid_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
}
