// Package web serves Ladder's local reading desk and its same-origin API.
package web

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"mime"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"ladder/internal/localstate"
	"ladder/internal/service"
	"ladder/internal/startup"

	"github.com/gofiber/fiber/v2"
	nethtml "golang.org/x/net/html"
)

//go:embed assets/*
var assets embed.FS

// Options binds the reader to its server-local history and optional URL prefix.
// StartupStatus is injectable so tests never query the host service manager.
type Options struct {
	Store         *localstate.Store
	BasePath      string
	StartupStatus func() (startup.State, error)
	TrustedHosts  []string
}

const maxJSONBytes = 16 << 10

// ReaderStyle is the only inline style permitted by the reading desk's CSP.
// Its digest lets the isolated, script-free article keep its own typography.
const ReaderStyle = `:root{color-scheme:light}html{background:#faf8f2;color:#21251f}body{margin:0;padding:28px 36px 64px;font:20px/1.8 Georgia,"Times New Roman",serif;overflow-wrap:anywhere}article{max-width:70ch;margin:auto}h1,h2,h3,h4{line-height:1.25;margin:1.5em 0 .65em}h1{font-size:1.8em}h2{font-size:1.4em}p,ul,ol,blockquote,pre{margin:0 0 1.2em}a{color:#4d634b;text-decoration-thickness:1px;text-underline-offset:3px}blockquote{border-left:3px solid #a5af9b;padding-left:1em;margin-left:0;color:#555e50}pre{white-space:pre-wrap;font:15px/1.65 ui-monospace,SFMono-Regular,Menlo,monospace;background:#efede5;padding:16px}code{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:.8em}table{border-collapse:collapse;display:block;overflow-x:auto}td,th{border:1px solid #d5d6c9;padding:8px}hr{border:0;border-top:1px solid #d5d6c9;margin:2em 0}@media(max-width:600px){body{padding:20px 20px 48px;font-size:18px}}`

// Register installs all trusted app routes before the legacy proxy catchall.
// router must already be grouped under Options.BasePath, as in the server.
func Register(router fiber.Router, svc *service.Service, options Options) {
	base := normalizeBase(options.BasePath)
	trusted := map[string]bool{"localhost": true, "127.0.0.1": true, "::1": true}
	for _, host := range options.TrustedHosts {
		if host = strings.ToLower(strings.TrimSpace(host)); host != "" && !strings.Contains(host, "*") {
			trusted[host] = true
		}
	}
	if options.StartupStatus == nil {
		options.StartupStatus = func() (startup.State, error) { return startup.Status(startup.Options{}) }
	}
	page := template.Must(template.New("reader").Parse(mustAsset("assets/index.html")))
	digest := sha256.Sum256([]byte(ReaderStyle))
	csp := "default-src 'none'; script-src 'self'; style-src 'self' 'sha256-" + base64.StdEncoding.EncodeToString(digest[:]) + "'; img-src 'self' data:; connect-src 'self'; frame-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'; object-src 'none'"
	secure := func(c *fiber.Ctx) {
		c.Set("Content-Security-Policy", csp)
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("Referrer-Policy", "no-referrer")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Cache-Control", "no-store")
	}
	servePage := func(c *fiber.Ctx) error {
		secure(c)
		var out bytes.Buffer
		if err := page.Execute(&out, struct{ BasePath string }{base}); err != nil {
			return apiError(c, 500, "page_error", "The reading desk could not be opened.")
		}
		c.Type("html", "utf-8")
		return c.Send(out.Bytes())
	}
	router.Get("/", servePage)
	router.Get("/reader", servePage)
	for _, name := range []string{"app.js", "style.css"} {
		name := name
		router.Get("/reader-assets/"+name, func(c *fiber.Ctx) error {
			secure(c)
			if strings.HasSuffix(name, ".js") {
				c.Type("js", "utf-8")
				style, _ := json.Marshal(ReaderStyle)
				hash, _ := json.Marshal(base64.StdEncoding.EncodeToString(digest[:]))
				return c.SendString("const READER_STYLE = " + string(style) + ";\nconst READER_STYLE_HASH = " + string(hash) + ";\n" + mustAsset("assets/"+name))
			} else {
				c.Type("css", "utf-8")
			}
			return c.SendString(mustAsset("assets/" + name))
		})
	}
	api := router.Group("/api/v1", func(c *fiber.Ctx) error {
		secure(c)
		host, valid := requestHostname(string(c.Request().Header.Host()))
		if !valid || !trusted[host] {
			return apiError(c, 403, "host_denied", "This hostname is not trusted by Ladder. Add the server hostname to TRUSTED_HOSTS and restart Ladder.")
		}
		if c.Get("X-Ladder-Client") != "reader" {
			return apiError(c, 403, "reader_required", "Open the Ladder reading desk to use this endpoint.")
		}
		origin := c.Get("Origin")
		// Use the actual Host, never X-Forwarded-Host, for the origin boundary.
		expected := c.Protocol() + "://" + string(c.Request().Header.Host())
		if (origin != "" && origin != expected) || c.Get("Sec-Fetch-Site") == "cross-site" {
			return apiError(c, 403, "origin_denied", "This endpoint only accepts requests from this Ladder origin.")
		}
		return c.Next()
	})
	api.Post("/read", func(c *fiber.Ctx) error {
		var request service.Request
		if !decode(c, &request) {
			return nil
		}
		if svc == nil {
			return apiError(c, 503, "reader_unavailable", "The reading service is unavailable.")
		}
		ctx, cancel := context.WithTimeout(c.UserContext(), 45*time.Second)
		defer cancel()
		doc, err := svc.Read(ctx, request)
		if err != nil {
			return serviceError(c, err)
		}
		doc.Content.HTML = externalLinks(doc.Content.HTML)
		response := struct {
			service.Document
			HistoryError string `json:"historyError,omitempty"`
		}{Document: doc}
		if options.Store == nil {
			response.HistoryError = "Local history is unavailable; this read was not saved."
		} else if err := options.Store.Record(doc.Metadata.SourceURL, doc.Title, time.Now()); err != nil {
			response.HistoryError = "Local history could not be saved: " + err.Error()
		}
		return c.JSON(response)
	})
	api.Get("/history", func(c *fiber.Ctx) error {
		if options.Store == nil {
			return apiError(c, 503, "history_unavailable", "Local history is unavailable on this server.")
		}
		return history(c, options.Store)
	})
	api.Delete("/history/:id", func(c *fiber.Ctx) error {
		if options.Store == nil {
			return apiError(c, 503, "history_unavailable", "Local history is unavailable on this server.")
		}
		id := c.Params("id")
		if len(id) != 24 || strings.IndexFunc(id, func(r rune) bool { return !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') }) >= 0 {
			return apiError(c, 400, "invalid_history_id", "History entry ID must contain 24 lowercase hexadecimal characters.")
		}
		if err := options.Store.Delete(id); err != nil {
			return stateError(c, err)
		}
		return history(c, options.Store)
	})
	api.Delete("/history", func(c *fiber.Ctx) error {
		if options.Store == nil {
			return apiError(c, 503, "history_unavailable", "Local history is unavailable on this server.")
		}
		if err := options.Store.Clear(); err != nil {
			return stateError(c, err)
		}
		return history(c, options.Store)
	})
	api.Get("/settings", func(c *fiber.Ctx) error {
		state, err := options.StartupStatus()
		response := struct {
			HistoryAvailable bool          `json:"historyAvailable"`
			HistoryEnabled   bool          `json:"historyEnabled"`
			HistoryLimit     int           `json:"historyLimit"`
			HistoryError     string        `json:"historyError,omitempty"`
			Startup          startup.State `json:"startup"`
			StartupError     string        `json:"startupError,omitempty"`
		}{HistoryAvailable: options.Store != nil, HistoryLimit: 100, Startup: state}
		if options.Store != nil {
			snapshot, stateErr := options.Store.SnapshotChecked()
			if stateErr != nil {
				response.HistoryAvailable = false
				response.HistoryError = "Local history could not be read: " + stateErr.Error()
			} else {
				response.HistoryEnabled = snapshot.HistoryEnabled
			}
		}
		if err != nil {
			response.StartupError = err.Error()
		}
		return c.JSON(response)
	})
	api.Patch("/settings", func(c *fiber.Ctx) error {
		var request struct {
			HistoryEnabled *bool `json:"historyEnabled"`
		}
		if !decode(c, &request) {
			return nil
		}
		if request.HistoryEnabled == nil {
			return apiError(c, 400, "invalid_settings", "Provide historyEnabled as true or false.")
		}
		if options.Store == nil {
			return apiError(c, 503, "history_unavailable", "Local history is unavailable on this server.")
		}
		if err := options.Store.SetEnabled(*request.HistoryEnabled); err != nil {
			return stateError(c, err)
		}
		return history(c, options.Store)
	})
	api.Get("/rules", func(c *fiber.Ctx) error {
		if svc == nil {
			return apiError(c, 503, "reader_unavailable", "The reading service is unavailable.")
		}
		return c.JSON(fiber.Map{"packs": svc.Packs()})
	})
	api.Post("/rules/test", func(c *fiber.Ctx) error {
		var request service.Request
		if !decode(c, &request) {
			return nil
		}
		if svc == nil {
			return apiError(c, 503, "reader_unavailable", "The reading service is unavailable.")
		}
		ctx, cancel := context.WithTimeout(c.UserContext(), 45*time.Second)
		defer cancel()
		result, err := svc.TestURL(ctx, request)
		if err != nil {
			return serviceError(c, err)
		}
		return c.JSON(result)
	})
	api.Delete("/cache", func(c *fiber.Ctx) error {
		if svc == nil {
			return apiError(c, 503, "reader_unavailable", "The reading service is unavailable.")
		}
		svc.ClearCache()
		return c.JSON(fiber.Map{"cleared": true})
	})
	api.All("/*", func(c *fiber.Ctx) error {
		return apiError(c, 404, "endpoint_not_found", "This reader endpoint or method does not exist.")
	})
}

// externalLinks operates on the already sanitized article in an inert Go tree.
// Only the isolated frame receives this markup, never the app's DOM.
func externalLinks(fragment string) string {
	doc, err := nethtml.Parse(strings.NewReader(fragment))
	if err != nil {
		return ""
	}
	var body *nethtml.Node
	var walk func(*nethtml.Node)
	walk = func(node *nethtml.Node) {
		if node.Type == nethtml.ElementNode && node.Data == "body" {
			body = node
		}
		if node.Type == nethtml.ElementNode && node.Data == "a" {
			var href string
			attrs := node.Attr[:0]
			for _, attr := range node.Attr {
				if attr.Key == "href" {
					href = attr.Val
				}
				if attr.Key != "target" && attr.Key != "rel" {
					attrs = append(attrs, attr)
				}
			}
			node.Attr = attrs
			parsed, err := url.Parse(href)
			if err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" && parsed.User == nil {
				node.Attr = append(node.Attr, nethtml.Attribute{Key: "target", Val: "_blank"}, nethtml.Attribute{Key: "rel", Val: "noopener noreferrer"})
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	if body == nil {
		return ""
	}
	var out bytes.Buffer
	for child := body.FirstChild; child != nil; child = child.NextSibling {
		_ = nethtml.Render(&out, child)
	}
	return out.String()
}

func requestHostname(authority string) (string, bool) {
	host := authority
	if strings.Contains(authority, ":") {
		parsed, port, err := net.SplitHostPort(authority)
		if err == nil {
			n, err := strconv.Atoi(port)
			if err != nil || n < 1 || n > 65535 {
				return "", false
			}
			host = parsed
		} else if strings.HasPrefix(authority, "[") && strings.HasSuffix(authority, "]") && net.ParseIP(authority[1:len(authority)-1]) != nil {
			host = authority[1 : len(authority)-1]
		} else {
			return "", false
		}
	}
	if host == "" || strings.ContainsAny(host, "/\\@ \t\r\n") {
		return "", false
	}
	return strings.ToLower(host), true
}

func normalizeBase(base string) string {
	base = strings.Trim(strings.TrimSpace(base), "/")
	if base == "" {
		return ""
	}
	return "/" + base
}

func mustAsset(name string) string {
	b, err := assets.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func decode(c *fiber.Ctx, target any) bool {
	reject := func(status int, code, message string) bool { _ = apiError(c, status, code, message); return false }
	media, _, err := mime.ParseMediaType(c.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return reject(415, "json_required", "Send an application/json request body.")
	}
	if len(c.Body()) > maxJSONBytes {
		return reject(413, "body_too_large", "Reader requests must be 16 KiB or smaller.")
	}
	body := bytes.TrimSpace(c.Body())
	if len(body) == 0 || body[0] != '{' {
		return reject(400, "invalid_json", "Send a JSON object with the requested fields.")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return reject(400, "invalid_json", "The JSON body contains invalid or unknown fields.")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return reject(400, "invalid_json", "Send exactly one JSON object.")
	}
	return true
}

func history(c *fiber.Ctx, store *localstate.Store) error {
	snapshot, err := store.SnapshotChecked()
	if err != nil {
		return apiError(c, 500, "history_read_failed", "Local history could not be read: "+err.Error())
	}
	return c.JSON(snapshot)
}

func apiError(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(fiber.Map{"error": service.Problem{Code: code, Message: message}})
}

func serviceError(c *fiber.Ctx, err error) error {
	var problem *service.Error
	if errors.As(err, &problem) {
		status := problem.Status
		if status < 400 || status > 599 {
			status = 500
		}
		return apiError(c, status, problem.Code, problem.Message)
	}
	return apiError(c, 500, "reader_error", "The document could not be read. Try again or open the original URL.")
}

func stateError(c *fiber.Ctx, err error) error {
	return apiError(c, 500, "history_write_failed", "Local history could not be updated: "+err.Error())
}
