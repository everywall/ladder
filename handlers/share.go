package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
)

// computeShareToken returns the first 32 hex chars of HMAC-SHA256(secret, targetURL).
// 32 hex chars = 128 bits of security, keeps URLs concise.
func computeShareToken(secret, targetURL string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(targetURL))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

// ShareLink handles GET /share?url=<target>
// Requires Basic Auth. Returns a JSON shareable link for the target URL that
// can be used without credentials.
func ShareLink(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if secret == "" {
			return c.Status(fiber.StatusServiceUnavailable).SendString("share feature is disabled (SHARE_SECRET not set)")
		}

		targetURL := c.Query("url")
		if targetURL == "" {
			return c.Status(fiber.StatusBadRequest).SendString("missing 'url' query parameter")
		}

		token := computeShareToken(secret, targetURL)
		shareURL := fmt.Sprintf("%s://%s/share/%s/%s", c.Protocol(), c.Hostname(), token, targetURL)

		return c.JSON(fiber.Map{
			"url": shareURL,
		})
	}
}

// ProxySiteViaShare replaces the bare /* proxy route when SHARE_SECRET is set.
// It redirects every request to its /share/:token/* equivalent so that all
// proxied URLs are in the share-link format.
func ProxySiteViaShare(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if secret == "" {
			return c.Status(fiber.StatusServiceUnavailable).SendString("share feature is disabled (SHARE_SECRET not set)")
		}

		targetURL, err := extractUrl(c)
		if err != nil || targetURL == "" {
			log.Println("ERROR in share redirect URL extraction:", err)
			return c.Status(fiber.StatusBadRequest).SendString("invalid URL")
		}

		token := computeShareToken(secret, targetURL)
		return c.Redirect("/share/"+token+"/"+targetURL, fiber.StatusFound)
	}
}

// ShareProxy handles GET /share/:token/*
// Validates the HMAC token against the target URL and proxies the content if valid.
// No Basic Auth is required — the token itself acts as the credential.
func ShareProxy(secret, rulesetPath string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if secret == "" {
			return c.Status(fiber.StatusNotFound).SendString("not found")
		}

		providedToken := c.Params("token")

		targetURL, err := extractUrl(c)
		if err != nil {
			log.Println("ERROR in share URL extraction:", err)
			return c.Status(fiber.StatusBadRequest).SendString("invalid URL")
		}

		if targetURL == "" {
			return c.Status(fiber.StatusBadRequest).SendString("missing target URL")
		}

		expectedToken := computeShareToken(secret, targetURL)

		// Constant-time comparison to prevent timing attacks
		if !hmac.Equal([]byte(providedToken), []byte(expectedToken)) {
			return c.Status(fiber.StatusForbidden).SendString("invalid share token")
		}

		queries := c.Queries()
		body, _, resp, err := fetchSite(targetURL, queries)
		if err != nil {
			log.Println("ERROR in share proxy:", err)
			c.SendStatus(fiber.StatusInternalServerError)
			return c.SendString(err.Error())
		}

		c.Set("Content-Type", resp.Header.Get("Content-Type"))
		c.Set("Content-Security-Policy", resp.Header.Get("Content-Security-Policy"))

		return c.SendString(body)
	}
}
