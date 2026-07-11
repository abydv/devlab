// Package api implements DevLab's REST API: a thin Fiber HTTP surface
// over internal/engine. It holds no business logic — every handler
// validates its input, delegates to the Engine, and translates the
// result (or error) into an HTTP response.
package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/abydv/devlab/internal/engine"
)

// Server holds the dependencies every API handler needs.
type Server struct {
	engine *engine.Engine
}

// New returns a Fiber app exposing e over HTTP.
func New(e *engine.Engine) *fiber.App {
	s := &Server{engine: e}

	app := fiber.New(fiber.Config{
		AppName: "devlab",
	})
	app.Use(recover.New())

	app.Get("/healthz", s.health)

	v1 := app.Group("/api")

	v1.Get("/templates", s.listTemplates)
	v1.Get("/templates/:name", s.getTemplate)

	v1.Post("/workspaces", s.createWorkspace)
	v1.Get("/workspaces", s.listWorkspaces)
	v1.Get("/workspaces/:id", s.getWorkspace)
	v1.Delete("/workspaces/:id", s.deleteWorkspace)
	v1.Post("/workspaces/:id/start", s.startWorkspace)
	v1.Post("/workspaces/:id/stop", s.stopWorkspace)
	v1.Post("/workspaces/:id/reset", s.resetWorkspace)
	v1.Get("/workspaces/:id/status", s.workspaceStatus)
	v1.Get("/workspaces/:id/logs", s.workspaceLogs)

	return app
}

func (s *Server) health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}
