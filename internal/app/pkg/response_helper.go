package pkg

import (
	"errors"
	"reflect"

	"github.com/gofiber/fiber/v2"
	appError "github.com/safatanc/gsalt-core/internal/app/errors"
	"github.com/safatanc/gsalt-core/internal/app/models"
	"github.com/sirupsen/logrus"
)

func SuccessResponse[T any](c *fiber.Ctx, data T) error {
	return c.JSON(models.WebResponse[T]{
		Success: true,
		Data:    data,
	})
}

func ErrorResponse(c *fiber.Ctx, err error) error {
	var appErr *appError.AppError
	if errors.As(err, &appErr) {
		return c.Status(appErr.StatusCode).JSON(models.WebResponse[any]{
			Success: false,
			Message: appErr.Message,
		})
	}

	logrus.Errorf("[%s] %s", reflect.TypeOf(err).String(), err)

	return c.Status(fiber.StatusInternalServerError).JSON(models.WebResponse[any]{
		Success: false,
		Message: "Internal Server Error",
	})
}
