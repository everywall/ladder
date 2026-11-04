package handlers

import (
	"github.com/gofiber/fiber/v2"
	"ladder/internal/service"
)

func Raw(c *fiber.Ctx) error {
	target, err := routeTarget(c, false)
	if err != nil {
		return legacyError(c, &service.Error{Code: "invalid_url", Status: 400, Message: err.Error()})
	}
	s, err := configured()
	if err != nil {
		return legacyError(c, &service.Error{Code: "configuration", Status: 500, Message: "invalid transport configuration"})
	}
	f, err := s.Fetch(c.UserContext(), service.Request{URL: target})
	if err != nil {
		return legacyError(c, err)
	}
	sandbox(c)
	sendHeaders(c, f.Header)
	return c.Status(f.Metadata.UpstreamStatus).Send(f.Body)
}
