package models

import (
	"time"
)

// Flip Payment Methods
type FlipPaymentMethod string

const (
	FlipPaymentMethodUnknown        FlipPaymentMethod = "UNKNOWN"
	FlipPaymentMethodVirtualAccount FlipPaymentMethod = "VA"
	FlipPaymentMethodQRIS           FlipPaymentMethod = "QRIS"
	FlipPaymentMethodEWallet        FlipPaymentMethod = "EWALLET"
	FlipPaymentMethodRetailOutlet   FlipPaymentMethod = "RETAIL"
	FlipPaymentMethodCreditCard     FlipPaymentMethod = "CREDIT_CARD"
	FlipPaymentMethodDebitCard      FlipPaymentMethod = "DEBIT_CARD"
	FlipPaymentMethodDirectDebit    FlipPaymentMethod = "DIRECT_DEBIT"
	FlipPaymentMethodBankTransfer   FlipPaymentMethod = "BANK_TRANSFER"
)

// Sender Bank Types for Flip V3 API
type FlipSenderBankType string

const (
	FlipSenderBankTypeVirtualAccount     FlipSenderBankType = "virtual_account"
	FlipSenderBankTypeWalletAccount      FlipSenderBankType = "wallet_account"
	FlipSenderBankTypeCreditCardAccount  FlipSenderBankType = "credit_card_account"
	FlipSenderBankTypeDebitCardAccount   FlipSenderBankType = "debit_card_account"
	FlipSenderBankTypeOnlineToOffline    FlipSenderBankType = "online_to_offline_account"
	FlipSenderBankTypeDirectDebitAccount FlipSenderBankType = "direct_debit_account"
	FlipSenderBankTypeQRIS               FlipSenderBankType = "qris"
)

// Virtual Account Banks for Flip
type FlipVABank string

const (
	FlipVABankBCA     FlipVABank = "bca"
	FlipVABankBNI     FlipVABank = "bni"
	FlipVABankBRI     FlipVABank = "bri"
	FlipVABankMANDIRI FlipVABank = "mandiri"
	FlipVABankCIMB    FlipVABank = "cimb"
	FlipVABankPERMATA FlipVABank = "permata"
	FlipVABankBSI     FlipVABank = "bsi"
	FlipVABankDANAMON FlipVABank = "danamon"
	FlipVABankMAYBANK FlipVABank = "maybank"
)

// E-wallet providers for Flip
type FlipEWalletProvider string

const (
	FlipEWalletOVO       FlipEWalletProvider = "ovo"
	FlipEWalletDANA      FlipEWalletProvider = "dana"
	FlipEWalletGOPAY     FlipEWalletProvider = "gopay"
	FlipEWalletLINKAJA   FlipEWalletProvider = "linkaja"
	FlipEWalletSHOPEEPAY FlipEWalletProvider = "shopeepay"
)

// Credit Card providers for Flip
type FlipCreditCardProvider string

const (
	FlipCreditCardVISA       FlipCreditCardProvider = "visa"
	FlipCreditCardMASTERCARD FlipCreditCardProvider = "mastercard"
	FlipCreditCardJCB        FlipCreditCardProvider = "jcb"
	FlipCreditCardAMEX       FlipCreditCardProvider = "amex"
)

// Retail Outlet providers for Flip
type FlipRetailProvider string

const (
	FlipRetailALFAMART  FlipRetailProvider = "alfamart"
	FlipRetailINDOMART  FlipRetailProvider = "indomaret"
	FlipRetailCIRCLEK   FlipRetailProvider = "circlek"
	FlipRetailLAWSON    FlipRetailProvider = "lawson"
	FlipRetailDANDANPAY FlipRetailProvider = "dandanpay"
)

// Direct Debit Bank providers
type FlipDirectDebitBank string

const (
	FlipDirectDebitBCA     FlipDirectDebitBank = "bca"
	FlipDirectDebitBNI     FlipDirectDebitBank = "bni"
	FlipDirectDebitBRI     FlipDirectDebitBank = "bri"
	FlipDirectDebitMANDIRI FlipDirectDebitBank = "mandiri"
)

// CreateBillRequest represents the V3 API request structure for creating bills
type CreateBillRequest struct {
	Title                 string       `json:"title" validate:"required,max=255"`
	Type                  string       `json:"type" validate:"required,oneof=single multiple"`                       // "single" or "multiple"
	Step                  string       `json:"step" validate:"required,oneof=checkout checkout_seamless direct_api"` // "checkout", "checkout_seamless", "direct_api"
	Amount                int64        `json:"amount" validate:"required,min=100"`
	ExpiredDate           string       `json:"expired_date,omitempty"` // ISO 8601 format or relative time
	RedirectURL           string       `json:"redirect_url,omitempty" validate:"omitempty,url"`
	IsAddressRequired     bool         `json:"is_address_required,omitempty"`
	SenderAddress         string       `json:"sender_address,omitempty" validate:"omitempty,max=500"`
	IsPhoneNumberRequired bool         `json:"is_phone_number_required,omitempty"`
	SenderPhoneNumber     string       `json:"sender_phone_number,omitempty" validate:"omitempty,max=20"`
	SenderName            string       `json:"sender_name,omitempty" validate:"omitempty,max=255"`
	SenderEmail           string       `json:"sender_email,omitempty" validate:"omitempty,email"`
	SenderBank            string       `json:"sender_bank,omitempty" validate:"omitempty,max=20"`
	SenderBankType        string       `json:"sender_bank_type,omitempty" validate:"omitempty,max=50"`
	ReferenceID           string       `json:"reference_id,omitempty" validate:"omitempty,max=255"`
	ChargeFee             bool         `json:"charge_fee,omitempty"`
	ItemDetails           []ItemDetail `json:"item_details,omitempty"`
}

// ItemDetail represents product/service details in the bill
type ItemDetail struct {
	ID       string `json:"id,omitempty" validate:"omitempty,max=100"`
	Name     string `json:"name" validate:"required,max=255"`
	Price    int64  `json:"price" validate:"required,min=0"`
	Quantity int    `json:"quantity" validate:"required,min=1"`
	Desc     string `json:"desc,omitempty" validate:"omitempty,max=500"`
	ImageURL string `json:"image_url,omitempty" validate:"omitempty,url"`
}

// Bill Response
type BillResponse struct {
	LinkID            int       `json:"link_id"`
	LinkURL           string    `json:"link_url"`
	Title             string    `json:"title"`
	Type              string    `json:"type"`
	Amount            int64     `json:"amount"`
	ExpiredDate       string    `json:"expired_date"`
	RedirectURL       string    `json:"redirect_url"`
	IsAddressRequired int       `json:"is_address_required"`
	IsPhoneRequired   int       `json:"is_phone_number_required"`
	Step              int       `json:"step"`
	CreatedFrom       string    `json:"created_from"`
	Status            string    `json:"status"`
	Created           time.Time `json:"created"`
}

// PaymentGatewayResponse is a comprehensive response struct for Flip's Create Bill API
type PaymentGatewayResponse struct {
	LinkID                int              `json:"link_id"`
	LinkURL               string           `json:"link_url"`
	Title                 string           `json:"title"`
	Type                  string           `json:"type"`
	Amount                int64            `json:"amount"`
	RedirectURL           string           `json:"redirect_url"`
	ExpiredDate           *string          `json:"expired_date"` // Use pointer for nullable fields
	CreatedFrom           string           `json:"created_from,omitempty"`
	Status                string           `json:"status"`
	Step                  int              `json:"step"`
	Customer              *CustomerDetails `json:"customer,omitempty"`
	BillPayment           *BillPayment     `json:"bill_payment,omitempty"` // Pointer, as it's not always present
	PaymentURL            string           `json:"payment_url,omitempty"`
	Instructions          string           `json:"instructions,omitempty"`
	InternalReference     string           `json:"internal_reference,omitempty"`
	PaymentMethod         string           `json:"payment_method,omitempty"`
	SenderBank            string           `json:"sender_bank,omitempty"`
	SenderBankType        string           `json:"sender_bank_type,omitempty"`
	IsAddressRequired     int              `json:"is_address_required"`
	IsPhoneNumberRequired int              `json:"is_phone_number_required"`
	CreatedAt             int64            `json:"created_at,omitempty"`
}

type CustomerDetails struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
}

type BillPayment struct {
	ID                  string               `json:"id"`
	Amount              int64                `json:"amount"`
	UniqueCode          int                  `json:"unique_code"`
	Status              string               `json:"status"`
	SenderBank          string               `json:"sender_bank"`
	SenderBankType      string               `json:"sender_bank_type"`
	ReceiverBankAccount *ReceiverBankAccount `json:"receiver_bank_account,omitempty"`
	UserAddress         string               `json:"user_address,omitempty"`
	UserPhone           string               `json:"user_phone,omitempty"`
	CreatedAt           int64                `json:"created_at"` // Assuming Unix timestamp
}

type ReceiverBankAccount struct {
	AccountNumber string `json:"account_number"`
	AccountType   string `json:"account_type"`
	BankCode      string `json:"bank_code"`
	AccountHolder string `json:"account_holder"`
}

// Payment Status
type FlipPaymentStatus string

const (
	FlipStatusPending    FlipPaymentStatus = "PENDING"
	FlipStatusSuccessful FlipPaymentStatus = "SUCCESSFUL"
	FlipStatusFailed     FlipPaymentStatus = "FAILED"
	FlipStatusCancelled  FlipPaymentStatus = "CANCELLED"
	FlipStatusExpired    FlipPaymentStatus = "EXPIRED"
)

// Bill Payment Details
type BillPaymentResponse struct {
	ID                int               `json:"id"`
	BillLinkID        int               `json:"bill_link_id"`
	BillTitle         string            `json:"bill_title"`
	SenderName        string            `json:"sender_name"`
	SenderEmail       string            `json:"sender_email"`
	SenderPhoneNumber string            `json:"sender_phone_number"`
	SenderAddress     string            `json:"sender_address"`
	Amount            int64             `json:"amount"`
	Status            FlipPaymentStatus `json:"status"`
	Settlement        string            `json:"settlement"`
	Created           time.Time         `json:"created"`
	Updated           time.Time         `json:"updated"`
}

// Get Bill Response
type GetBillResponse struct {
	LinkID            int                   `json:"link_id"`
	LinkURL           string                `json:"link_url"`
	Title             string                `json:"title"`
	Type              string                `json:"type"`
	Amount            int64                 `json:"amount"`
	ExpiredDate       string                `json:"expired_date"`
	RedirectURL       string                `json:"redirect_url"`
	IsAddressRequired int                   `json:"is_address_required"`
	IsPhoneRequired   int                   `json:"is_phone_number_required"`
	Step              int                   `json:"step"`
	CreatedFrom       string                `json:"created_from"`
	Status            string                `json:"status"`
	Created           time.Time             `json:"created"`
	Updated           time.Time             `json:"updated"`
	BillPayments      []BillPaymentResponse `json:"bill_payments"`
}

// Webhook payload structure
type FlipWebhookPayload struct {
	ID                int               `json:"id"`
	BillLinkID        int               `json:"bill_link_id"`
	BillTitle         string            `json:"bill_title"`
	SenderName        string            `json:"sender_name"`
	SenderEmail       string            `json:"sender_email"`
	SenderPhoneNumber string            `json:"sender_phone_number"`
	SenderAddress     string            `json:"sender_address"`
	Amount            int64             `json:"amount"`
	Status            FlipPaymentStatus `json:"status"`
	Settlement        string            `json:"settlement"`
	Created           string            `json:"created"`
	Updated           string            `json:"updated"`
}

// API Error Response
type FlipErrorResponse struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors,omitempty"`
}

// Our simplified payment response format for consistency
type FlipPaymentResponse struct {
	BillID        int               `json:"bill_id"`
	LinkID        int               `json:"link_id"`
	LinkURL       string            `json:"link_url"`
	Title         string            `json:"title"`
	Amount        int64             `json:"amount"`
	Status        FlipPaymentStatus `json:"status"`
	PaymentMethod string            `json:"payment_method"`
	ExpiryTime    *time.Time        `json:"expiry_time,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	RedirectURL   string            `json:"redirect_url,omitempty"`
}

// FlipPaymentRequestData represents the data needed to create a payment
type FlipPaymentRequestData struct {
	TransactionID         string                  `json:"transaction_id" validate:"required,max=255"`
	Title                 string                  `json:"title" validate:"required,max=255"`
	Amount                int64                   `json:"amount" validate:"required,min=100"`
	ExpiryHours           int                     `json:"expiry_hours" validate:"required,min=1,max=168"` // Hours from now, max 7 days
	PaymentMethod         FlipPaymentMethod       `json:"payment_method" validate:"required"`
	CustomerName          *string                 `json:"customer_name,omitempty" validate:"omitempty,max=255"`
	CustomerEmail         *string                 `json:"customer_email,omitempty" validate:"omitempty,email"`
	CustomerPhone         *string                 `json:"customer_phone,omitempty" validate:"omitempty,max=20"`
	CustomerAddress       *string                 `json:"customer_address,omitempty" validate:"omitempty,max=500"`
	RedirectURL           *string                 `json:"redirect_url,omitempty" validate:"omitempty,url"`
	VABank                *FlipVABank             `json:"va_bank,omitempty"`
	EWalletProvider       *FlipEWalletProvider    `json:"ewallet_provider,omitempty"`
	CreditCardProvider    *FlipCreditCardProvider `json:"credit_card_provider,omitempty"`
	RetailProvider        *FlipRetailProvider     `json:"retail_provider,omitempty"`
	DirectDebitBank       *FlipDirectDebitBank    `json:"direct_debit_bank,omitempty"`
	ReferenceID           *string                 `json:"reference_id,omitempty" validate:"omitempty,max=255"`
	ChargeFee             bool                    `json:"charge_fee,omitempty"`
	ItemDetails           []ItemDetail            `json:"item_details,omitempty"`
	IsAddressRequired     bool                    `json:"is_address_required,omitempty"`
	IsPhoneNumberRequired bool                    `json:"is_phone_number_required,omitempty"`
}

// ========== GENERAL API MODELS ==========

// FlipBalance represents the account balance response
type FlipBalance struct {
	Balance int64 `json:"balance"`
}

// FlipBank represents bank information
type FlipBank struct {
	BankCode        string `json:"bank_code"`
	Name            string `json:"name"`
	Fee             int64  `json:"fee"`
	Queue           int    `json:"queue"`
	Status          string `json:"status"`
	MinAmount       int64  `json:"min_amount"`
	MaxAmount       int64  `json:"max_amount"`
	MaintenanceFrom string `json:"maintenance_from,omitempty"`
	MaintenanceTo   string `json:"maintenance_to,omitempty"`
}

// FlipMaintenanceInfo represents maintenance information
type FlipMaintenanceInfo struct {
	Maintenance bool                      `json:"maintenance"`
	Banks       []FlipMaintenanceBankInfo `json:"banks"`
}

// FlipMaintenanceBankInfo represents bank maintenance info
type FlipMaintenanceBankInfo struct {
	BankCode string `json:"bank_code"`
	Name     string `json:"name"`
	Status   string `json:"status"`
}

// ========== DISBURSEMENT/MONEY TRANSFER MODELS ==========

// Bank Account Inquiry Request
type BankAccountInquiryRequest struct {
	AccountNumber string `json:"account_number" form:"account_number" validate:"required,max=50"`
	BankCode      string `json:"bank_code" form:"bank_code" validate:"required,max=10"`
	InquiryKey    string `json:"inquiry_key,omitempty" form:"inquiry_key" validate:"omitempty,max=100"`
}

// Bank Account Inquiry Response
type BankAccountInquiryResponse struct {
	BankCode      string `json:"bank_code"`
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
	Status        string `json:"status"`
	InquiryKey    string `json:"inquiry_key,omitempty"`
}

// Disbursement Request
type DisbursementRequest struct {
	AccountNumber    string `json:"account_number" form:"account_number" validate:"required,max=50"`
	BankCode         string `json:"bank_code" form:"bank_code" validate:"required,max=10"`
	Amount           int64  `json:"amount" form:"amount" validate:"required,min=10000"`
	Remark           string `json:"remark,omitempty" form:"remark" validate:"omitempty,max=255"`
	RecipientCity    string `json:"recipient_city,omitempty" form:"recipient_city" validate:"omitempty,max=100"`
	BeneficiaryEmail string `json:"beneficiary_email,omitempty" form:"beneficiary_email" validate:"omitempty,email"`
	IdempotencyKey   string `json:"-"` // Sent as header
	Timestamp        string `json:"-"` // Sent as header
}

// Special Disbursement Request
type SpecialDisbursementRequest struct {
	DisbursementRequest
	SenderCountry        string `json:"sender_country" form:"sender_country" validate:"required,max=50"`
	SenderPlaceOfBirth   string `json:"sender_place_of_birth,omitempty" form:"sender_place_of_birth" validate:"omitempty,max=100"`
	SenderDateOfBirth    string `json:"sender_date_of_birth,omitempty" form:"sender_date_of_birth" validate:"omitempty,datetime=2006-01-02"`
	SenderIdentityType   string `json:"sender_identity_type,omitempty" form:"sender_identity_type" validate:"omitempty,max=20"`
	SenderName           string `json:"sender_name" form:"sender_name" validate:"required,max=255"`
	SenderAddress        string `json:"sender_address" form:"sender_address" validate:"required,max=500"`
	SenderIdentityNumber string `json:"sender_identity_number,omitempty" form:"sender_identity_number" validate:"omitempty,max=50"`
	SenderJob            string `json:"sender_job,omitempty" form:"sender_job" validate:"omitempty,max=100"`
	Direction            string `json:"direction" form:"direction" validate:"required,oneof=DOMESTIC INTERNATIONAL"`
}

// Disbursement Response
type DisbursementResponse struct {
	ID             int                 `json:"id"`
	UserID         int                 `json:"user_id"`
	Amount         int64               `json:"amount"`
	Status         string              `json:"status"`
	Reason         string              `json:"reason"`
	Timestamp      string              `json:"timestamp"`
	BankCode       string              `json:"bank_code"`
	AccountNumber  string              `json:"account_number"`
	RecipientName  string              `json:"recipient_name"`
	SenderBank     string              `json:"sender_bank"`
	Remark         string              `json:"remark"`
	Receipt        string              `json:"receipt"`
	TimeServed     string              `json:"time_served"`
	BundleID       int                 `json:"bundle_id"`
	CompanyID      int                 `json:"company_id"`
	RecipientCity  int                 `json:"recipient_city"`
	CreatedFrom    string              `json:"created_from"`
	Direction      string              `json:"direction"`
	Sender         *DisbursementSender `json:"sender,omitempty"`
	Fee            int64               `json:"fee"`
	IdempotencyKey string              `json:"idempotency_key"`
}

// Disbursement Sender
type DisbursementSender struct {
	SenderName           string `json:"sender_name"`
	SenderAddress        string `json:"sender_address"`
	SenderIdentityNumber string `json:"sender_identity_number"`
	SenderCountry        int    `json:"sender_country"`
	SenderJob            string `json:"sender_job"`
	SenderPlaceOfBirth   int    `json:"sender_place_of_birth"`
	SenderDateOfBirth    string `json:"sender_date_of_birth"`
	SenderIdentityType   string `json:"sender_identity_type"`
}

// Disbursement List Response
type DisbursementListResponse struct {
	Data        []DisbursementResponse `json:"data"`
	TotalData   int                    `json:"total_data"`
	DataPerPage int                    `json:"data_per_page"`
	TotalPage   int                    `json:"total_page"`
	Page        int                    `json:"page"`
}

// Disbursement Status
type DisbursementStatus string

const (
	DisbursementStatusPending   DisbursementStatus = "PENDING"
	DisbursementStatusProcessed DisbursementStatus = "PROCESSED"
	DisbursementStatusDone      DisbursementStatus = "DONE"
	DisbursementStatusCancelled DisbursementStatus = "CANCELLED"
)

// Disbursement Identity Type
type DisbursementIdentityType string

const (
	DisbursementIdentityNationalID DisbursementIdentityType = "nat_id"
	DisbursementIdentityDriverLic  DisbursementIdentityType = "drv_lic"
	DisbursementIdentityPassport   DisbursementIdentityType = "passport"
	DisbursementIdentityBankAcc    DisbursementIdentityType = "bank_acc"
)

// ========== GSALT SPECIFIC MODELS ==========

// GSALT Disbursement Request
type GSALTDisbursementRequest struct {
	TransactionID  string `json:"transaction_id" validate:"required,max=255"`
	RecipientName  string `json:"recipient_name" validate:"required,max=255"`
	AccountNumber  string `json:"account_number" validate:"required,max=50"`
	BankCode       string `json:"bank_code" validate:"required,max=10"`
	AmountGSALT    int64  `json:"amount_gsalt" validate:"required,min=1"`   // Amount in GSALT units
	AmountIDR      int64  `json:"amount_idr" validate:"required,min=10000"` // Amount in IDR
	Description    string `json:"description,omitempty" validate:"omitempty,max=500"`
	RecipientEmail string `json:"recipient_email,omitempty" validate:"omitempty,email"`
	IdempotencyKey string `json:"idempotency_key" validate:"required,max=255"`
}

// GSALT Disbursement Response
type GSALTDisbursementResponse struct {
	TransactionID  string             `json:"transaction_id"`
	DisbursementID int                `json:"disbursement_id"`
	Status         DisbursementStatus `json:"status"`
	AmountGSALT    int64              `json:"amount_gsalt"`
	AmountIDR      int64              `json:"amount_idr"`
	Fee            int64              `json:"fee"`
	RecipientName  string             `json:"recipient_name"`
	AccountNumber  string             `json:"account_number"`
	BankCode       string             `json:"bank_code"`
	Receipt        string             `json:"receipt,omitempty"`
	ProcessedAt    *time.Time         `json:"processed_at,omitempty"`
	CompletedAt    *time.Time         `json:"completed_at,omitempty"`
	IdempotencyKey string             `json:"idempotency_key"`
}

// Payment Method Details for specific payment methods like QR codes, VA numbers, etc.
type PaymentMethodDetails struct {
	LinkID            int    `json:"link_id"`
	LinkURL           string `json:"link_url"`
	Status            string `json:"status"`
	Amount            int64  `json:"amount"`
	ExpiredDate       string `json:"expired_date"`
	PaymentURL        string `json:"payment_url"`
	SenderName        string `json:"sender_name,omitempty"`
	SenderEmail       string `json:"sender_email,omitempty"`
	SenderPhoneNumber string `json:"sender_phone_number,omitempty"`

	// Payment method identification
	Method      FlipPaymentMethod `json:"method"`                 // Payment method type
	Provider    string            `json:"provider,omitempty"`     // Provider name (e.g., bank name, ewallet provider)
	AccountInfo string            `json:"account_info,omitempty"` // Account specific info (e.g., VA number, account number)

	// Payment method specific fields
	QRCodeURL    string `json:"qr_code_url,omitempty"`    // For QRIS payments
	QRCodeString string `json:"qr_code_string,omitempty"` // For QRIS payments
	VANumber     string `json:"va_number,omitempty"`      // For Virtual Account payments
	VABank       string `json:"va_bank,omitempty"`        // For Virtual Account payments
	EWalletURL   string `json:"ewallet_url,omitempty"`    // For E-wallet payments
	RetailCode   string `json:"retail_code,omitempty"`    // For retail outlet payments
}

// FlipPaymentResponseWithDetails combines basic payment response with detailed payment method information
type FlipPaymentResponseWithDetails struct {
	FlipPaymentResponse  FlipPaymentResponse   `json:"payment_response"`
	PaymentMethodDetails *PaymentMethodDetails `json:"payment_method_details"`
}
