package services

import (
	"time"

	"github.com/google/uuid"
	"github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/safatanc/gsalt-core/internal/infrastructures"
	"gorm.io/gorm"
)

type VoucherService struct {
	db        *gorm.DB
	validator *infrastructures.Validator
}

func NewVoucherService(db *gorm.DB, validator *infrastructures.Validator) *VoucherService {
	return &VoucherService{
		db:        db,
		validator: validator,
	}
}

func (s *VoucherService) CreateVoucher(req *models.VoucherCreateRequest) (*models.Voucher, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	// Check if voucher code already exists
	var existingVoucher models.Voucher
	err := s.db.Where("code = ?", req.Code).First(&existingVoucher).Error
	if err == nil {
		return nil, errors.NewBadRequestError("Voucher code already exists")
	}

	// Create voucher
	voucher := &models.Voucher{
		Code:               req.Code,
		Name:               req.Name,
		Description:        req.Description,
		Type:               req.Type,
		Value:              req.Value,
		Currency:           req.Currency,
		LoyaltyPointsValue: req.LoyaltyPointsValue,
		DiscountPercentage: req.DiscountPercentage,
		DiscountAmount:     req.DiscountAmount,
		MaxRedeemCount:     req.MaxRedeemCount,
		CurrentRedeemCount: 0,
		ValidFrom:          req.ValidFrom,
		ValidUntil:         req.ValidUntil,
		Status:             models.VoucherStatusActive,
	}

	// Parse created by UUID if provided
	if req.CreatedBy != nil {
		createdBy, err := uuid.Parse(*req.CreatedBy)
		if err != nil {
			return nil, errors.NewBadRequestError("Invalid created by ID format")
		}
		voucher.CreatedBy = &createdBy
	}

	if err := s.db.Create(voucher).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to create voucher")
	}

	return voucher, nil
}

func (s *VoucherService) GetVoucher(voucherId string) (*models.Voucher, error) {
	voucherUUID, err := uuid.Parse(voucherId)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid voucher ID format")
	}

	var voucher models.Voucher
	err = s.db.Where("id = ?", voucherUUID).First(&voucher).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Voucher not found")
		}
		return nil, errors.NewInternalServerError(err, "Failed to get voucher")
	}

	return &voucher, nil
}

func (s *VoucherService) GetVoucherByCode(code string) (*models.Voucher, error) {
	var voucher models.Voucher
	err := s.db.Where("code = ?", code).First(&voucher).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Voucher not found")
		}
		return nil, errors.NewInternalServerError(err, "Failed to get voucher")
	}

	return &voucher, nil
}

func (s *VoucherService) GetVouchers(pagination *models.PaginationRequest, status *models.VoucherStatus) (*models.Pagination[[]models.Voucher], error) {
	// Set defaults
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}
	if pagination.Page <= 0 {
		pagination.Page = 1
	}

	offset := (pagination.Page - 1) * pagination.Limit

	// Build query for counting
	countQuery := s.db.Model(&models.Voucher{})
	if status != nil {
		countQuery = countQuery.Where("status = ?", *status)
	}

	// Count total items
	var totalItems int64
	if err := countQuery.Count(&totalItems).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to count vouchers")
	}

	var vouchers []models.Voucher
	query := s.db.Order("created_at DESC")

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if pagination.Limit > 0 {
		query = query.Limit(pagination.Limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&vouchers).Error
	if err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to get vouchers")
	}

	// Calculate pagination metadata
	totalPages := int((totalItems + int64(pagination.Limit) - 1) / int64(pagination.Limit))
	hasNext := pagination.Page < totalPages
	hasPrev := pagination.Page > 1

	result := &models.Pagination[[]models.Voucher]{
		Page:       pagination.Page,
		Limit:      pagination.Limit,
		TotalPages: totalPages,
		TotalItems: int(totalItems),
		HasNext:    hasNext,
		HasPrev:    hasPrev,
		Items:      vouchers,
	}

	return result, nil
}

func (s *VoucherService) UpdateVoucher(voucherId string, req *models.VoucherUpdateRequest) (*models.Voucher, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	voucher, err := s.GetVoucher(voucherId)
	if err != nil {
		return nil, err
	}

	// Check if new code already exists (if code is being updated)
	if req.Code != nil && *req.Code != voucher.Code {
		var existingVoucher models.Voucher
		err := s.db.Where("code = ? AND id != ?", *req.Code, voucher.ID).First(&existingVoucher).Error
		if err == nil {
			return nil, errors.NewBadRequestError("Voucher code already exists")
		}
	}

	// Update fields if provided
	if req.Code != nil {
		voucher.Code = *req.Code
	}
	if req.Name != nil {
		voucher.Name = *req.Name
	}
	if req.Description != nil {
		voucher.Description = req.Description
	}
	if req.Type != nil {
		voucher.Type = *req.Type
	}
	if req.Value != nil {
		voucher.Value = *req.Value
	}
	if req.Currency != nil {
		voucher.Currency = *req.Currency
	}
	if req.LoyaltyPointsValue != nil {
		voucher.LoyaltyPointsValue = req.LoyaltyPointsValue
	}
	if req.DiscountPercentage != nil {
		voucher.DiscountPercentage = req.DiscountPercentage
	}
	if req.DiscountAmount != nil {
		voucher.DiscountAmount = req.DiscountAmount
	}
	if req.MaxRedeemCount != nil {
		voucher.MaxRedeemCount = *req.MaxRedeemCount
	}
	if req.ValidFrom != nil {
		voucher.ValidFrom = *req.ValidFrom
	}
	if req.ValidUntil != nil {
		voucher.ValidUntil = req.ValidUntil
	}
	if req.Status != nil {
		voucher.Status = *req.Status
	}

	if err := s.db.Save(voucher).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to update voucher")
	}

	return voucher, nil
}

func (s *VoucherService) DeleteVoucher(voucherId string) error {
	voucher, err := s.GetVoucher(voucherId)
	if err != nil {
		return err
	}

	if err := s.db.Delete(voucher).Error; err != nil {
		return errors.NewInternalServerError(err, "Failed to delete voucher")
	}

	return nil
}

func (s *VoucherService) ValidateVoucher(code string) (*models.Voucher, error) {
	voucher, err := s.GetVoucherByCode(code)
	if err != nil {
		return nil, err
	}

	// Check if voucher is active
	if voucher.Status != models.VoucherStatusActive {
		return nil, errors.NewBadRequestError("Voucher is not active")
	}

	// Check if voucher is within valid period
	now := time.Now()
	if now.Before(voucher.ValidFrom) {
		return nil, errors.NewBadRequestError("Voucher is not yet valid")
	}

	if voucher.ValidUntil != nil && now.After(*voucher.ValidUntil) {
		return nil, errors.NewBadRequestError("Voucher has expired")
	}

	// Check if voucher has remaining redemptions
	if voucher.CurrentRedeemCount >= voucher.MaxRedeemCount {
		return nil, errors.NewBadRequestError("Voucher has reached maximum redemption limit")
	}

	return voucher, nil
}

func (s *VoucherService) IncrementRedeemCount(voucherId string) error {
	voucher, err := s.GetVoucher(voucherId)
	if err != nil {
		return err
	}

	voucher.CurrentRedeemCount++

	// Update status if max redemptions reached
	if voucher.CurrentRedeemCount >= voucher.MaxRedeemCount {
		voucher.Status = models.VoucherStatusRedeemed
	}

	if err := s.db.Save(voucher).Error; err != nil {
		return errors.NewInternalServerError(err, "Failed to update voucher redeem count")
	}

	return nil
}

func (s *VoucherService) UpdateExpiredVouchers() error {
	now := time.Now()

	// Update expired vouchers
	err := s.db.Model(&models.Voucher{}).
		Where("status = ? AND valid_until < ?", models.VoucherStatusActive, now).
		Update("status", models.VoucherStatusExpired).Error

	if err != nil {
		return errors.NewInternalServerError(err, "Failed to update expired vouchers")
	}

	return nil
}
