package infrastructures

import (
	"github.com/go-playground/validator/v10"
	"github.com/safatanc/gsalt-core/internal/app/errors"
)

type Validator struct {
	validate *validator.Validate
}

func NewValidator() *Validator {
	return &Validator{
		validate: validator.New(),
	}
}

func (v *Validator) Validate(i interface{}) error {
	if i == nil {
		return errors.NewBadRequestError("Invalid request body")
	}

	err := v.validate.Struct(i)
	if err != nil {
		return errors.NewBadRequestError(err.Error())
	}
	return nil
}
