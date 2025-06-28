package services

import (
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"gorm.io/gorm"
)

type AccountService struct {
	db             *gorm.DB
	validator      *validator.Validate
	connectService *ConnectService
}

func NewAccountService(db *gorm.DB, validator *validator.Validate, connectService *ConnectService) *AccountService {
	return &AccountService{
		db:             db,
		validator:      validator,
		connectService: connectService,
	}
}

func (s *AccountService) CreateAccount(accessToken string) (*models.Account, error) {
	connectUser, err := s.connectService.GetCurrentUser(accessToken)
	if err != nil {
		return nil, err
	}

	if connectUser == nil {
		return nil, errors.NewBadRequestError("Connect user not found")
	}

	// Check if account already exists
	var existingAccount models.Account
	err = s.db.Where("connect_id = ?", connectUser.ID).First(&existingAccount).Error
	if existingAccount.ConnectID != uuid.Nil {
		return nil, errors.NewBadRequestError("Account already exists")
	}

	// Create Account
	account := &models.Account{
		ConnectID: connectUser.ID,
		Balance:   0,
		Points:    0,
	}

	if err := s.db.Create(account).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to create account")
	}

	return account, nil
}

func (s *AccountService) GetAccount(connectId string) (*models.Account, error) {
	connectIdUUID, err := uuid.Parse(connectId)
	if err != nil {
		return nil, errors.NewBadRequestError("Invalid connect ID format")
	}

	var account models.Account
	err = s.db.Where("connect_id = ?", connectIdUUID).First(&account).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Account not found")
		}
		return nil, errors.NewInternalServerError(err, "Failed to get account")
	}

	return &account, nil
}

func (s *AccountService) GetAccountByToken(accessToken string) (*models.Account, error) {
	connectUser, err := s.connectService.GetCurrentUser(accessToken)
	if err != nil {
		return nil, err
	}

	if connectUser == nil {
		return nil, errors.NewBadRequestError("Connect user not found")
	}

	return s.GetAccount(connectUser.ID.String())
}

func (s *AccountService) UpdateAccount(connectId string, req *models.AccountUpdateDto) (*models.Account, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, errors.NewBadRequestError(err.Error())
	}

	account, err := s.GetAccount(connectId)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Balance != nil {
		account.Balance = *req.Balance
	}
	if req.Points != nil {
		account.Points = *req.Points
	}

	if err := s.db.Save(account).Error; err != nil {
		return nil, errors.NewInternalServerError(err, "Failed to update account")
	}

	return account, nil
}

func (s *AccountService) DeleteAccount(connectId string) error {
	account, err := s.GetAccount(connectId)
	if err != nil {
		return err
	}

	if err := s.db.Delete(account).Error; err != nil {
		return errors.NewInternalServerError(err, "Failed to delete account")
	}

	return nil
}
