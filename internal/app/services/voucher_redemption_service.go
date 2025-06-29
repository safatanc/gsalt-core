package services

import (
	"github.com/google/uuid"
	"github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/app/pkg"
	"github.com/safatanc/gsalt-core/internal/infrastructures"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type VoucherRedemptionService struct {
	db                 *gorm.DB
	validator          *infrastructures.Validator
	voucherService     *VoucherService
	accountService     *AccountService
	transactionService *TransactionService
}

func NewVoucherRedemptionService(db *gorm.DB, validator *infrastructures.Validator, voucherService *VoucherService, accountService *AccountService, transactionService *TransactionService) *VoucherRedemptionService {
	return &VoucherRedemptionService{
		db:                 db,
		validator:          validator,
		voucherService:     voucherService,
		accountService:     accountService,
		transactionService: transactionService,
	}
}

func (s *VoucherRedemptionService) CreateRedemption(req *models.VoucherRedemptionCreateDto) (*models.VoucherRedemption, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, errors.NewBadRequestError(err.Error())
	}

	// Parse UUIDs
	voucherUUID, err := uuid.Parse(req.VoucherID)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid voucher ID format")
	}

	accountUUID, err := uuid.Parse(req.AccountID)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid account ID format")
	}

	// Check if redemption already exists for this voucher and account
	var existingRedemption models.VoucherRedemption
	err = s.db.Where("voucher_id = ? AND account_id = ?", voucherUUID, accountUUID).First(&existingRedemption).Error
	if err == nil {
		return nil, errors.NewBadRequestError("Voucher already redeemed by this account")
	}

	// Create redemption
	redemption := &models.VoucherRedemption{
		VoucherID: voucherUUID,
		AccountID: accountUUID,
	}

	// Parse transaction ID if provided
	if req.TransactionID != nil {
		transactionUUID, err := uuid.Parse(*req.TransactionID)
		if err != nil {
			return nil, errors.NewBadRequestError("Invalid transaction ID format")
		}
		redemption.TransactionID = &transactionUUID
	}

	if err := s.db.Create(redemption).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to create voucher redemption")
	}

	return redemption, nil
}

func (s *VoucherRedemptionService) GetRedemption(redemptionId string) (*models.VoucherRedemption, error) {
	redemptionUUID, err := uuid.Parse(redemptionId)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid redemption ID format")
	}

	var redemption models.VoucherRedemption
	err = s.db.Where("id = ?", redemptionUUID).First(&redemption).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Voucher redemption not found")
		}
		return nil, errors.NewInternalServerError(err, "Failed to get voucher redemption")
	}

	return &redemption, nil
}

func (s *VoucherRedemptionService) GetRedemptionsByAccount(accountId string, pagination *models.PaginationRequest) (*models.Pagination[[]models.VoucherRedemption], error) {
	accountUUID, err := uuid.Parse(accountId)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid account ID format")
	}

	// Set defaults
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}
	if pagination.Page <= 0 {
		pagination.Page = 1
	}

	offset := (pagination.Page - 1) * pagination.Limit

	// Count total items
	var totalItems int64
	if err := s.db.Model(&models.VoucherRedemption{}).Where("account_id = ?", accountUUID).Count(&totalItems).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to count voucher redemptions")
	}

	var redemptions []models.VoucherRedemption
	query := s.db.Where("account_id = ?", accountUUID).Order("redeemed_at DESC")

	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err = query.Find(&redemptions).Error
	if err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to get voucher redemptions")
	}

	// Calculate pagination metadata
	totalPages := int((totalItems + int64(pagination.Limit) - 1) / int64(pagination.Limit))
	hasNext := pagination.Page < totalPages
	hasPrev := pagination.Page > 1

	result := &models.Pagination[[]models.VoucherRedemption]{
		Page:       pagination.Page,
		Limit:      pagination.Limit,
		TotalPages: totalPages,
		TotalItems: int(totalItems),
		HasNext:    hasNext,
		HasPrev:    hasPrev,
		Items:      redemptions,
	}

	return result, nil
}

func (s *VoucherRedemptionService) GetRedemptionsByVoucher(voucherId string, pagination *models.PaginationRequest) (*models.Pagination[[]models.VoucherRedemption], error) {
	voucherUUID, err := uuid.Parse(voucherId)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid voucher ID format")
	}

	// Set defaults
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}
	if pagination.Page <= 0 {
		pagination.Page = 1
	}

	offset := (pagination.Page - 1) * pagination.Limit

	// Count total items
	var totalItems int64
	if err := s.db.Model(&models.VoucherRedemption{}).Where("voucher_id = ?", voucherUUID).Count(&totalItems).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to count voucher redemptions")
	}

	var redemptions []models.VoucherRedemption
	query := s.db.Where("voucher_id = ?", voucherUUID).Order("redeemed_at DESC")

	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err = query.Find(&redemptions).Error
	if err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to get voucher redemptions")
	}

	// Calculate pagination metadata
	totalPages := int((totalItems + int64(pagination.Limit) - 1) / int64(pagination.Limit))
	hasNext := pagination.Page < totalPages
	hasPrev := pagination.Page > 1

	result := &models.Pagination[[]models.VoucherRedemption]{
		Page:       pagination.Page,
		Limit:      pagination.Limit,
		TotalPages: totalPages,
		TotalItems: int(totalItems),
		HasNext:    hasNext,
		HasPrev:    hasPrev,
		Items:      redemptions,
	}

	return result, nil
}

func (s *VoucherRedemptionService) UpdateRedemption(redemptionId string, req *models.VoucherRedemptionUpdateDto) (*models.VoucherRedemption, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, errors.NewBadRequestError(err.Error())
	}

	redemption, err := s.GetRedemption(redemptionId)
	if err != nil {
		return nil, err
	}

	// Update transaction ID if provided
	if req.TransactionID != nil {
		transactionUUID, err := uuid.Parse(*req.TransactionID)
		if err != nil {
			return nil, errors.NewBadRequestError("Invalid transaction ID format")
		}
		redemption.TransactionID = &transactionUUID
	}

	if err := s.db.Save(redemption).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to update voucher redemption")
	}

	return redemption, nil
}

func (s *VoucherRedemptionService) RedeemVoucher(accountId, voucherCode string) (*models.VoucherRedemption, *models.Transaction, error) {
	// Validate voucher
	voucher, err := s.voucherService.ValidateVoucher(voucherCode)
	if err != nil {
		return nil, nil, err
	}

	// Verify account exists
	account, err := s.accountService.GetAccount(accountId)
	if err != nil {
		return nil, nil, err
	}

	// Check if voucher was already redeemed by this account
	var existingRedemption models.VoucherRedemption
	err = s.db.Where("voucher_id = ? AND account_id = ?", voucher.ID, account.ConnectID).First(&existingRedemption).Error
	if err == nil {
		return nil, nil, errors.NewBadRequestError("Voucher already redeemed by this account")
	}

	// Start database transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var transaction *models.Transaction

	// Process voucher based on type
	switch voucher.Type {
	case models.VoucherTypeBalance:
		// Calculate GSALT units from voucher value
		var amountGsaltUnits int64
		if voucher.Currency == "GSALT" {
			amountGsaltUnits = voucher.Value.Mul(decimal.NewFromInt(100)).IntPart()
		} else if voucher.Currency == "IDR" {
			// Convert IDR to GSALT units (1000 IDR = 1 GSALT = 100 units)
			gsaltDecimal := voucher.Value.Div(decimal.NewFromInt(1000))
			amountGsaltUnits = gsaltDecimal.Mul(decimal.NewFromInt(100)).IntPart()
		} else {
			tx.Rollback()
			return nil, nil, errors.NewBadRequestError("Unsupported voucher currency for balance type")
		}

		// Add balance to account (in GSALT units)
		account.Balance += amountGsaltUnits
		if err := tx.Save(account).Error; err != nil {
			tx.Rollback()
			return nil, nil, errors.NewInternalServerError(err, "Failed to update account balance")
		}

		// Create transaction record
		transaction = &models.Transaction{
			AccountID:        account.ConnectID,
			Type:             models.TransactionTypeVoucherRedemption,
			AmountGsaltUnits: amountGsaltUnits,
			Currency:         "GSALT",
			Status:           models.TransactionStatusCompleted,
			Description:      pkg.StringPtr("Balance voucher redemption: " + voucher.Name),
			VoucherCode:      &voucher.Code,
		}

	case models.VoucherTypeLoyaltyPoints:
		// Convert loyalty points to GSALT units (1 point = 0.01 GSALT = 1 unit)
		var amountGsaltUnits int64
		if voucher.LoyaltyPointsValue != nil {
			amountGsaltUnits = *voucher.LoyaltyPointsValue
			account.Balance += amountGsaltUnits
			if err := tx.Save(account).Error; err != nil {
				tx.Rollback()
				return nil, nil, errors.NewInternalServerError(err, "Failed to update account balance")
			}
		}

		// Create transaction record
		transaction = &models.Transaction{
			AccountID:        account.ConnectID,
			Type:             models.TransactionTypeVoucherRedemption,
			AmountGsaltUnits: amountGsaltUnits,
			Currency:         "GSALT",
			Status:           models.TransactionStatusCompleted,
			Description:      pkg.StringPtr("Loyalty points voucher redemption: " + voucher.Name),
			VoucherCode:      &voucher.Code,
		}

	case models.VoucherTypeDiscount:
		// For discount vouchers, we just record the redemption
		// The actual discount application will be handled during payment processing
		// No balance change for discount vouchers
		transaction = &models.Transaction{
			AccountID:        account.ConnectID,
			Type:             models.TransactionTypeVoucherRedemption,
			AmountGsaltUnits: 0, // No balance change for discount vouchers
			Currency:         "GSALT",
			Status:           models.TransactionStatusCompleted,
			Description:      pkg.StringPtr("Discount voucher redemption: " + voucher.Name),
			VoucherCode:      &voucher.Code,
		}
	}

	// Create transaction
	if err := tx.Create(transaction).Error; err != nil {
		tx.Rollback()
		return nil, nil, errors.NewInternalServerError(err, "Failed to create voucher redemption transaction")
	}

	// Convert transaction ID string to UUID
	transactionUUID, err := uuid.Parse(transaction.ID)
	if err != nil {
		tx.Rollback()
		return nil, nil, errors.NewInternalServerError(err, "Failed to parse transaction ID")
	}

	// Create redemption record
	redemption := &models.VoucherRedemption{
		VoucherID:     voucher.ID,
		AccountID:     account.ConnectID,
		TransactionID: &transactionUUID,
	}

	if err := tx.Create(redemption).Error; err != nil {
		tx.Rollback()
		return nil, nil, errors.NewInternalServerError(err, "Failed to create voucher redemption")
	}

	// Increment voucher redeem count
	voucher.CurrentRedeemCount++
	if voucher.CurrentRedeemCount >= voucher.MaxRedeemCount {
		voucher.Status = models.VoucherStatusRedeemed
	}

	if err := tx.Save(voucher).Error; err != nil {
		tx.Rollback()
		return nil, nil, errors.NewInternalServerError(err, "Failed to update voucher redeem count")
	}

	tx.Commit()
	return redemption, transaction, nil
}

func (s *VoucherRedemptionService) DeleteRedemption(redemptionId string) error {
	redemption, err := s.GetRedemption(redemptionId)
	if err != nil {
		return err
	}

	if err := s.db.Delete(redemption).Error; err != nil {
		return errors.NewInternalServerError(err, "Failed to delete voucher redemption")
	}

	return nil
}
