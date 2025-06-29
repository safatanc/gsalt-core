package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/infrastructures"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type FlipService struct {
	client *infrastructures.FlipClient
	db     *gorm.DB
}

func NewFlipService(flipClient *infrastructures.FlipClient, db *gorm.DB) *FlipService {
	return &FlipService{
		client: flipClient,
		db:     db,
	}
}

// CreateBill creates a new payment bill using Flip V3 API
func (s *FlipService) CreateBill(ctx context.Context, data models.FlipPaymentRequestData) (*models.FlipPaymentResponse, error) {
	// Calculate expiry date
	expiryTime := time.Now().Add(time.Duration(data.ExpiryHours) * time.Hour)

	// Create item details for the payment
	itemDetails := data.ItemDetails
	if len(itemDetails) == 0 {
		// Default item if none provided
		itemDetails = []models.ItemDetail{
			{
				Name:     data.Title,
				Price:    data.Amount,
				Quantity: 1,
				Desc:     "GSALT Payment",
			},
		}
	}

	// Create V3 API request
	billRequest := models.CreateBillRequest{
		Title:                 data.Title,
		Type:                  "single",   // Always single use for our payments
		Step:                  "checkout", // Use checkout step for payment selection
		Amount:                data.Amount,
		RedirectURL:           *data.RedirectURL,
		IsAddressRequired:     data.IsAddressRequired,
		IsPhoneNumberRequired: data.IsPhoneNumberRequired,
		ReferenceID:           data.TransactionID, // Use transaction ID as reference
		ChargeFee:             data.ChargeFee,
		ItemDetails:           itemDetails,
	}

	// Set customer information if provided
	if data.CustomerName != nil {
		billRequest.SenderName = *data.CustomerName
	}
	if data.CustomerEmail != nil {
		billRequest.SenderEmail = *data.CustomerEmail
	}
	if data.CustomerPhone != nil {
		billRequest.SenderPhoneNumber = *data.CustomerPhone
	}
	if data.CustomerAddress != nil {
		billRequest.SenderAddress = *data.CustomerAddress
	}

	// Set payment method specific configurations
	s.setPaymentMethodConfigV3(&billRequest, data)

	// Convert to JSON
	jsonData, err := json.Marshal(billRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := s.client.GetFullURL("/pwf/bill")
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers for V3 API
	req.Header.Set("Authorization", s.client.GetAuthHeader())
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errorResp models.FlipErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("flip API error: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("flip API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse successful response
	var billResp models.BillResponse
	if err := json.Unmarshal(body, &billResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to our response format
	paymentResp := s.convertToBillPaymentResponse(billResp, data.PaymentMethod)
	paymentResp.ExpiryTime = &expiryTime

	return paymentResp, nil
}

// GetBillStatus retrieves the status of a bill and its payments using V3 API
func (s *FlipService) GetBillStatus(ctx context.Context, linkID int) (*models.GetBillResponse, error) {
	// Create HTTP request using V3 endpoint
	url := s.client.GetFullURL(fmt.Sprintf("/pwf/%d/bill", linkID))
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", s.client.GetAuthHeader())
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp models.FlipErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("flip API error: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("flip API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var billStatus models.GetBillResponse
	if err := json.Unmarshal(body, &billStatus); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &billStatus, nil
}

// CreateTopupPayment creates a bill for GSALT topup with V3 features
func (s *FlipService) CreateTopupPayment(ctx context.Context, transactionID string, amountIDR int64, paymentMethod models.FlipPaymentMethod, customerInfo *models.Account) (*models.FlipPaymentResponse, error) {
	data := models.FlipPaymentRequestData{
		TransactionID:         transactionID,
		Title:                 fmt.Sprintf("GSALT Topup - %s", transactionID),
		Amount:                amountIDR,
		ExpiryHours:           24, // 24 hours expiry
		PaymentMethod:         paymentMethod,
		ReferenceID:           &transactionID, // Use transaction ID as reference
		ChargeFee:             false,          // Don't charge additional fee to customer
		IsAddressRequired:     false,          // Address not required for topup
		IsPhoneNumberRequired: true,           // Phone number required for verification
	}

	// Set customer information from account if available
	if customerInfo != nil {
		// Use ConnectID as customer reference since Account model doesn't have Username
		customerID := customerInfo.ConnectID.String()
		data.CustomerName = &customerID
		// Note: You might want to add Email and Phone fields to Account model
		// or create a separate Customer model for complete information
	}

	// Calculate GSALT amount for display
	gsaltAmount := decimal.NewFromInt(amountIDR).Div(decimal.NewFromInt(1000))

	// Add item details for better payment description
	data.ItemDetails = []models.ItemDetail{
		{
			Name:     fmt.Sprintf("GSALT Topup - %s GSALT", gsaltAmount.String()),
			Price:    amountIDR,
			Quantity: 1,
			Desc:     fmt.Sprintf("Topup %s GSALT to account %s", gsaltAmount.String(), customerInfo.ConnectID.String()),
		},
	}

	return s.CreateBill(ctx, data)
}

// Helper methods

func (s *FlipService) setPaymentMethodConfigV3(billRequest *models.CreateBillRequest, data models.FlipPaymentRequestData) {
	// Set step based on payment method
	switch data.PaymentMethod {
	case models.FlipPaymentMethodVirtualAccount:
		billRequest.Step = "checkout" // VA usually requires checkout step
		billRequest.SenderBankType = string(models.FlipSenderBankTypeVirtualAccount)
		if data.VABank != nil {
			billRequest.SenderBank = string(*data.VABank)
		}
	case models.FlipPaymentMethodQRIS:
		billRequest.Step = "checkout" // QRIS usually requires checkout step
		billRequest.SenderBankType = string(models.FlipSenderBankTypeQRIS)
	case models.FlipPaymentMethodEWallet:
		billRequest.Step = "checkout" // E-wallet usually requires checkout step
		billRequest.SenderBankType = string(models.FlipSenderBankTypeWalletAccount)
		if data.EWalletProvider != nil {
			billRequest.SenderBank = string(*data.EWalletProvider)
		}
	case models.FlipPaymentMethodRetailOutlet:
		billRequest.Step = "checkout" // Retail outlet usually requires checkout step
		billRequest.SenderBankType = string(models.FlipSenderBankTypeOnlineToOffline)
		if data.RetailProvider != nil {
			billRequest.SenderBank = string(*data.RetailProvider)
		}
	case models.FlipPaymentMethodCreditCard:
		billRequest.Step = "checkout" // Credit card requires checkout step
		billRequest.SenderBankType = string(models.FlipSenderBankTypeCreditCardAccount)
		if data.CreditCardProvider != nil {
			billRequest.SenderBank = string(*data.CreditCardProvider)
		} else {
			billRequest.SenderBank = "credit_card" // Default credit card
		}
	case models.FlipPaymentMethodDebitCard:
		billRequest.Step = "checkout" // Debit card requires checkout step
		billRequest.SenderBankType = string(models.FlipSenderBankTypeDebitCardAccount)
		if data.CreditCardProvider != nil {
			billRequest.SenderBank = string(*data.CreditCardProvider)
		} else {
			billRequest.SenderBank = "debit_card" // Default debit card
		}
	case models.FlipPaymentMethodDirectDebit:
		billRequest.Step = "checkout" // Direct debit requires checkout step
		billRequest.SenderBankType = string(models.FlipSenderBankTypeDirectDebitAccount)
		if data.DirectDebitBank != nil {
			billRequest.SenderBank = string(*data.DirectDebitBank)
		}
	case models.FlipPaymentMethodBankTransfer:
		billRequest.Step = "checkout" // Bank transfer requires checkout step
		billRequest.SenderBankType = string(models.FlipSenderBankTypeVirtualAccount)
		if data.VABank != nil {
			billRequest.SenderBank = string(*data.VABank)
		}
	default:
		billRequest.Step = "checkout" // Default step
		billRequest.SenderBankType = string(models.FlipSenderBankTypeQRIS)
	}
}

func (s *FlipService) convertToBillPaymentResponse(billResp models.BillResponse, paymentMethod models.FlipPaymentMethod) *models.FlipPaymentResponse {
	// Parse expiry time
	var expiryTime *time.Time
	if billResp.ExpiredDate != "" {
		if parsedTime, err := time.Parse("2006-01-02 15:04:05", billResp.ExpiredDate); err == nil {
			expiryTime = &parsedTime
		}
	}

	return &models.FlipPaymentResponse{
		LinkID:        billResp.LinkID,
		LinkURL:       billResp.LinkURL,
		Title:         billResp.Title,
		Amount:        billResp.Amount,
		Status:        models.FlipPaymentStatus(billResp.Status),
		PaymentMethod: string(paymentMethod),
		ExpiryTime:    expiryTime,
		CreatedAt:     billResp.Created,
		RedirectURL:   billResp.RedirectURL,
	}
}

// ConvertPaymentResponseToJSON converts FlipPaymentResponse to JSON string for database storage
func (s *FlipService) ConvertPaymentResponseToJSON(paymentResp *models.FlipPaymentResponse) (string, error) {
	jsonBytes, err := json.Marshal(paymentResp)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// Helper function to convert GSALT units to IDR (same conversion as before)
func (s *FlipService) ConvertGSALTToIDR(gsaltUnits int64) int64 {
	// 1 GSALT = 100 units = 1000 IDR
	gsaltAmount := decimal.NewFromInt(gsaltUnits).Div(decimal.NewFromInt(100))
	idrAmount := gsaltAmount.Mul(decimal.NewFromInt(1000))
	return idrAmount.IntPart()
}

// GetSupportedPaymentMethods returns list of supported payment methods for Flip
func (s *FlipService) GetSupportedPaymentMethods() map[models.FlipPaymentMethod][]string {
	return map[models.FlipPaymentMethod][]string{
		models.FlipPaymentMethodQRIS: {"QRIS"},
		models.FlipPaymentMethodVirtualAccount: {
			string(models.FlipVABankBCA), string(models.FlipVABankBNI), string(models.FlipVABankBRI),
			string(models.FlipVABankMANDIRI), string(models.FlipVABankCIMB), string(models.FlipVABankPERMATA),
			string(models.FlipVABankBSI),
		},
		models.FlipPaymentMethodEWallet: {
			string(models.FlipEWalletOVO), string(models.FlipEWalletDANA), string(models.FlipEWalletGOPAY),
			string(models.FlipEWalletLINKAJA), string(models.FlipEWalletSHOPEEPAY),
		},
		models.FlipPaymentMethodRetailOutlet: {"RETAIL"},
	}
}

// ProcessWebhook processes incoming webhooks from Flip
func (s *FlipService) ProcessWebhook(ctx context.Context, payload models.FlipWebhookPayload) error {
	// Log webhook receipt
	fmt.Printf("Received Flip webhook for bill_link_id: %d, status: %s\n", payload.BillLinkID, payload.Status)

	// Here you would typically:
	// 1. Verify webhook authenticity (if Flip provides signature verification)
	// 2. Update transaction status in your database
	// 3. Trigger business logic based on payment status
	// 4. Send notifications to users

	// Example implementation (adjust based on your transaction model):
	/*
		tx := &models.Transaction{}
		if err := s.db.Where("payment_reference = ?", strconv.Itoa(payload.BillLinkID)).First(tx).Error; err != nil {
			return errors.NewNotFoundError("Transaction not found")
		}

		// Update transaction status
		switch payload.Status {
		case models.FlipStatusSuccessful:
			tx.Status = models.TransactionStatusCompleted
			// Add GSALT balance to user account
		case models.FlipStatusFailed, models.FlipStatusCancelled, models.FlipStatusExpired:
			tx.Status = models.TransactionStatusFailed
		}

		if err := s.db.Save(tx).Error; err != nil {
			return errors.NewInternalServerError(err, "Failed to update transaction")
		}
	*/

	return nil
}

// ========== GENERAL API METHODS ==========

// GetBalance retrieves the current balance from Flip account
func (s *FlipService) GetBalance(ctx context.Context) (*models.FlipBalance, error) {
	// Create HTTP request - General API uses V2
	url := s.client.GetFullURL("/v2/general/balance")
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", s.client.GetAuthHeader())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp models.FlipErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("flip API error: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("flip API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var balance models.FlipBalance
	if err := json.Unmarshal(body, &balance); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &balance, nil
}

// GetBanks retrieves list of available banks
func (s *FlipService) GetBanks(ctx context.Context) ([]models.FlipBank, error) {
	// Create HTTP request - General API uses V2
	url := s.client.GetFullURL("/v2/general/banks")
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", s.client.GetAuthHeader())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp models.FlipErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("flip API error: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("flip API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var banks []models.FlipBank
	if err := json.Unmarshal(body, &banks); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return banks, nil
}

// GetMaintenanceInfo retrieves maintenance information
func (s *FlipService) GetMaintenanceInfo(ctx context.Context) (*models.FlipMaintenanceInfo, error) {
	// Create HTTP request - General API uses V2
	url := s.client.GetFullURL("/v2/general/maintenance")
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", s.client.GetAuthHeader())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp models.FlipErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("flip API error: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("flip API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var maintenance models.FlipMaintenanceInfo
	if err := json.Unmarshal(body, &maintenance); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &maintenance, nil
}

// ========== DISBURSEMENT/MONEY TRANSFER API METHODS ==========

// BankAccountInquiry performs bank account inquiry
func (s *FlipService) BankAccountInquiry(ctx context.Context, req models.BankAccountInquiryRequest) (*models.BankAccountInquiryResponse, error) {
	// Create form data
	data := url.Values{}
	data.Set("account_number", req.AccountNumber)
	data.Set("bank_code", req.BankCode)
	if req.InquiryKey != "" {
		data.Set("inquiry_key", req.InquiryKey)
	}

	// Create HTTP request - Disbursement API uses V2
	url := s.client.GetFullURL("/v2/disbursement/bank-account-inquiry")
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", s.client.GetAuthHeader())
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := s.client.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp models.FlipErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("flip API error: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("flip API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var inquiry models.BankAccountInquiryResponse
	if err := json.Unmarshal(body, &inquiry); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &inquiry, nil
}

// CreateDisbursement creates a money transfer/disbursement
func (s *FlipService) CreateDisbursement(ctx context.Context, req models.DisbursementRequest) (*models.DisbursementResponse, error) {
	// Create form data
	data := url.Values{}
	data.Set("account_number", req.AccountNumber)
	data.Set("bank_code", req.BankCode)
	data.Set("amount", fmt.Sprintf("%d", req.Amount))
	if req.Remark != "" {
		data.Set("remark", req.Remark)
	}
	if req.RecipientCity != "" {
		data.Set("recipient_city", req.RecipientCity)
	}
	if req.BeneficiaryEmail != "" {
		data.Set("beneficiary_email", req.BeneficiaryEmail)
	}

	// Create HTTP request - Disbursement API uses V3
	url := s.client.GetFullURL("/v3/disbursement")
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", s.client.GetAuthHeader())
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Add required headers
	if req.IdempotencyKey != "" {
		httpReq.Header.Set("idempotency-key", req.IdempotencyKey)
	}
	if req.Timestamp != "" {
		httpReq.Header.Set("X-TIMESTAMP", req.Timestamp)
	}

	// Send request
	resp, err := s.client.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp models.FlipErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("flip API error: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("flip API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var disbursement models.DisbursementResponse
	if err := json.Unmarshal(body, &disbursement); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &disbursement, nil
}

// CreateSpecialDisbursement creates a special money transfer/disbursement
func (s *FlipService) CreateSpecialDisbursement(ctx context.Context, req models.SpecialDisbursementRequest) (*models.DisbursementResponse, error) {
	// Create form data
	data := url.Values{}
	data.Set("account_number", req.AccountNumber)
	data.Set("bank_code", req.BankCode)
	data.Set("amount", fmt.Sprintf("%d", req.Amount))
	data.Set("sender_country", req.SenderCountry)
	data.Set("sender_name", req.SenderName)
	data.Set("sender_address", req.SenderAddress)
	data.Set("direction", req.Direction)

	// Optional fields
	if req.Remark != "" {
		data.Set("remark", req.Remark)
	}
	if req.RecipientCity != "" {
		data.Set("recipient_city", req.RecipientCity)
	}
	if req.BeneficiaryEmail != "" {
		data.Set("beneficiary_email", req.BeneficiaryEmail)
	}
	if req.SenderPlaceOfBirth != "" {
		data.Set("sender_place_of_birth", req.SenderPlaceOfBirth)
	}
	if req.SenderDateOfBirth != "" {
		data.Set("sender_date_of_birth", req.SenderDateOfBirth)
	}
	if req.SenderIdentityType != "" {
		data.Set("sender_identity_type", req.SenderIdentityType)
	}
	if req.SenderIdentityNumber != "" {
		data.Set("sender_identity_number", req.SenderIdentityNumber)
	}
	if req.SenderJob != "" {
		data.Set("sender_job", req.SenderJob)
	}

	// Create HTTP request - Special disbursement uses V3
	url := s.client.GetFullURL("/v3/disbursement/create-special-disbursement")
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", s.client.GetAuthHeader())
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Add required headers
	if req.IdempotencyKey != "" {
		httpReq.Header.Set("idempotency-key", req.IdempotencyKey)
	}
	if req.Timestamp != "" {
		httpReq.Header.Set("X-TIMESTAMP", req.Timestamp)
	}

	// Send request
	resp, err := s.client.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp models.FlipErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("flip API error: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("flip API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var disbursement models.DisbursementResponse
	if err := json.Unmarshal(body, &disbursement); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &disbursement, nil
}

// GetDisbursements retrieves list of disbursements with pagination
func (s *FlipService) GetDisbursements(ctx context.Context, pagination bool, page int, sort string) (*models.DisbursementListResponse, error) {
	// Build query parameters
	params := url.Values{}
	if pagination {
		params.Set("pagination", "1")
	}
	if page > 0 {
		params.Set("page", fmt.Sprintf("%d", page))
	}
	if sort != "" {
		params.Set("sort", sort)
	}

	// Create HTTP request
	baseURL := s.client.GetFullURL("/v3/disbursement")
	if len(params) > 0 {
		baseURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", s.client.GetAuthHeader())

	// Send request
	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp models.FlipErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("flip API error: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("flip API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var disbursements models.DisbursementListResponse
	if err := json.Unmarshal(body, &disbursements); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &disbursements, nil
}

// GetDisbursementByID retrieves a disbursement by ID
func (s *FlipService) GetDisbursementByID(ctx context.Context, id string) (*models.DisbursementResponse, error) {
	// Create HTTP request
	url := s.client.GetFullURL("/v3/get-disbursement?id=" + id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", s.client.GetAuthHeader())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp models.FlipErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("flip API error: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("flip API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var disbursement models.DisbursementResponse
	if err := json.Unmarshal(body, &disbursement); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &disbursement, nil
}

// GetDisbursementByIdempotencyKey retrieves a disbursement by idempotency key
func (s *FlipService) GetDisbursementByIdempotencyKey(ctx context.Context, idempotencyKey string) (*models.DisbursementResponse, error) {
	// Create HTTP request
	url := s.client.GetFullURL("/v3/get-disbursement?idempotency-key=" + idempotencyKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", s.client.GetAuthHeader())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp models.FlipErrorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return nil, fmt.Errorf("flip API error: %s", errorResp.Message)
		}
		return nil, fmt.Errorf("flip API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var disbursement models.DisbursementResponse
	if err := json.Unmarshal(body, &disbursement); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &disbursement, nil
}

// ========== GSALT SPECIFIC DISBURSEMENT METHODS ==========

// CreateGSALTDisbursement creates a disbursement for GSALT withdrawal
func (s *FlipService) CreateGSALTDisbursement(ctx context.Context, req models.GSALTDisbursementRequest) (*models.GSALTDisbursementResponse, error) {
	// Generate timestamp for idempotency
	timestamp := time.Now().Format(time.RFC3339)

	// Create disbursement request
	disbursementReq := models.DisbursementRequest{
		AccountNumber:    req.AccountNumber,
		BankCode:         req.BankCode,
		Amount:           req.AmountIDR,
		Remark:           fmt.Sprintf("GSALT Withdrawal - %s", req.TransactionID),
		BeneficiaryEmail: req.RecipientEmail,
		IdempotencyKey:   req.IdempotencyKey,
		Timestamp:        timestamp,
	}

	// Add description if provided
	if req.Description != "" {
		disbursementReq.Remark = fmt.Sprintf("%s - %s", disbursementReq.Remark, req.Description)
	}

	// Create disbursement
	disbursement, err := s.CreateDisbursement(ctx, disbursementReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create disbursement: %w", err)
	}

	// Convert to GSALT response format
	response := &models.GSALTDisbursementResponse{
		TransactionID:  req.TransactionID,
		DisbursementID: disbursement.ID,
		Status:         models.DisbursementStatus(disbursement.Status),
		AmountGSALT:    req.AmountGSALT,
		AmountIDR:      req.AmountIDR,
		Fee:            disbursement.Fee,
		RecipientName:  req.RecipientName,
		AccountNumber:  req.AccountNumber,
		BankCode:       req.BankCode,
		Receipt:        disbursement.Receipt,
		IdempotencyKey: req.IdempotencyKey,
	}

	// Parse timestamps if available
	if disbursement.Timestamp != "" {
		if processedAt, err := time.Parse(time.RFC3339, disbursement.Timestamp); err == nil {
			response.ProcessedAt = &processedAt
		}
	}
	if disbursement.TimeServed != "" {
		if completedAt, err := time.Parse(time.RFC3339, disbursement.TimeServed); err == nil {
			response.CompletedAt = &completedAt
		}
	}

	return response, nil
}

// GetGSALTDisbursementStatus retrieves the status of a GSALT disbursement
func (s *FlipService) GetGSALTDisbursementStatus(ctx context.Context, transactionID string) (*models.GSALTDisbursementResponse, error) {
	// First try to get by ID (if transactionID is the disbursement ID)
	disbursement, err := s.GetDisbursementByID(ctx, transactionID)
	if err != nil {
		// If not found by ID, try by idempotency key
		disbursement, err = s.GetDisbursementByIdempotencyKey(ctx, transactionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get disbursement: %w", err)
		}
	}

	// Convert to GSALT response format
	response := &models.GSALTDisbursementResponse{
		TransactionID:  transactionID,
		DisbursementID: disbursement.ID,
		Status:         models.DisbursementStatus(disbursement.Status),
		AmountIDR:      disbursement.Amount,
		Fee:            disbursement.Fee,
		RecipientName:  disbursement.RecipientName,
		AccountNumber:  disbursement.AccountNumber,
		BankCode:       disbursement.BankCode,
		Receipt:        disbursement.Receipt,
		IdempotencyKey: disbursement.IdempotencyKey,
	}

	// Calculate GSALT amount (reverse conversion)
	response.AmountGSALT = s.ConvertIDRToGSALT(disbursement.Amount)

	// Parse timestamps if available
	if disbursement.Timestamp != "" {
		if processedAt, err := time.Parse(time.RFC3339, disbursement.Timestamp); err == nil {
			response.ProcessedAt = &processedAt
		}
	}
	if disbursement.TimeServed != "" {
		if completedAt, err := time.Parse(time.RFC3339, disbursement.TimeServed); err == nil {
			response.CompletedAt = &completedAt
		}
	}

	return response, nil
}

// ValidateBankAccount validates a bank account before creating disbursement
func (s *FlipService) ValidateBankAccount(ctx context.Context, accountNumber, bankCode string) (*models.BankAccountInquiryResponse, error) {
	req := models.BankAccountInquiryRequest{
		AccountNumber: accountNumber,
		BankCode:      bankCode,
		InquiryKey:    fmt.Sprintf("gsalt_%s_%s_%d", bankCode, accountNumber, time.Now().Unix()),
	}

	return s.BankAccountInquiry(ctx, req)
}

// ConvertIDRToGSALT converts IDR amount to GSALT units (reverse of ConvertGSALTToIDR)
func (s *FlipService) ConvertIDRToGSALT(idrAmount int64) int64 {
	// 1 GSALT = 1000 IDR, so divide by 1000
	return idrAmount / 1000
}

// GetSupportedBanks returns list of banks that support disbursement
func (s *FlipService) GetSupportedBanks(ctx context.Context) ([]models.FlipBank, error) {
	banks, err := s.GetBanks(ctx)
	if err != nil {
		return nil, err
	}

	// Filter banks that support disbursement (have status "OPERATIONAL")
	var supportedBanks []models.FlipBank
	for _, bank := range banks {
		if bank.Status == "OPERATIONAL" {
			supportedBanks = append(supportedBanks, bank)
		}
	}

	return supportedBanks, nil
}

// CheckDisbursementAvailability checks if disbursement service is available
func (s *FlipService) CheckDisbursementAvailability(ctx context.Context) (bool, error) {
	maintenance, err := s.GetMaintenanceInfo(ctx)
	if err != nil {
		return false, err
	}

	// If general maintenance is active, disbursement is not available
	if maintenance.Maintenance {
		return false, nil
	}

	// Check if any critical banks are under maintenance
	criticalBanks := []string{"bca", "bni", "bri", "mandiri", "cimb"}
	maintenanceCount := 0

	for _, bank := range maintenance.Banks {
		for _, critical := range criticalBanks {
			if bank.BankCode == critical && bank.Status != "OPERATIONAL" {
				maintenanceCount++
				break
			}
		}
	}

	// If more than half of critical banks are under maintenance, consider service unavailable
	return maintenanceCount < len(criticalBanks)/2, nil
}
