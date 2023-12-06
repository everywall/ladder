package proxychain

import (
	"errors"
	"fmt"
	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	profiles "github.com/bogdanfinn/tls-client/profiles"
	"io"
	"log"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
)

/*
ProxyChain manages the process of forwarding an HTTP request to an upstream server,
applying request and response modifications along the way.

  - It accepts incoming HTTP requests (as a Fiber *ctx), and applies
    request modifiers (ReqMods) and response modifiers (ResMods) before passing the
    upstream response back to the client.

  - ProxyChains can be reused to avoid memory allocations. However, they are not concurrent-safe
    so a ProxyChainPool should be used with mutexes to avoid memory errors.

---

# EXAMPLE

```

import (

	rx "ladder/pkg/proxychain/requestmodifiers"
	tx "ladder/pkg/proxychain/responsemodifiers"
	"ladder/pkg/proxychain/responsemodifiers/rewriters"
	"ladder/internal/proxychain"

)

proxychain.NewProxyChain().

	SetFiberCtx(c).
	SetRequestModifications(
		rx.BlockOutgoingCookies(),
		rx.SpoofOrigin(),
		rx.SpoofReferrer(),
	).
	SetResultModifications(
		tx.BlockIncomingCookies(),
		tx.RewriteHTMLResourceURLs()
	).
	Execute()

```

	client            ladder service            upstream

┌─────────┐    ┌────────────────────────┐    ┌─────────┐
│         │GET │                        │    │         │
│       req────┼───► ProxyChain         │    │         │
│         │    │       │                │    │         │
│         │    │       ▼                │    │         │
│         │    │     apply              │    │         │
│         │    │ RequestModifications   │    │         │
│         │    │       │                │    │         │
│         │    │       ▼                │    │         │
│         │    │     send        GET    │    │         │
│         │    │     Request req────────┼─►  │         │
│         │    │                        │    │         │
│         │    │                 200 OK │    │         │
│         │    │       ┌────────────────┼─response     │
│         │    │       ▼                │    │         │
│         │    │     apply              │    │         │
│         │    │ ResultModifications    │    │         │
│         │    │       │                │    │         │
│         │◄───┼───────┘                │    │         │
│         │    │ 200 OK                 │    │         │
│         │    │                        │    │         │
└─────────┘    └────────────────────────┘    └─────────┘
*/
type ProxyChain struct {
	Context                   *fiber.Ctx
	Client                    HTTPClient
	onceClient                HTTPClient
	Request                   *http.Request
	Response                  *http.Response
	requestModifications      []RequestModification
	onceRequestModifications  []RequestModification
	onceResponseModifications []ResponseModification
	responseModifications     []ResponseModification
	debugMode                 bool
	abortErr                  error
	APIPrefix                 string
}

// a ProxyStrategy is a pre-built proxychain with purpose-built defaults
type ProxyStrategy ProxyChain

// A RequestModification is a function that should operate on the
// ProxyChain Req or Client field, using the fiber ctx as needed.
type RequestModification func(*ProxyChain) error

// A ResponseModification is a function that should operate on the
// ProxyChain Res (http result) & Body (buffered http response body) field
type ResponseModification func(*ProxyChain) error

// abstraction over HTTPClient
type HTTPClient interface {
	GetCookies(u *url.URL) []*http.Cookie
	SetCookies(u *url.URL, cookies []*http.Cookie)
	SetCookieJar(jar http.CookieJar)
	GetCookieJar() http.CookieJar
	SetProxy(proxyURL string) error
	GetProxy() string
	SetFollowRedirect(followRedirect bool)
	GetFollowRedirect() bool
	CloseIdleConnections()
	Do(req *http.Request) (*http.Response, error)
	Get(url string) (resp *http.Response, err error)
	Head(url string) (resp *http.Response, err error)
	Post(url, contentType string, body io.Reader) (resp *http.Response, err error)
}

// SetRequestModifications sets the ProxyChain's request modifiers
// the modifier will not fire until ProxyChain.Execute() is run.
func (chain *ProxyChain) SetRequestModifications(mods ...RequestModification) *ProxyChain {
	chain.requestModifications = mods
	return chain
}

// AddRequestModifications adds more request modifiers to the ProxyChain
// the modifier will not fire until ProxyChain.Execute() is run.
func (chain *ProxyChain) AddRequestModifications(mods ...RequestModification) *ProxyChain {
	chain.requestModifications = append(chain.requestModifications, mods...)
	return chain
}

// AddOnceRequestModifications adds a request modifier to the ProxyChain that should only fire once
// the modifier will not fire until ProxyChain.Execute() is run and will be removed after it has been applied.
func (chain *ProxyChain) AddOnceRequestModifications(mods ...RequestModification) *ProxyChain {
	chain.onceRequestModifications = append(chain.onceRequestModifications, mods...)
	return chain
}

// AddOnceResponseModifications adds a response modifier to the ProxyChain that should only fire once
// the modifier will not fire until ProxyChain.Execute() is run and will be removed after it has been applied.
func (chain *ProxyChain) AddOnceResponseModifications(mods ...ResponseModification) *ProxyChain {
	chain.onceResponseModifications = append(chain.onceResponseModifications, mods...)
	return chain
}

// AddResponseModifications sets the ProxyChain's response modifiers
// the modifier will not fire until ProxyChain.Execute() is run.
func (chain *ProxyChain) AddResponseModifications(mods ...ResponseModification) *ProxyChain {
	chain.responseModifications = mods
	return chain
}

// WithAPIPath trims the path during URL extraction.
// example: using path = "api/outline/", a path like "http://localhost:8080/api/outline/https://example.com" becomes "https://example.com"
func (chain *ProxyChain) WithAPIPath(path string) *ProxyChain {
	chain.APIPrefix = path
	chain.APIPrefix = strings.TrimSuffix(chain.APIPrefix, "*")
	return chain
}

func (chain *ProxyChain) _initializeRequest() (*http.Request, error) {
	if chain.Context == nil {
		chain.abortErr = chain.abort(errors.New("no context set"))
		return nil, chain.abortErr
	}
	// initialize a request (without url)
	req, err := http.NewRequest(chain.Context.Method(), "", nil)
	if err != nil {
		return nil, err
	}
	chain.Request = req
	switch chain.Context.Method() {
	case "GET":
	case "DELETE":
	case "HEAD":
	case "OPTIONS":
		break
	case "POST":
	case "PUT":
	case "PATCH":
		// stream content of body from client request to upstream request
		chain.Request.Body = io.NopCloser(chain.Context.Request().BodyStream())
	default:
		return nil, fmt.Errorf("unsupported request method from client: '%s'", chain.Context.Method())
	}

	return req, nil
}

// reconstructURLFromReferer reconstructs the URL using the referer's scheme, host, and the relative path / queries
func reconstructURLFromReferer(referer *url.URL, relativeURL *url.URL) (*url.URL, error) {
	// Extract the real url from referer path
	realURL, err := url.Parse(strings.TrimPrefix(referer.Path, "/"))
	if err != nil {
		return nil, fmt.Errorf("error parsing real URL from referer '%s': %v", referer.Path, err)
	}

	if realURL.Scheme == "" || realURL.Host == "" {
		return nil, fmt.Errorf("invalid referer URL: '%s' on request '%s", referer.String(), relativeURL.String())
	}

	log.Printf("rewrite relative URL using referer: '%s' -> '%s'\n", relativeURL.String(), realURL.String())

	return &url.URL{
		Scheme:   referer.Scheme,
		Host:     referer.Host,
		Path:     realURL.Path,
		RawQuery: realURL.RawQuery,
	}, nil
}

// prevents calls like: http://localhost:8080/http://localhost:8080
func preventRecursiveProxyRequest(urlQuery *url.URL, baseProxyURL string) *url.URL {
	u := urlQuery.String()
	isRecursive := strings.HasPrefix(u, baseProxyURL) || u == baseProxyURL
	if !isRecursive {
		return urlQuery
	}

	fixedURL, err := url.Parse(strings.TrimPrefix(strings.TrimPrefix(urlQuery.String(), baseProxyURL), "/"))
	if err != nil {
		log.Printf("proxychain: failed to fix recursive request: '%s' -> '%s\n'", baseProxyURL, u)
		return urlQuery
	}
	return preventRecursiveProxyRequest(fixedURL, baseProxyURL)
}

// extractURL extracts a URL from the request ctx
func (chain *ProxyChain) extractURL() (*url.URL, error) {
	isLocal := strings.HasPrefix(chain.Context.BaseURL(), "http://localhost") || strings.HasPrefix(chain.Context.BaseURL(), "http://127.0.0.1")
	isReqPath := strings.HasPrefix(chain.Context.Path(), "/http")
	isAPI := strings.HasPrefix(chain.Context.Path(), "/api")
	isOutline := strings.HasPrefix(chain.Context.Path(), "/outline")

	if isLocal || isReqPath || isAPI || isOutline {
		return chain.extractURLFromPath()
	}

	u, err := url.Parse(chain.Context.BaseURL())
	if err != nil {
		return &url.URL{}, err
	}
	parts := strings.Split(u.Hostname(), ".")
	if len(parts) < 2 {
		fmt.Println("path")
		return chain.extractURLFromPath()
	}

	return chain.extractURLFromSubdomain()
}

// extractURLFromPath extracts a URL from the request ctx if subdomains are used.
func (chain *ProxyChain) extractURLFromSubdomain() (*url.URL, error) {
	u, err := url.Parse(chain.Context.BaseURL())
	if err != nil {
		return &url.URL{}, err
	}
	parts := strings.Split(u.Hostname(), ".")
	if len(parts) < 2 {
		// no subdomain set, fallback to path extraction
		//panic("asdf")
		return chain.extractURLFromPath()
	}
	subdomain := strings.Join(parts[:len(parts)-2], ".")
	subURL := subdomain
	subURL = strings.ReplaceAll(subURL, "--", "|")
	subURL = strings.ReplaceAll(subURL, "-", ".")
	subURL = strings.ReplaceAll(subURL, "|", "-")
	return url.Parse(fmt.Sprintf("https://%s/%s", subURL, u.Path))
}

// extractURLFromPath extracts a URL from the request ctx. If the URL in the request
// is a relative path, it reconstructs the full URL using the referer header.
func (chain *ProxyChain) extractURLFromPath() (*url.URL, error) {
	reqURL := chain.Context.Params("*")

	fmt.Println("XXXXXXXXXXXXXXXX")
	fmt.Println(reqURL)
	fmt.Println(chain.APIPrefix)

	reqURL = strings.TrimPrefix(reqURL, chain.APIPrefix)

	// sometimes client requests doubleroot '//'
	// there is a bug somewhere else, but this is a workaround until we find it
	if strings.HasPrefix(reqURL, "/") || strings.HasPrefix(reqURL, `%2F`) {
		reqURL = strings.TrimPrefix(reqURL, "/")
		reqURL = strings.TrimPrefix(reqURL, `%2F`)
	}

	// unescape url query
	uReqURL, err := url.QueryUnescape(reqURL)
	if err == nil {
		reqURL = uReqURL
	}

	urlQuery, err := url.Parse(reqURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing request URL '%s': %v", reqURL, err)
	}

	// prevent recursive proxy requests
	fullURL := chain.Context.Request().URI()
	proxyURL := fmt.Sprintf("%s://%s", fullURL.Scheme(), fullURL.Host())
	urlQuery = preventRecursiveProxyRequest(urlQuery, proxyURL)

	// Handle standard paths
	// eg: https://localhost:8080/https://realsite.com/images/foobar.jpg -> https://realsite.com/images/foobar.jpg
	isRelativePath := urlQuery.Scheme == ""
	if !isRelativePath {
		return urlQuery, nil
	}

	// Handle relative URLs
	// eg: https://localhost:8080/images/foobar.jpg -> https://realsite.com/images/foobar.jpg
	referer, err := url.Parse(chain.Context.Get("referer"))
	relativePath := urlQuery
	if err != nil {
		return nil, fmt.Errorf("error parsing referer URL from req: '%s': %v", relativePath, err)
	}
	return reconstructURLFromReferer(referer, relativePath)
}

// SetFiberCtx takes the request ctx from the client
// for the modifiers and execute function to use.
// it must be set everytime a new request comes through
// if the upstream request url cannot be extracted from the ctx,
// a 500 error will be sent back to the client
func (chain *ProxyChain) SetFiberCtx(ctx *fiber.Ctx) *ProxyChain {
	chain.Context = ctx

	// initialize the request and prepare it for modification
	req, err := chain._initializeRequest()
	if err != nil {
		chain.abortErr = chain.abort(err)
	}
	chain.Request = req

	// extract the URL for the request and add it to the new request
	url, err := chain.extractURL()
	if err != nil {
		chain.abortErr = chain.abort(err)
	} else {
		chain.Request.URL = url
		fmt.Printf("extracted URL: %s\n", chain.Request.URL)
	}

	return chain
}

func (chain *ProxyChain) validateCtxIsSet() error {
	if chain.Context != nil {
		return nil
	}
	err := errors.New("proxyChain was called without setting a fiber Ctx. Use ProxyChain.SetFiberCtx()")
	chain.abortErr = chain.abort(err)
	return chain.abortErr
}

// SetHTTPClient sets a new upstream http client transport
// useful for modifying TLS
func (chain *ProxyChain) SetHTTPClient(httpClient HTTPClient) *ProxyChain {
	chain.Client = httpClient
	return chain
}

// SetOnceHTTPClient sets a new upstream http client transport temporarily
// and clears it once it is used.
func (chain *ProxyChain) SetOnceHTTPClient(httpClient HTTPClient) *ProxyChain {
	chain.onceClient = httpClient
	return chain
}

// SetVerbose changes the logging behavior to print
// the modification steps and applied rulesets for debugging
func (chain *ProxyChain) SetDebugLogging(isDebugMode bool) *ProxyChain {
	if isDebugMode {
		log.Println("DEBUG MODE ENABLED")
	}
	chain.debugMode = isDebugMode
	return chain
}

// abort proxychain and return 500 error to client
// this will prevent Execute from firing and reset the state
// returns the initial error enriched with context
func (chain *ProxyChain) abort(err error) error {
	// defer chain._reset()
	chain.abortErr = err
	chain.Context.Response().SetStatusCode(500)
	var e error
	if chain.Request.URL != nil {
		e = fmt.Errorf("ProxyChain error for '%s': %s", chain.Request.URL.String(), err.Error())
	} else {
		e = fmt.Errorf("ProxyChain error: '%s'", err.Error())
	}
	chain.Context.SendString(e.Error())
	log.Println(e.Error())
	return e
}

// internal function to reset state of ProxyChain for reuse
func (chain *ProxyChain) _reset() {
	chain.abortErr = nil
	chain.Request = nil
	// chain.Response = nil
	chain.Context = nil
	chain.onceResponseModifications = []ResponseModification{}
	chain.onceRequestModifications = []RequestModification{}
	// chain.onceClient = nil
}

// NewProxyChain initializes a new ProxyChain
func NewProxyChain() *ProxyChain {
	chain := new(ProxyChain)

	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(20),
		//tls_client.WithRandomTLSExtensionOrder(),
		tls_client.WithClientProfile(profiles.Chrome_117),
		// tls_client.WithNotFollowRedirects(),
		// tls_client.WithCookieJar(jar), // create cookieJar instance and pass it as argument
	}
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		panic(err)
	}
	chain.Client = client

	return chain
}

/// ========================================================================================================

// _execute sends the request for the ProxyChain and returns the raw body only
// the caller is responsible for returning a response back to the requestor
// the caller is also responsible for calling chain._reset() when they are done with the body
func (chain *ProxyChain) _execute() (io.Reader, error) {
	// ================== PREFLIGHT CHECKS =============================
	if chain.validateCtxIsSet() != nil || chain.abortErr != nil {
		return nil, chain.abortErr
	}
	if chain.Request == nil {
		return nil, errors.New("proxychain request not yet initialized")
	}
	if chain.Request.URL.Scheme == "" {
		return nil, errors.New("request url not set or invalid. Check ProxyChain ReqMods for issues")
	}

	// ======== REQUEST MODIFICATIONS :: [client -> ladder] -> upstream -> ladder -> client =============================
	// Apply requestModifications to proxychain
	for _, applyRequestModificationsTo := range chain.requestModifications {
		err := applyRequestModificationsTo(chain)
		if err != nil {
			return nil, chain.abort(err)
		}
	}

	// Apply onceRequestModifications to proxychain and clear them
	for _, applyOnceRequestModificationsTo := range chain.onceRequestModifications {
		err := applyOnceRequestModificationsTo(chain)
		if err != nil {
			return nil, chain.abort(err)
		}
	}
	chain.onceRequestModifications = []RequestModification{}

	// ======== SEND REQUEST UPSTREAM :: client -> [ladder -> upstream] -> ladder -> client =============================
	// Send Request Upstream
	if chain.onceClient != nil {
		// if chain.SetOnceClient() is used, use that client instead of the
		// default http client temporarily.
		resp, err := chain.onceClient.Do(chain.Request)
		if err != nil {
			return nil, chain.abort(err)
		}
		chain.Response = resp
		// chain.onceClient = nil
	} else {
		resp, err := chain.Client.Do(chain.Request)
		if err != nil {
			return nil, chain.abort(err)
		}
		chain.Response = resp
	}

	// ======== APPLY RESPONSE MODIFIERS :: client -> ladder -> [upstream -> ladder] -> client =============================
	// Apply ResponseModifiers to proxychain
	for _, applyResultModificationsTo := range chain.responseModifications {
		err := applyResultModificationsTo(chain)
		if err != nil {
			return nil, chain.abort(err)
		}
	}

	// Apply onceResponseModifications to proxychain and clear them
	for _, applyOnceResponseModificationsTo := range chain.onceResponseModifications {
		err := applyOnceResponseModificationsTo(chain)
		if err != nil {
			return nil, chain.abort(err)
		}
	}
	chain.onceResponseModifications = []ResponseModification{}

	// ======== RETURN BODY TO CLIENT :: client -> ladder -> upstream -> [ladder -> client] =============================
	return chain.Response.Body, nil
}

// Execute sends the request for the ProxyChain and returns the request to the sender
// and resets the fields so that the ProxyChain can be reused.
// if any step in the ProxyChain fails, the request will abort and a 500 error will
// be returned to the client
func (chain *ProxyChain) Execute() error {
	defer chain._reset()
	body, err := chain._execute()
	if err != nil {
		log.Println(err)
		return err
	}
	if chain.Context == nil {
		return errors.New("no context set")
	}

	// in case api user did not set or forward content-type, we do it for them
	if chain.Context.Get("content-type") == "" {
		chain.Context.Set("content-type", chain.Response.Header.Get("content-type"))
	}

	// Return request back to client
	return chain.Context.SendStream(body)

	// return chain.Context.SendStream(body)
}
