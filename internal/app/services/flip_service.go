package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/infrastructures"
)

// FlipService provides API client methods for Flip API
type FlipService struct {
	client *infrastructures.FlipClient
}

// NewFlipService creates a new FlipService instance
func NewFlipService(flipClient *infrastructures.FlipClient) *FlipService {
	return &FlipService{
		client: flipClient,
	}
}

// ========== GENERAL API METHODS (V2) ==========

// GetBalance retrieves the current balance from Flip account
// GET {{base_url_v2}}/general/balance
func (s *FlipService) GetBalance(ctx context.Context) (*models.FlipBalance, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.client.GetFullURL("/v2/general/balance"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", s.client.GetAuthHeader())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleAPIError(resp.StatusCode, body)
	}

	var balance models.FlipBalance
	if err := json.Unmarshal(body, &balance); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &balance, nil
}

// GetBanks retrieves list of available banks
// GET {{base_url_v2}}/general/banks
func (s *FlipService) GetBanks(ctx context.Context) ([]models.FlipBank, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.client.GetFullURL("/v2/general/banks"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", s.client.GetAuthHeader())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleAPIError(resp.StatusCode, body)
	}

	var banks []models.FlipBank
	if err := json.Unmarshal(body, &banks); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return banks, nil
}

// GetMaintenanceInfo retrieves maintenance information
// GET {{base_url_v2}}/general/maintenance
func (s *FlipService) GetMaintenanceInfo(ctx context.Context) (*models.FlipMaintenanceInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.client.GetFullURL("/v2/general/maintenance"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", s.client.GetAuthHeader())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleAPIError(resp.StatusCode, body)
	}

	var maintenance models.FlipMaintenanceInfo
	if err := json.Unmarshal(body, &maintenance); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &maintenance, nil
}

// ========== ACCEPT PAYMENT V2 API METHODS ==========

// CreateBill creates a new payment bill using V2 API
// POST {{base_url_v2}}/pwf/bill
func (s *FlipService) CreateBill(ctx context.Context, req models.CreateBillRequest) (*models.PaymentGatewayResponse, error) {
	// Convert request to form data
	formData := s.createBillRequestToFormData(req)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.client.GetFullURL("/v2/pwf/bill"), strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", s.client.GetAuthHeader())
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, s.handleAPIError(resp.StatusCode, body)
	}

	var billResp models.PaymentGatewayResponse
	if err := json.Unmarshal(body, &billResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &billResp, nil
}

// GetBill retrieves a specific bill by ID
// GET {{base_url_v2}}/pwf/:bill_id/bill
func (s *FlipService) GetBill(ctx context.Context, billID int) (*models.GetBillResponse, error) {
	url := s.client.GetFullURL(fmt.Sprintf("/v2/pwf/%d/bill", billID))
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", s.client.GetAuthHeader())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleAPIError(resp.StatusCode, body)
	}

	var billStatus models.GetBillResponse
	if err := json.Unmarshal(body, &billStatus); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &billStatus, nil
}

// GetAllBills retrieves all bills
// GET {{base_url_v2}}/pwf/bill
func (s *FlipService) GetAllBills(ctx context.Context) ([]models.GetBillResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.client.GetFullURL("/v2/pwf/bill"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", s.client.GetAuthHeader())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleAPIError(resp.StatusCode, body)
	}

	var bills []models.GetBillResponse
	if err := json.Unmarshal(body, &bills); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return bills, nil
}

// ========== MONEY TRANSFER/DISBURSEMENT API METHODS ==========

// BankAccountInquiry validates bank account information
// POST {{base_url_v2}}/disbursement/bank-account-inquiry
func (s *FlipService) BankAccountInquiry(ctx context.Context, req models.BankAccountInquiryRequest) (*models.BankAccountInquiryResponse, error) {
	formData := url.Values{}
	formData.Set("account_number", req.AccountNumber)
	formData.Set("bank_code", req.BankCode)
	if req.InquiryKey != "" {
		formData.Set("inquiry_key", req.InquiryKey)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.client.GetFullURL("/v2/disbursement/bank-account-inquiry"), strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", s.client.GetAuthHeader())
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleAPIError(resp.StatusCode, body)
	}

	var inquiry models.BankAccountInquiryResponse
	if err := json.Unmarshal(body, &inquiry); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &inquiry, nil
}

// CreateDisbursement creates a new disbursement
// POST {{base_url_v2}}/disbursement
func (s *FlipService) CreateDisbursement(ctx context.Context, req models.DisbursementRequest) (*models.DisbursementResponse, error) {
	formData := url.Values{}
	formData.Set("account_number", req.AccountNumber)
	formData.Set("bank_code", req.BankCode)
	formData.Set("amount", strconv.FormatInt(req.Amount, 10))
	if req.Remark != "" {
		formData.Set("remark", req.Remark)
	}
	if req.RecipientCity != "" {
		formData.Set("recipient_city", req.RecipientCity)
	}
	if req.BeneficiaryEmail != "" {
		formData.Set("beneficiary_email", req.BeneficiaryEmail)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.client.GetFullURL("/v2/disbursement"), strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", s.client.GetAuthHeader())
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Add idempotency headers if provided
	if req.IdempotencyKey != "" {
		httpReq.Header.Set("idempotency-key", req.IdempotencyKey)
	}
	if req.Timestamp != "" {
		httpReq.Header.Set("X-Idempotency-Timestamp", req.Timestamp)
	}

	resp, err := s.client.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, s.handleAPIError(resp.StatusCode, body)
	}

	var disbursement models.DisbursementResponse
	if err := json.Unmarshal(body, &disbursement); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &disbursement, nil
}

// GetDisbursements retrieves disbursements with optional pagination
// GET {{base_url_v2}}/disbursement
func (s *FlipService) GetDisbursements(ctx context.Context, pagination bool, page int, sort string) (*models.DisbursementListResponse, error) {
	baseURL := s.client.GetFullURL("/v2/disbursement")

	// Build query parameters
	params := url.Values{}
	if pagination {
		params.Set("pagination", "1")
		if page > 0 {
			params.Set("page", strconv.Itoa(page))
		}
	}
	if sort != "" {
		params.Set("sort", sort)
	}

	fullURL := baseURL
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", s.client.GetAuthHeader())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleAPIError(resp.StatusCode, body)
	}

	var disbursements models.DisbursementListResponse
	if err := json.Unmarshal(body, &disbursements); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &disbursements, nil
}

// GetDisbursementByID retrieves a specific disbursement by ID
// GET {{base_url_v2}}/disbursement/:id
func (s *FlipService) GetDisbursementByID(ctx context.Context, id string) (*models.DisbursementResponse, error) {
	url := s.client.GetFullURL(fmt.Sprintf("/v2/disbursement/%s", id))
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", s.client.GetAuthHeader())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleAPIError(resp.StatusCode, body)
	}

	var disbursement models.DisbursementResponse
	if err := json.Unmarshal(body, &disbursement); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &disbursement, nil
}

// GetDisbursementByIdempotencyKey retrieves a disbursement by idempotency key
// GET {{base_url_v2}}/get-disbursement
func (s *FlipService) GetDisbursementByIdempotencyKey(ctx context.Context, idempotencyKey string) (*models.DisbursementResponse, error) {
	params := url.Values{}
	params.Set("idempotency_key", idempotencyKey)

	fullURL := s.client.GetFullURL("/v2/get-disbursement") + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", s.client.GetAuthHeader())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, s.handleAPIError(resp.StatusCode, body)
	}

	var disbursement models.DisbursementResponse
	if err := json.Unmarshal(body, &disbursement); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &disbursement, nil
}

// CreatePayment creates a new payment using Flip's API
func (s *FlipService) CreatePayment(req models.FlipPaymentRequestData) (*models.FlipPaymentResponse, error) {
	// Convert request to form data
	formData := url.Values{}
	formData.Set("title", req.Title)
	formData.Set("amount", strconv.FormatInt(req.Amount, 10))
	formData.Set("payment_method", string(req.PaymentMethod))
	formData.Set("expiry_hours", strconv.Itoa(req.ExpiryHours))
	formData.Set("reference_id", *req.ReferenceID)

	if req.CustomerName != nil {
		formData.Set("customer_name", *req.CustomerName)
	}
	if req.CustomerEmail != nil {
		formData.Set("customer_email", *req.CustomerEmail)
	}
	if req.CustomerPhone != nil {
		formData.Set("customer_phone", *req.CustomerPhone)
	}
	if req.CustomerAddress != nil {
		formData.Set("customer_address", *req.CustomerAddress)
	}
	if req.RedirectURL != nil {
		formData.Set("redirect_url", *req.RedirectURL)
	}

	// Add item details if present
	for i, item := range req.ItemDetails {
		prefix := fmt.Sprintf("item_details[%d]", i)
		formData.Set(prefix+"[name]", item.Name)
		formData.Set(prefix+"[price]", strconv.FormatInt(item.Price, 10))
		formData.Set(prefix+"[quantity]", strconv.Itoa(item.Quantity))
		if item.Desc != "" {
			formData.Set(prefix+"[desc]", item.Desc)
		}
	}

	httpReq, err := http.NewRequest("POST", s.client.GetFullURL("/v2/pwf/payment"), strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", s.client.GetAuthHeader())
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, s.handleAPIError(resp.StatusCode, body)
	}

	var paymentResp models.FlipPaymentResponse
	if err := json.Unmarshal(body, &paymentResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &paymentResp, nil
}

// ========== HELPER METHODS ==========

// createBillRequestToFormData converts CreateBillRequest to form data
func (s *FlipService) createBillRequestToFormData(req models.CreateBillRequest) url.Values {
	formData := url.Values{}

	formData.Set("title", req.Title)
	formData.Set("type", req.Type)
	formData.Set("step", req.Step)
	formData.Set("amount", strconv.FormatInt(req.Amount, 10))

	if req.ExpiredDate != "" {
		formData.Set("expired_date", req.ExpiredDate)
	}
	if req.RedirectURL != "" {
		formData.Set("redirect_url", req.RedirectURL)
	}
	if req.IsAddressRequired {
		formData.Set("is_address_required", "1")
	}
	if req.SenderAddress != "" {
		formData.Set("sender_address", req.SenderAddress)
	}
	if req.IsPhoneNumberRequired {
		formData.Set("is_phone_number_required", "1")
	}
	if req.SenderPhoneNumber != "" {
		formData.Set("sender_phone_number", req.SenderPhoneNumber)
	}
	if req.SenderName != "" {
		formData.Set("sender_name", req.SenderName)
	}
	if req.SenderEmail != "" {
		formData.Set("sender_email", req.SenderEmail)
	}
	if req.SenderBank != "" {
		formData.Set("sender_bank", req.SenderBank)
	}
	if req.SenderBankType != "" {
		formData.Set("sender_bank_type", req.SenderBankType)
	}
	if req.ReferenceID != "" {
		formData.Set("reference_id", req.ReferenceID)
	}
	if req.ChargeFee {
		formData.Set("charge_fee", "1")
	}

	// Add item details if present
	for i, item := range req.ItemDetails {
		prefix := fmt.Sprintf("item_details[%d]", i)
		if item.ID != "" {
			formData.Set(prefix+"[id]", item.ID)
		}
		formData.Set(prefix+"[name]", item.Name)
		formData.Set(prefix+"[price]", strconv.FormatInt(item.Price, 10))
		formData.Set(prefix+"[quantity]", strconv.Itoa(item.Quantity))
		if item.Desc != "" {
			formData.Set(prefix+"[desc]", item.Desc)
		}
		if item.ImageURL != "" {
			formData.Set(prefix+"[image_url]", item.ImageURL)
		}
	}

	return formData
}

// handleAPIError handles API error responses
func (s *FlipService) handleAPIError(statusCode int, body []byte) error {
	var apiError models.FlipErrorResponse
	if err := json.Unmarshal(body, &apiError); err != nil {
		return fmt.Errorf("API error (status %d): %s", statusCode, string(body))
	}
	return fmt.Errorf("API error (status %d): %s", statusCode, apiError.Message)
}
