package api

import "github.com/gofiber/fiber/v2"

func (s *Server) listTemplates(c *fiber.Ctx) error {
	return c.JSON(s.engine.ListTemplates())
}

func (s *Server) getTemplate(c *fiber.Ctx) error {
	tmpl, err := s.engine.GetTemplate(c.Params("name"))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(tmpl)
}
