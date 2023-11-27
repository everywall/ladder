package rewriters

import (
	"bytes"
	"io"

	"golang.org/x/net/html"
)

// IHTMLTokenRewriter defines an interface for modifying HTML tokens.
type IHTMLTokenRewriter interface {
	// ShouldModify determines whether a given HTML token requires modification.
	ShouldModify(*html.Token) bool

	// ModifyToken applies modifications to a given HTML token.
	// It returns strings representing content to be prepended and
	// appended to the token. If no modifications are required or if an error occurs,
	// it returns empty strings for both 'prepend' and 'append'.
	// Note: The original token is not modified if an error occurs.
	ModifyToken(*html.Token) (prepend, append string)
}

// HTMLRewriter is a struct that can take multiple TokenHandlers and process all
// HTML tokens from http.Response.Body in a single pass, making changes and returning a new io.ReadCloser
//
//   - HTMLRewriter reads the http.Response.Body stream,
//     parsing each HTML token one at a time and making modifications (defined by implementations of IHTMLTokenRewriter)
//
//   - When ProxyChain.Execute() is called, the response body will be read from the server
//     and pulled through each ResponseModification which wraps the ProxyChain.Response.Body
//     without ever buffering the entire HTTP response in memory.
type HTMLRewriter struct {
	tokenizer             *html.Tokenizer
	currentToken          *html.Token
	tokenBuffer           *bytes.Buffer
	currentTokenProcessed bool
	rewriters             []IHTMLTokenRewriter
}

// NewHTMLRewriter creates a new HTMLRewriter instance.
// It processes HTML tokens from an io.ReadCloser source (typically http.Response.Body)
// using a series of HTMLTokenRewriters. Each HTMLTokenRewriter in the 'rewriters' slice
// applies its specific modifications to the HTML tokens.
// The HTMLRewriter reads from the provided 'src', applies the modifications,
// and returns the processed content as a new io.ReadCloser.
// This new io.ReadCloser can be used to stream the modified content back to the client.
//
// Parameters:
//   - src: An io.ReadCloser representing the source of the HTML content, such as http.Response.Body.
//   - rewriters: A slice of HTMLTokenRewriters that define the modifications to be applied to the HTML tokens.
//
// Returns:
//   - A pointer to an HTMLRewriter, which implements io.ReadCloser, containing the modified HTML content.
func NewHTMLRewriter(src io.ReadCloser, rewriters ...IHTMLTokenRewriter) *HTMLRewriter {
	return &HTMLRewriter{
		tokenizer:             html.NewTokenizer(src),
		currentToken:          nil,
		tokenBuffer:           new(bytes.Buffer),
		currentTokenProcessed: false,
		rewriters:             rewriters,
	}
}

// Close resets the internal state of HTMLRewriter, clearing buffers and token data.
func (r *HTMLRewriter) Close() error {
	r.tokenBuffer.Reset()
	r.currentToken = nil
	r.currentTokenProcessed = false
	return nil
}

// Read processes the HTML content, rewriting URLs and managing the state of tokens.
func (r *HTMLRewriter) Read(p []byte) (int, error) {

	if r.currentToken == nil || r.currentToken.Data == "" || r.currentTokenProcessed {
		tokenType := r.tokenizer.Next()

		// done reading html, close out reader
		if tokenType == html.ErrorToken {
			if r.tokenizer.Err() == io.EOF {
				return 0, io.EOF
			}
			return 0, r.tokenizer.Err()
		}

		// get the next token; reset buffer
		t := r.tokenizer.Token()
		r.currentToken = &t
		r.tokenBuffer.Reset()

		// buffer += "<prepends> <token> <appends>"
		// process token through all registered rewriters
		// rewriters will modify the token, and optionally
		// return a <prepend> or <append> string token
		appends := make([]string, 0, len(r.rewriters))
		for _, rewriter := range r.rewriters {
			if !rewriter.ShouldModify(r.currentToken) {
				continue
			}
			prepend, a := rewriter.ModifyToken(r.currentToken)
			appends = append(appends, a)
			// add <prepends> to buffer
			r.tokenBuffer.WriteString(prepend)
		}

		// add <token> to buffer
		if tokenType == html.TextToken {
			// don't unescape textTokens (such as inline scripts).
			// Token.String() by default will escape the inputs, but
			// we don't want to modify the original source
			r.tokenBuffer.WriteString(r.currentToken.Data)
		} else {
			r.tokenBuffer.WriteString(r.currentToken.String())
		}

		// add <appends> to buffer
		for _, a := range appends {
			r.tokenBuffer.WriteString(a)
		}

		r.currentTokenProcessed = false
	}

	n, err := r.tokenBuffer.Read(p)
	if err == io.EOF || r.tokenBuffer.Len() == 0 {
		r.currentTokenProcessed = true
		err = nil // EOF in this context is expected and not an actual error
	}
	return n, err
}
