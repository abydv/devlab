package api

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/abydv/devlab/internal/service"
	"github.com/abydv/devlab/internal/template"
	"github.com/abydv/devlab/internal/workspace"
)

// writeError maps a domain error to an HTTP response. This is
// transport-level translation, not business logic: the API layer
// still holds no decision-making of its own — it only decides which
// status code represents an error Engine already produced.
func writeError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, workspace.ErrNotFound),
		errors.Is(err, template.ErrNotFound),
		errors.Is(err, service.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(errorResponse{Error: err.Error()})

	case errors.Is(err, workspace.ErrNameExists),
		errors.Is(err, template.ErrNameExists):
		return c.Status(fiber.StatusConflict).JSON(errorResponse{Error: err.Error()})

	case errors.Is(err, workspace.ErrNameRequired),
		errors.Is(err, template.ErrNameRequired),
		errors.Is(err, template.ErrServicesRequired),
		errors.Is(err, template.ErrUnknownService):
		return c.Status(fiber.StatusBadRequest).JSON(errorResponse{Error: err.Error()})

	default:
		return c.Status(fiber.StatusInternalServerError).JSON(errorResponse{Error: err.Error()})
	}
}

type errorResponse struct {
	Error string `json:"error"`
}
