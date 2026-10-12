package client

import (
	"fmt"
	"strconv"
	"strings"

	http "github.com/aarock1234/fphttp"
)

// defaultVersion returns the browser version matching the fphttp profile
// that http.Profile selects for browser and platform, so the User-Agent
// agrees with the TLS and HTTP/2 fingerprint.
func defaultVersion(browser http.Browser, platform http.Platform) string {
	switch {
	case browser == http.BrowserFirefox:
		return "144.0"
	case browser == http.BrowserSafari:
		return "27.0"
	case browser == http.BrowserChrome && platform == http.PlatformMac:
		return "154.0.0.0"
	default:
		return "148.0.0.0"
	}
}

// identityHeaders returns the headers that identify the browser on every
// request: the User-Agent, the accepted content encodings, and, for Chromium
// browsers, client hints. Every advertised encoding is one the client can
// decompress.
func identityHeaders(browser http.Browser, platform http.Platform, version string) (http.Header, error) {
	switch browser {
	case http.BrowserChrome, http.BrowserEdge, http.BrowserBrave:
		return chromiumHeaders(browser, platform, version)
	case http.BrowserFirefox:
		return firefoxHeaders(platform, version)
	case http.BrowserSafari:
		return safariHeaders(platform, version)
	default:
		return nil, fmt.Errorf("unsupported browser %q", browser)
	}
}

func chromiumHeaders(browser http.Browser, platform http.Platform, version string) (http.Header, error) {
	var system, hintPlatform string

	switch platform {
	case http.PlatformWindows:
		system, hintPlatform = "Windows NT 10.0; Win64; x64", "Windows"
	case http.PlatformMac:
		system, hintPlatform = "Macintosh; Intel Mac OS X 10_15_7", "macOS"
	case http.PlatformLinux:
		system, hintPlatform = "X11; Linux x86_64", "Linux"
	case http.PlatformAndroid:
		system, hintPlatform = "Linux; Android 10; K", "Android"
	default:
		return nil, fmt.Errorf("unsupported platform %q for %s", platform, browser)
	}

	mobile := platform == http.PlatformAndroid

	product := "Safari/537.36"
	if mobile {
		product = "Mobile " + product
	}

	userAgent := fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s %s", system, version, product)

	brand := "Google Chrome"

	switch browser {
	case http.BrowserEdge:
		brand = "Microsoft Edge"

		if mobile {
			userAgent += " EdgA/" + version
		} else {
			userAgent += " Edg/" + version
		}
	case http.BrowserBrave:
		brand = "Brave"
	}

	major, _, _ := strings.Cut(version, ".")

	mobileHint := "?0"
	if mobile {
		mobileHint = "?1"
	}

	headers := http.Header{}
	headers.Set("User-Agent", userAgent)
	headers.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	headers.Set("Sec-Ch-Ua", fmt.Sprintf(`"Chromium";v=%q, %q;v=%q, "Not/A)Brand";v="8"`, major, brand, major))
	headers.Set("Sec-Ch-Ua-Mobile", mobileHint)
	headers.Set("Sec-Ch-Ua-Platform", strconv.Quote(hintPlatform))

	return headers, nil
}

func firefoxHeaders(platform http.Platform, version string) (http.Header, error) {
	var system string

	switch platform {
	case http.PlatformWindows:
		system = "Windows NT 10.0; Win64; x64"
	case http.PlatformMac:
		system = "Macintosh; Intel Mac OS X 10.15"
	case http.PlatformLinux:
		system = "X11; Linux x86_64"
	default:
		return nil, fmt.Errorf("unsupported platform %q for firefox", platform)
	}

	headers := http.Header{}
	headers.Set("User-Agent", fmt.Sprintf("Mozilla/5.0 (%s; rv:%s) Gecko/20100101 Firefox/%s", system, version, version))
	headers.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	return headers, nil
}

// safariHeaders models Safari's frozen operating system versions. iPadOS
// Safari requests desktop sites, so it reports the macOS User-Agent.
func safariHeaders(platform http.Platform, version string) (http.Header, error) {
	var userAgent string

	switch platform {
	case http.PlatformMac, http.PlatformIPadOS:
		userAgent = fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/%s Safari/605.1.15", version)
	case http.PlatformIOS:
		userAgent = fmt.Sprintf("Mozilla/5.0 (iPhone; CPU iPhone OS 18_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/%s Mobile/15E148 Safari/604.1", version)
	default:
		return nil, fmt.Errorf("unsupported platform %q for safari", platform)
	}

	headers := http.Header{}
	headers.Set("User-Agent", userAgent)
	headers.Set("Accept-Encoding", "gzip, deflate, br")

	return headers, nil
}

// mergeHeaders returns a copy of base with each override applied. An
// override with an empty value removes the header.
func mergeHeaders(base, overrides http.Header) http.Header {
	merged := base.Clone()

	for key, values := range overrides {
		merged.Del(key)

		if len(values) == 0 || values[0] == "" {
			continue
		}

		merged[http.CanonicalHeaderKey(key)] = values
	}

	return merged
}
