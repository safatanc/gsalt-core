package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/infrastructures"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type XenditService struct {
	client *infrastructures.XenditClient
	db     *gorm.DB
}

func NewXenditService(xenditClient *infrastructures.XenditClient, db *gorm.DB) *XenditService {
	return &XenditService{
		client: xenditClient,
		db:     db,
	}
}

// CreatePaymentRequest creates a new payment request in Xendit
func (s *XenditService) CreatePaymentRequest(ctx context.Context, data models.PaymentRequestData) (*models.PaymentResponse, error) {
	// Generate external reference ID
	externalID := fmt.Sprintf("gsalt-topup-%s-%d", data.TransactionID, time.Now().Unix())

	// Set default expiry time (24 hours from now)
	expiryTime := time.Now().Add(24 * time.Hour)
	if data.ExpiryTime != nil {
		expiryTime = *data.ExpiryTime
	}

	// Build payment method request
	paymentMethod := s.buildPaymentMethodRequest(data)

	// Create payment request body
	requestBody := models.CreatePaymentRequestBody{
		ReferenceID:   externalID,
		Amount:        float64(data.Amount),
		Currency:      data.Currency,
		Country:       "ID", // Indonesia
		PaymentMethod: paymentMethod,
		Description:   data.Description,
		ExpiresAt:     &expiryTime,
	}

	// Set customer information if provided
	if data.CustomerName != nil || data.CustomerEmail != nil || data.CustomerPhone != nil {
		customer := &models.CustomerRequest{
			ReferenceID: data.TransactionID,
		}
		if data.CustomerName != nil {
			customer.GivenNames = *data.CustomerName
		}
		if data.CustomerEmail != nil {
			customer.Email = *data.CustomerEmail
		}
		if data.CustomerPhone != nil {
			customer.MobileNumber = *data.CustomerPhone
		}
		requestBody.Customer = customer
	}

	// Set callback URLs
	requestBody.CallbackURLs = &models.CallbackURLs{
		Success: s.getCallbackURL("success"),
		Failure: s.getCallbackURL("failure"),
	}

	// Make HTTP request to Xendit API
	resp, err := s.makeXenditRequest(ctx, "POST", "/payment_requests", requestBody)
	if err != nil {
		return nil, err
	}

	var xenditResp models.PaymentRequestResponse
	if err := json.Unmarshal(resp, &xenditResp); err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to parse Xendit response")
	}

	// Convert to our response format
	return s.convertToPaymentResponse(xenditResp), nil
}

// GetPaymentRequest retrieves payment request details
func (s *XenditService) GetPaymentRequest(ctx context.Context, paymentRequestID string) (*models.PaymentResponse, error) {
	endpoint := fmt.Sprintf("/payment_requests/%s", paymentRequestID)
	resp, err := s.makeXenditRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var xenditResp models.PaymentRequestResponse
	if err := json.Unmarshal(resp, &xenditResp); err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to parse Xendit response")
	}

	return s.convertToPaymentResponse(xenditResp), nil
}

// CreateTopupPayment creates a payment request for GSALT topup
func (s *XenditService) CreateTopupPayment(ctx context.Context, transactionID string, amountIDR int64, paymentMethod models.PaymentMethod, customerInfo *models.Account) (*models.PaymentResponse, error) {
	data := models.PaymentRequestData{
		TransactionID: transactionID,
		Amount:        amountIDR,
		Currency:      "IDR",
		PaymentMethod: paymentMethod,
		Description:   fmt.Sprintf("GSALT Topup - %s", transactionID),
	}

	// Set customer information from account if available
	// Note: Account model might need additional fields like Email and Phone
	if customerInfo != nil {
		// For now, we'll skip setting customer info since Account model doesn't have these fields
		// You can extend the Account model or create a Customer model later
	}

	return s.CreatePaymentRequest(ctx, data)
}

// Helper functions

func (s *XenditService) buildPaymentMethodRequest(data models.PaymentRequestData) models.PaymentMethodRequest {
	paymentMethod := models.PaymentMethodRequest{
		Reusability: "ONE_TIME_USE",
	}

	switch data.PaymentMethod {
	case models.PaymentMethodQRIS:
		paymentMethod.Type = "QR_CODE"
		paymentMethod.QRCode = &models.QRCodeRequest{
			ChannelCode: "QRIS",
		}
	case models.PaymentMethodVirtualAccount:
		paymentMethod.Type = "VIRTUAL_ACCOUNT"
		channelCode := "BCA" // Default
		if data.VABank != nil {
			channelCode = string(*data.VABank)
		}
		paymentMethod.VirtualAccount = &models.VirtualAccountRequest{
			ChannelCode: channelCode,
		}
	case models.PaymentMethodEWallet:
		paymentMethod.Type = "EWALLET"
		channelCode := "OVO" // Default
		if data.EWalletProvider != nil {
			channelCode = string(*data.EWalletProvider)
		}
		paymentMethod.EWallet = &models.EWalletRequest{
			ChannelCode: channelCode,
		}
	}

	return paymentMethod
}

func (s *XenditService) makeXenditRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	url := s.client.Config.BaseURL + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, errors.NewInternalServerError(err, "Failed to marshal request body")
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to create HTTP request")
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", s.client.Config.SecretKey))

	resp, err := s.client.HTTPClient.Do(req)
	if err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to make request to Xendit")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to read response body")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.NewInternalServerError(fmt.Errorf("xendit API error: %s", string(respBody)), "Xendit API request failed")
	}

	return respBody, nil
}

func (s *XenditService) convertToPaymentResponse(xenditResp models.PaymentRequestResponse) *models.PaymentResponse {
	response := &models.PaymentResponse{
		PaymentRequestID: xenditResp.ID,
		Status:           xenditResp.Status,
		Amount:           int64(xenditResp.Amount),
		Currency:         xenditResp.Currency,
		PaymentMethod:    xenditResp.PaymentMethod.Type,
		CreatedAt:        xenditResp.Created,
		ExpiryTime:       xenditResp.ExpiresAt,
	}

	// Set payment method specific details
	switch xenditResp.PaymentMethod.Type {
	case "QR_CODE":
		if xenditResp.PaymentMethod.QRCode != nil {
			response.QRCode = &xenditResp.PaymentMethod.QRCode.QRString
		}
	case "VIRTUAL_ACCOUNT":
		if xenditResp.PaymentMethod.VirtualAccount != nil {
			response.VirtualAccount = &models.VirtualAccountDetails{
				BankCode: xenditResp.PaymentMethod.VirtualAccount.ChannelCode,
				// AccountNumber will be in channel_properties
			}
			if props := xenditResp.PaymentMethod.VirtualAccount.ChannelProperties; props != nil {
				if accountNumber, ok := props["virtual_account_number"].(string); ok {
					response.VirtualAccount.AccountNumber = accountNumber
				}
			}
		}
	case "EWALLET":
		if xenditResp.PaymentMethod.EWallet != nil {
			response.EWallet = &models.EWalletDetails{
				Provider: xenditResp.PaymentMethod.EWallet.ChannelCode,
			}
			if props := xenditResp.PaymentMethod.EWallet.ChannelProperties; props != nil {
				if checkoutURL, ok := props["checkout_url"].(string); ok {
					response.EWallet.CheckoutURL = &checkoutURL
				}
			}
		}
	}

	// Set actions
	for _, action := range xenditResp.Actions {
		response.Actions = append(response.Actions, models.PaymentAction{
			Action: action.Action,
			URL:    action.URL,
			Method: action.Method,
		})
	}

	return response
}

// ConvertPaymentResponseToJSON converts PaymentResponse to JSON string for database storage
func (s *XenditService) ConvertPaymentResponseToJSON(paymentResp *models.PaymentResponse) (string, error) {
	jsonBytes, err := json.Marshal(paymentResp)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

func (s *XenditService) getCallbackURL(callbackType string) string {
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return fmt.Sprintf("%s/api/v1/webhooks/xendit/%s", baseURL, callbackType)
}

// Helper function to convert GSALT units to IDR
func (s *XenditService) ConvertGSALTToIDR(gsaltUnits int64) int64 {
	// 1 GSALT = 100 units = 1000 IDR
	gsaltAmount := decimal.NewFromInt(gsaltUnits).Div(decimal.NewFromInt(100))
	idrAmount := gsaltAmount.Mul(decimal.NewFromInt(1000))
	return idrAmount.IntPart()
}

// GetSupportedPaymentMethods returns list of supported payment methods
func (s *XenditService) GetSupportedPaymentMethods() map[models.PaymentMethod][]string {
	return map[models.PaymentMethod][]string{
		models.PaymentMethodQRIS: {"QRIS"},
		models.PaymentMethodVirtualAccount: {
			string(models.VABankBCA), string(models.VABankBNI), string(models.VABankBRI),
			string(models.VABankMANDIRI), string(models.VABankBSI), string(models.VABankPERMATA),
		},
		models.PaymentMethodEWallet: {
			string(models.EWalletOVO), string(models.EWalletDANA), string(models.EWalletGOPAY),
			string(models.EWalletLINKAJA), string(models.EWalletSHOPEEPAY),
		},
	}
}
