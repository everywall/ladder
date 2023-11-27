package requestmodifers

import (
	"ladder/proxychain"

	"strconv"

	"crypto/sha256"

	//	"crypto/tls"
	"context"
	"errors"
	"fmt"
	"net"

	"strings"
	"sync"

	http "github.com/Danny-Dasilva/fhttp"
	"github.com/Danny-Dasilva/fhttp/http2"
	utls "github.com/Danny-Dasilva/utls"
	"golang.org/x/net/proxy"
)

// SpoofJA3fingerprint modifies the TLS client and user agent to spoof a particular JA3 fingerprint
// Some anti-bot WAFs such as cloudflare can fingerprint the fields of the TLS hello packet, and the order in which they appear
// https://web.archive.org/web/20231126224326/https://engineering.salesforce.com/tls-fingerprinting-with-ja3-and-ja3s-247362855967/
// https://web.archive.org/web/20231119065253/https://developers.cloudflare.com/bots/concepts/ja3-fingerprint/
func SpoofJA3fingerprint(ja3 string, userAgent string) proxychain.RequestModification {
	//fmt.Println(ja3)
	return func(chain *proxychain.ProxyChain) error {
		spoofConfig := browser{
			UserAgent:          userAgent,
			JA3:                ja3,
			InsecureSkipVerify: false,
		}

		// deep copy existing client while modifying http transport
		ja3SpoofClient := &http.Client{
			//Transport:     newJA3SpooferTransport(spoofConfig, proxy),
			Transport:     newJA3SpooferTransport(spoofConfig),
			Timeout:       chain.Client.Timeout,
			CheckRedirect: chain.Client.CheckRedirect,
		}

		chain.SetOnceHTTPClient(ja3SpoofClient)
		return nil
	}
}

// SpoofJA3fingerprintWithProxy modifies the TLS client and user agent to spoof a particular JA3 fingerprint and use a proxy.ContextDialer from the "golang.org/x/net/proxy"
// Some anti-bot WAFs such as cloudflare can fingerprint the fields of the TLS hello packet, and the order in which they appear
// https://web.archive.org/web/20231126224326/https://engineering.salesforce.com/tls-fingerprinting-with-ja3-and-ja3s-247362855967/
// https://web.archive.org/web/20231119065253/https://developers.cloudflare.com/bots/concepts/ja3-fingerprint/
func SpoofJA3fingerprintWithProxy(ja3 string, userAgent string, proxy proxy.ContextDialer) proxychain.RequestModification {
	return func(chain *proxychain.ProxyChain) error {
		spoofConfig := browser{
			UserAgent:          userAgent,
			JA3:                ja3,
			InsecureSkipVerify: false,
		}

		// deep copy existing client while modifying http transport
		ja3SpoofClient := &http.Client{
			//Transport:     newJA3SpooferTransport(spoofConfig, proxy),
			Transport:     newJA3SpooferTransport(spoofConfig, proxy),
			Timeout:       chain.Client.Timeout,
			CheckRedirect: chain.Client.CheckRedirect,
		}

		chain.SetOnceHTTPClient(ja3SpoofClient)
		return nil
	}
}

// ============================================================================================== //
// The following code is adapated from https://github.com/Danny-Dasilva/CycleTLS with GPL3 license
// ============================================================================================== //
var errProtocolNegotiated = errors.New("protocol negotiated")

type roundTripper struct {
	sync.Mutex
	// fix typing
	JA3       string
	UserAgent string

	InsecureSkipVerify bool
	cachedConnections  map[string]net.Conn
	cachedTransports   map[string]http.RoundTripper

	dialer proxy.ContextDialer
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", rt.UserAgent)
	addr := rt.getDialTLSAddr(req)
	if _, ok := rt.cachedTransports[addr]; !ok {
		if err := rt.getTransport(req, addr); err != nil {
			return nil, err
		}
	}
	return rt.cachedTransports[addr].RoundTrip(req)
}

func (rt *roundTripper) getTransport(req *http.Request, addr string) error {
	switch strings.ToLower(req.URL.Scheme) {
	case "http":
		rt.cachedTransports[addr] = &http.Transport{DialContext: rt.dialer.DialContext, DisableKeepAlives: true}
		return nil
	case "https":
	default:
		return fmt.Errorf("invalid URL scheme: [%v]", req.URL.Scheme)
	}

	_, err := rt.dialTLS(req.Context(), "tcp", addr)
	switch err {
	case errProtocolNegotiated:
	case nil:
		// Should never happen.
		panic("dialTLS returned no error when determining cachedTransports")
	default:
		return err
	}

	return nil
}

func (rt *roundTripper) dialTLS(ctx context.Context, network, addr string) (net.Conn, error) {
	rt.Lock()
	defer rt.Unlock()

	// If we have the connection from when we determined the HTTPS
	// cachedTransports to use, return that.
	if conn := rt.cachedConnections[addr]; conn != nil {
		return conn, nil
	}
	rawConn, err := rt.dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	var host string
	if host, _, err = net.SplitHostPort(addr); err != nil {
		host = addr
	}
	//////////////////

	spec, err := StringToSpec(rt.JA3, rt.UserAgent)
	if err != nil {
		return nil, err
	}

	conn := utls.UClient(rawConn, &utls.Config{ServerName: host, InsecureSkipVerify: rt.InsecureSkipVerify}, // MinVersion:         tls.VersionTLS10,
		// MaxVersion:         tls.VersionTLS13,

		utls.HelloCustom)

	if err := conn.ApplyPreset(spec); err != nil {
		return nil, err
	}

	if err = conn.Handshake(); err != nil {
		_ = conn.Close()

		if err.Error() == "tls: CurvePreferences includes unsupported curve" {
			//fix this
			return nil, fmt.Errorf("conn.Handshake() error for tls 1.3 (please retry request): %+v", err)
		}
		return nil, fmt.Errorf("uTlsConn.Handshake() error: %+v", err)
	}

	//////////
	if rt.cachedTransports[addr] != nil {
		return conn, nil
	}

	// No http.Transport constructed yet, create one based on the results
	// of ALPN.
	switch conn.ConnectionState().NegotiatedProtocol {
	case http2.NextProtoTLS:
		parsedUserAgent := parseUserAgent(rt.UserAgent).UserAgent
		t2 := http2.Transport{DialTLS: rt.dialTLSHTTP2,
			PushHandler: &http2.DefaultPushHandler{},
			Navigator:   parsedUserAgent,
		}
		rt.cachedTransports[addr] = &t2
	default:
		// Assume the remote peer is speaking HTTP 1.x + TLS.
		rt.cachedTransports[addr] = &http.Transport{DialTLSContext: rt.dialTLS}

	}

	// Stash the connection just established for use servicing the
	// actual request (should be near-immediate).
	rt.cachedConnections[addr] = conn

	return nil, errProtocolNegotiated
}

func (rt *roundTripper) dialTLSHTTP2(network, addr string, _ *utls.Config) (net.Conn, error) {
	return rt.dialTLS(context.Background(), network, addr)
}

func (rt *roundTripper) getDialTLSAddr(req *http.Request) string {
	host, port, err := net.SplitHostPort(req.URL.Host)
	if err == nil {
		return net.JoinHostPort(host, port)
	}
	return net.JoinHostPort(req.URL.Host, "443") // we can assume port is 443 at this point
}

func (rt *roundTripper) CloseIdleConnections() {
	for addr, conn := range rt.cachedConnections {
		_ = conn.Close()
		delete(rt.cachedConnections, addr)
	}
}

func newJA3SpooferTransport(browser browser, dialer ...proxy.ContextDialer) http.RoundTripper {
	if dialer != nil && len(dialer) > 0 {
		return &roundTripper{
			dialer:             dialer[0],
			JA3:                browser.JA3,
			UserAgent:          browser.UserAgent,
			cachedTransports:   make(map[string]http.RoundTripper),
			cachedConnections:  make(map[string]net.Conn),
			InsecureSkipVerify: browser.InsecureSkipVerify,
		}
	}

	return &roundTripper{
		dialer: proxy.Direct,

		JA3:                browser.JA3,
		UserAgent:          browser.UserAgent,
		cachedTransports:   make(map[string]http.RoundTripper),
		cachedConnections:  make(map[string]net.Conn),
		InsecureSkipVerify: browser.InsecureSkipVerify,
	}
}

type browser struct {
	// Return a greeting that embeds the name in a message.
	JA3                string
	UserAgent          string
	InsecureSkipVerify bool
}

// adapted from
// https://github.com/Danny-Dasilva/CycleTLS/blob/963daacd92414dc84a0ec6930dd252381cb1247a/cycletls/utils.go#L27
const (
	chrome  = "chrome"  //chrome User agent enum
	firefox = "firefox" //firefox User agent enum
)

type UserAgent struct {
	UserAgent   string
	HeaderOrder []string
}

// ParseUserAgent returns the pseudo header order and user agent string for chrome/firefox
func parseUserAgent(userAgent string) UserAgent {
	switch {
	case strings.Contains(strings.ToLower(userAgent), "chrome"):
		return UserAgent{chrome, []string{":method", ":authority", ":scheme", ":path"}}
	case strings.Contains(strings.ToLower(userAgent), "firefox"):
		return UserAgent{firefox, []string{":method", ":path", ":authority", ":scheme"}}
	default:
		return UserAgent{chrome, []string{":method", ":authority", ":scheme", ":path"}}
	}

}

// ==============

// StringToSpec creates a ClientHelloSpec based on a JA3 string
func StringToSpec(ja3 string, userAgent string) (*utls.ClientHelloSpec, error) {
	parsedUserAgent := parseUserAgent(userAgent).UserAgent
	extMap := genMap()
	tokens := strings.Split(ja3, ",")

	version := tokens[0]
	ciphers := strings.Split(tokens[1], "-")
	extensions := strings.Split(tokens[2], "-")
	curves := strings.Split(tokens[3], "-")
	if len(curves) == 1 && curves[0] == "" {
		curves = []string{}
	}
	pointFormats := strings.Split(tokens[4], "-")
	if len(pointFormats) == 1 && pointFormats[0] == "" {
		pointFormats = []string{}
	}
	// parse curves
	var targetCurves []utls.CurveID
	targetCurves = append(targetCurves, utls.CurveID(utls.GREASE_PLACEHOLDER)) //append grease for Chrome browsers
	for _, c := range curves {
		cid, err := strconv.ParseUint(c, 10, 16)
		if err != nil {
			return nil, err
		}
		targetCurves = append(targetCurves, utls.CurveID(cid))
		// if cid != uint64(utls.CurveP521) {
		// CurveP521 sometimes causes handshake errors
		// }
	}
	extMap["10"] = &utls.SupportedCurvesExtension{Curves: targetCurves}

	// parse point formats
	var targetPointFormats []byte
	for _, p := range pointFormats {
		pid, err := strconv.ParseUint(p, 10, 8)
		if err != nil {
			return nil, err
		}
		targetPointFormats = append(targetPointFormats, byte(pid))
	}
	extMap["11"] = &utls.SupportedPointsExtension{SupportedPoints: targetPointFormats}

	// set extension 43
	vid64, err := strconv.ParseUint(version, 10, 16)
	if err != nil {
		return nil, err
	}
	vid := uint16(vid64)
	// extMap["43"] = &utls.SupportedVersionsExtension{
	// 	Versions: []uint16{
	// 		utls.VersionTLS12,
	// 	},
	// }

	// build extenions list
	var exts []utls.TLSExtension
	//Optionally Add Chrome Grease Extension
	if parsedUserAgent == chrome {
		exts = append(exts, &utls.UtlsGREASEExtension{})
	}
	for _, e := range extensions {
		te, ok := extMap[e]
		if !ok {
			return nil, errors.New("string2spec extension error")
		}
		// //Optionally add Chrome Grease Extension
		if e == "21" && parsedUserAgent == chrome {
			exts = append(exts, &utls.UtlsGREASEExtension{})
		}
		exts = append(exts, te)
	}
	//Add this back in if user agent is chrome and no padding extension is given
	// if parsedUserAgent == chrome {
	// 	exts = append(exts, &utls.UtlsGREASEExtension{})
	// 	exts = append(exts, &utls.UtlsPaddingExtension{GetPaddingLen: utls.BoringPaddingStyle})
	// }
	// build SSLVersion
	// vid64, err := strconv.ParseUint(version, 10, 16)
	// if err != nil {
	// 	return nil, err
	// }

	// build CipherSuites
	var suites []uint16
	//Optionally Add Chrome Grease Extension
	if parsedUserAgent == chrome {
		suites = append(suites, utls.GREASE_PLACEHOLDER)
	}
	for _, c := range ciphers {
		cid, err := strconv.ParseUint(c, 10, 16)
		if err != nil {
			return nil, err
		}
		suites = append(suites, uint16(cid))
	}
	_ = vid
	return &utls.ClientHelloSpec{
		// TLSVersMin:         vid,
		// TLSVersMax:         vid,
		CipherSuites:       suites,
		CompressionMethods: []byte{0},
		Extensions:         exts,
		GetSessionID:       sha256.Sum256,
	}, nil
}

func genMap() (extMap map[string]utls.TLSExtension) {
	extMap = map[string]utls.TLSExtension{
		"0": &utls.SNIExtension{},
		"5": &utls.StatusRequestExtension{},
		// These are applied later
		// "10": &tls.SupportedCurvesExtension{...}
		// "11": &tls.SupportedPointsExtension{...}
		"13": &utls.SignatureAlgorithmsExtension{
			SupportedSignatureAlgorithms: []utls.SignatureScheme{
				utls.ECDSAWithP256AndSHA256,
				utls.ECDSAWithP384AndSHA384,
				utls.ECDSAWithP521AndSHA512,
				utls.PSSWithSHA256,
				utls.PSSWithSHA384,
				utls.PSSWithSHA512,
				utls.PKCS1WithSHA256,
				utls.PKCS1WithSHA384,
				utls.PKCS1WithSHA512,
				utls.ECDSAWithSHA1,
				utls.PKCS1WithSHA1,
			},
		},
		"16": &utls.ALPNExtension{
			AlpnProtocols: []string{"h2", "http/1.1"},
		},
		"17": &utls.GenericExtension{Id: 17}, // status_request_v2
		"18": &utls.SCTExtension{},
		"21": &utls.UtlsPaddingExtension{GetPaddingLen: utls.BoringPaddingStyle},
		"22": &utls.GenericExtension{Id: 22}, // encrypt_then_mac
		"23": &utls.UtlsExtendedMasterSecretExtension{},
		"27": &utls.CompressCertificateExtension{
			Algorithms: []utls.CertCompressionAlgo{utls.CertCompressionBrotli},
		},
		"28": &utls.FakeRecordSizeLimitExtension{}, //Limit: 0x4001
		"35": &utls.SessionTicketExtension{},
		"34": &utls.GenericExtension{Id: 34},
		"41": &utls.GenericExtension{Id: 41}, //FIXME pre_shared_key
		"43": &utls.SupportedVersionsExtension{Versions: []uint16{
			utls.GREASE_PLACEHOLDER,
			utls.VersionTLS13,
			utls.VersionTLS12,
			utls.VersionTLS11,
			utls.VersionTLS10}},
		"44": &utls.CookieExtension{},
		"45": &utls.PSKKeyExchangeModesExtension{Modes: []uint8{
			utls.PskModeDHE,
		}},
		"49": &utls.GenericExtension{Id: 49}, // post_handshake_auth
		"50": &utls.GenericExtension{Id: 50}, // signature_algorithms_cert
		"51": &utls.KeyShareExtension{KeyShares: []utls.KeyShare{
			{Group: utls.CurveID(utls.GREASE_PLACEHOLDER), Data: []byte{0}},
			{Group: utls.X25519},

			// {Group: utls.CurveP384}, known bug missing correct extensions for handshake
		}},
		"30032": &utls.GenericExtension{Id: 0x7550, Data: []byte{0}}, //FIXME
		"13172": &utls.NPNExtension{},
		"17513": &utls.ApplicationSettingsExtension{
			SupportedALPNList: []string{
				"h2",
			},
		},
		"65281": &utls.RenegotiationInfoExtension{
			Renegotiation: utls.RenegotiateOnceAsClient,
		},
		"65037": &utls.GenericExtension{Id: 65037},
	}
	return
}
