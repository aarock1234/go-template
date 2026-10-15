package client

import (
	"encoding/json"
	"fmt"
	"strings"

	http "github.com/aarock1234/fphttp/v2"
)

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
