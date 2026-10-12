package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	http "github.com/aarock1234/fphttp"
)

// CookieExtractor extracts cookies from an HTTP response. It is called
// after each round trip, including intermediate redirects, and the returned
// cookies are stored in the client's jar. An error fails the request.
type CookieExtractor func(resp *http.Response) ([]*http.Cookie, error)

// DefaultCookieExtractor parses the response's Set-Cookie headers with
// ParseSetCookie. Like a browser, it skips headers it cannot parse.
func DefaultCookieExtractor(resp *http.Response) ([]*http.Cookie, error) {
	var cookies []*http.Cookie

	for _, line := range resp.Header.Values("Set-Cookie") {
		cookie, err := ParseSetCookie(line)
		if err != nil {
			continue
		}

		cookies = append(cookies, cookie)
	}

	return cookies, nil
}

// ParseCookies parses a Cookie header value ("name=value; name2=value2") or
// a JSON array of cookie objects. Header pairs without a name or "=" are
// skipped.
func ParseCookies(s string) ([]*http.Cookie, error) {
	s = strings.TrimSpace(s)

	if strings.HasPrefix(s, "[") {
		var cookies []*http.Cookie
		if err := json.Unmarshal([]byte(s), &cookies); err != nil {
			return nil, fmt.Errorf("decode json cookies: %w", err)
		}

		return cookies, nil
	}

	var cookies []*http.Cookie

	for pair := range strings.SplitSeq(s, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(pair), "=")
		if !ok || name == "" {
			continue
		}

		cookies = append(cookies, &http.Cookie{
			Name:  name,
			Value: value,
		})
	}

	return cookies, nil
}

// ParseSetCookie parses a Set-Cookie header line. Unlike net/http, it
// accepts double quotes inside cookie values. Unknown or malformed
// attributes are ignored, as browsers do.
func ParseSetCookie(line string) (*http.Cookie, error) {
	pair, attributes, _ := strings.Cut(line, ";")

	name, value, ok := strings.Cut(strings.TrimSpace(pair), "=")
	if !ok {
		return nil, errors.New("missing = in cookie name-value pair")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("empty cookie name")
	}

	cookie := &http.Cookie{
		Name: name,
		Raw:  line,
	}

	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
		cookie.Quoted = true
	}

	for i := range len(value) {
		if !validCookieValueByte(value[i]) {
			return nil, fmt.Errorf("invalid byte %#x in cookie value", value[i])
		}
	}

	cookie.Value = value

	for attribute := range strings.SplitSeq(attributes, ";") {
		applyCookieAttribute(cookie, attribute)
	}

	return cookie, nil
}

// applyCookieAttribute applies one Set-Cookie attribute to cookie.
func applyCookieAttribute(cookie *http.Cookie, attribute string) {
	name, value, _ := strings.Cut(attribute, "=")
	name = strings.ToLower(strings.TrimSpace(name))
	value = strings.TrimSpace(value)

	switch name {
	case "secure":
		cookie.Secure = true
	case "httponly":
		cookie.HttpOnly = true
	case "partitioned":
		cookie.Partitioned = true
	case "domain":
		cookie.Domain = value
	case "path":
		cookie.Path = value
	case "expires":
		expires, err := parseCookieTime(value)
		if err != nil {
			return
		}

		cookie.Expires = expires.UTC()
		cookie.RawExpires = value
	case "max-age":
		seconds, err := strconv.Atoi(value)
		if err != nil || seconds != 0 && value[0] == '0' {
			return
		}

		// net/http represents "delete now" as a negative MaxAge, since
		// zero means the attribute is absent.
		if seconds <= 0 {
			seconds = -1
		}

		cookie.MaxAge = seconds
	case "samesite":
		switch strings.ToLower(value) {
		case "lax":
			cookie.SameSite = http.SameSiteLaxMode
		case "strict":
			cookie.SameSite = http.SameSiteStrictMode
		case "none":
			cookie.SameSite = http.SameSiteNoneMode
		default:
			cookie.SameSite = http.SameSiteDefaultMode
		}
	}
}

// parseCookieTime parses an Expires attribute in the HTTP date formats or
// the dashed Netscape format.
func parseCookieTime(value string) (time.Time, error) {
	if t, err := http.ParseTime(value); err == nil {
		return t, nil
	}

	return time.Parse("Mon, 02-Jan-2006 15:04:05 MST", value)
}

// cookieHeader formats cookies as a Cookie header value, restoring the
// quotes of quoted values.
func cookieHeader(cookies []*http.Cookie) string {
	var b strings.Builder

	for i, cookie := range cookies {
		if i > 0 {
			b.WriteString("; ")
		}

		b.WriteString(cookie.Name)
		b.WriteByte('=')

		if cookie.Quoted {
			b.WriteByte('"')
			b.WriteString(cookie.Value)
			b.WriteByte('"')

			continue
		}

		b.WriteString(cookie.Value)
	}

	return b.String()
}

// validCookieValueByte reports whether b is a valid byte in a cookie value.
// This is a relaxed version of RFC 6265 that allows double quotes.
func validCookieValueByte(b byte) bool {
	return 0x20 <= b && b < 0x7f && b != ';' && b != '\\'
}
