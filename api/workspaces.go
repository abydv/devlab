package api

import (
	"github.com/gofiber/fiber/v2"

	"github.com/abydv/devlab/internal/workspace"
)

type createWorkspaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Template    string `json:"template"`
}

type statusResponse struct {
	Status workspace.Status `json:"status"`
}

func (s *Server) createWorkspace(c *fiber.Ctx) error {
	var req createWorkspaceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errorResponse{Error: "invalid request body"})
	}

	ws, err := s.engine.CreateWorkspace(req.Name, req.Description, req.Template)
	if err != nil {
		return writeError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(ws)
}

func (s *Server) listWorkspaces(c *fiber.Ctx) error {
	list, err := s.engine.ListWorkspaces()
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(list)
}

func (s *Server) getWorkspace(c *fiber.Ctx) error {
	ws, err := s.engine.GetWorkspace(c.Params("id"))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(ws)
}

func (s *Server) deleteWorkspace(c *fiber.Ctx) error {
	if err := s.engine.DeleteWorkspace(c.Context(), c.Params("id")); err != nil {
		return writeError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) startWorkspace(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := s.engine.StartWorkspace(c.Context(), id); err != nil {
		return writeError(c, err)
	}
	ws, err := s.engine.GetWorkspace(id)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(ws)
}

func (s *Server) stopWorkspace(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := s.engine.StopWorkspace(c.Context(), id); err != nil {
		return writeError(c, err)
	}
	ws, err := s.engine.GetWorkspace(id)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(ws)
}

func (s *Server) resetWorkspace(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := s.engine.ResetWorkspace(c.Context(), id); err != nil {
		return writeError(c, err)
	}
	ws, err := s.engine.GetWorkspace(id)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(ws)
}

func (s *Server) workspaceStatus(c *fiber.Ctx) error {
	status, err := s.engine.WorkspaceStatus(c.Context(), c.Params("id"))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(statusResponse{Status: status})
}

func (s *Server) workspaceLogs(c *fiber.Ctx) error {
	logs, err := s.engine.WorkspaceLogs(c.Context(), c.Params("id"))
	if err != nil {
		return writeError(c, err)
	}
	c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
	return c.SendString(logs)
}
