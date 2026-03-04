package client

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

// Browser identifies the browser for TLS and HTTP/2 fingerprinting.
type Browser string

const (
	// BrowserChrome identifies Google Chrome.
	BrowserChrome Browser = "chrome"
	// BrowserEdge identifies Microsoft Edge.
	BrowserEdge Browser = "edge"
	// BrowserBrave identifies the Brave browser.
	BrowserBrave Browser = "brave"
	// BrowserSafari identifies Apple Safari.
	BrowserSafari Browser = "safari"
	// BrowserFirefox identifies Mozilla Firefox.
	BrowserFirefox Browser = "firefox"
)

// Platform identifies the OS platform for TLS and HTTP/2 fingerprinting.
type Platform string

const (
	// PlatformWindows targets the Windows operating system.
	PlatformWindows Platform = "windows"
	// PlatformMac targets macOS.
	PlatformMac Platform = "mac"
	// PlatformLinux targets Linux.
	PlatformLinux Platform = "linux"
	// PlatformIOS targets iOS.
	PlatformIOS Platform = "ios"
	// PlatformIPadOS targets iPadOS.
	PlatformIPadOS Platform = "ipados"
)

// H2Profile contains HTTP/2 connection settings that control the fingerprint
// of the HTTP/2 SETTINGS frame and WINDOW_UPDATE.
type H2Profile struct {
	Settings         []http2.Setting
	ConnectionWindow uint32
	Priority         http2.PriorityParam
	PseudoOrder      []string
}

// BrowserProfile combines a TLS ClientHelloID with HTTP/2 settings and
// default request headers for a specific browser identity.
type BrowserProfile struct {
	ClientHelloID  utls.ClientHelloID
	H2             H2Profile
	DefaultHeaders http.Header
}

const defaultBrowserVersion = "131.0.0.0"

const settingEnableConnectProtocol = http2.SettingID(0x8)

var h2Chrome = H2Profile{
	Settings: []http2.Setting{
		{ID: http2.SettingHeaderTableSize, Val: 65536},
		{ID: http2.SettingEnablePush, Val: 0},
		{ID: http2.SettingInitialWindowSize, Val: 6291456},
		{ID: http2.SettingMaxHeaderListSize, Val: 262144},
	},
	ConnectionWindow: 15663105,
	Priority: http2.PriorityParam{
		Weight: 255,
	},
	PseudoOrder: []string{":method", ":authority", ":scheme", ":path"},
}

var h2Safari = H2Profile{
	Settings: []http2.Setting{
		{ID: http2.SettingHeaderTableSize, Val: 4096},
		{ID: http2.SettingEnablePush, Val: 0},
		{ID: http2.SettingInitialWindowSize, Val: 2097152},
		{ID: http2.SettingMaxConcurrentStreams, Val: 100},
		{ID: settingEnableConnectProtocol, Val: 1},
	},
	ConnectionWindow: 10485760,
	Priority: http2.PriorityParam{
		Weight: 254,
	},
	PseudoOrder: []string{":method", ":scheme", ":path", ":authority"},
}

var h2Firefox = H2Profile{
	Settings: []http2.Setting{
		{ID: http2.SettingHeaderTableSize, Val: 65536},
		{ID: http2.SettingInitialWindowSize, Val: 131072},
		{ID: http2.SettingMaxFrameSize, Val: 16384},
	},
	ConnectionWindow: 12517377,
	Priority: http2.PriorityParam{
		StreamDep: 13,
		Weight:    41,
	},
	PseudoOrder: []string{":method", ":path", ":authority", ":scheme"},
}

// resolveProfile returns a BrowserProfile for the given browser, version,
// and platform combination.
func resolveProfile(browser Browser, version string, platform Platform) BrowserProfile {
	switch browser {
	case BrowserSafari:
		return safariProfile(version, platform)
	case BrowserFirefox:
		return firefoxProfile(version, platform)
	default:
		return chromiumProfile(browser, version, platform)
	}
}

func chromiumProfile(browser Browser, version string, platform Platform) BrowserProfile {
	major := parseMajor(version)

	var ua string
	switch platform {
	case PlatformMac:
		ua = fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", version)
	case PlatformLinux:
		ua = fmt.Sprintf("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", version)
	default:
		ua = fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", version)
	}

	if browser == BrowserEdge {
		ua += fmt.Sprintf(" Edg/%s", version)
	}

	var brandName string
	switch browser {
	case BrowserEdge:
		brandName = "Microsoft Edge"
	case BrowserBrave:
		brandName = "Brave"
	default:
		brandName = "Google Chrome"
	}

	secChUA := fmt.Sprintf(`"Chromium";v="%d", "%s";v="%d", "Not/A)Brand";v="8"`, major, brandName, major)

	var secChUAPlatform string
	switch platform {
	case PlatformMac:
		secChUAPlatform = `"macOS"`
	case PlatformLinux:
		secChUAPlatform = `"Linux"`
	default:
		secChUAPlatform = `"Windows"`
	}

	headers := http.Header{}
	headers.Set("user-agent", ua)
	headers.Set("sec-ch-ua", secChUA)
	headers.Set("sec-ch-ua-mobile", "?0")
	headers.Set("sec-ch-ua-platform", secChUAPlatform)

	return BrowserProfile{
		ClientHelloID:  utls.HelloChrome_Auto,
		H2:             h2Chrome,
		DefaultHeaders: headers,
	}
}

func safariProfile(version string, platform Platform) BrowserProfile {
	var (
		ua            string
		clientHelloID utls.ClientHelloID
	)

	switch platform {
	case PlatformIOS, PlatformIPadOS:
		major := parseMajor(version)
		minor := parseMinor(version)
		ua = fmt.Sprintf("Mozilla/5.0 (iPhone; CPU iPhone OS %d_%d like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/%s Mobile/15E148 Safari/604.1", major, minor, version)
		clientHelloID = utls.HelloIOS_Auto
	default:
		ua = fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/%s Safari/605.1.15", version)
		clientHelloID = utls.HelloSafari_Auto
	}

	headers := http.Header{}
	headers.Set("user-agent", ua)

	return BrowserProfile{
		ClientHelloID:  clientHelloID,
		H2:             h2Safari,
		DefaultHeaders: headers,
	}
}

func firefoxProfile(version string, platform Platform) BrowserProfile {
	var ua string
	switch platform {
	case PlatformMac:
		ua = fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:%s) Gecko/20100101 Firefox/%s", version, version)
	case PlatformLinux:
		ua = fmt.Sprintf("Mozilla/5.0 (X11; Linux x86_64; rv:%s) Gecko/20100101 Firefox/%s", version, version)
	default:
		ua = fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:%s) Gecko/20100101 Firefox/%s", version, version)
	}

	headers := http.Header{}
	headers.Set("user-agent", ua)

	return BrowserProfile{
		ClientHelloID:  utls.HelloFirefox_Auto,
		H2:             h2Firefox,
		DefaultHeaders: headers,
	}
}

// parseMajor returns the major version number from a dotted version string.
func parseMajor(version string) int {
	parts := strings.SplitN(version, ".", 2)
	n, _ := strconv.Atoi(parts[0])

	return n
}

// parseMinor returns the minor version number from a dotted version string.
func parseMinor(version string) int {
	parts := strings.SplitN(version, ".", 3)
	if len(parts) < 2 {
		return 0
	}

	n, _ := strconv.Atoi(parts[1])

	return n
}
