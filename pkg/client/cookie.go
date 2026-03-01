package client

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	http "github.com/saucesteals/fhttp"
)

// SetCookies stores cookies for the given URL in the client's jar.
func (c *Client) SetCookies(url *url.URL, cookies []*http.Cookie) {
	if c.jar == nil {
		return
	}

	c.jar.SetCookies(url, cookies)
}

// SetCookieString sets cookies with a specific domain for subdomain sharing.
// Domain should include leading dot (e.g., ".uber.com").
// Optional exclude parameter allows filtering out specific cookie names.
func (c *Client) SetCookieString(domain string, cookieString string, exclude ...string) error {
	if c.jar == nil {
		return nil
	}

	if cookieString == "" {
		return nil
	}

	slog.Debug("setting cookies for domain", "domain", domain, "cookieStr", cookieString)

	excludeSet := make(map[string]struct{}, len(exclude))
	for _, name := range exclude {
		excludeSet[name] = struct{}{}
	}

	cookies := ParseCookies(cookieString)

	for _, cookie := range cookies {
		slog.Debug("cookie", "name", cookie.Name, "value", cookie.Value)
	}

	var filtered []*http.Cookie
	for _, cookie := range cookies {
		if _, ok := excludeSet[cookie.Name]; ok {
			continue
		}

		existingCookie, ok := c.FindCookie(cookie.Name, domain)
		if ok && existingCookie.Value == cookie.Value {
			continue
		}

		cookie.Domain = strings.TrimPrefix(domain, "www")
		filtered = append(filtered, cookie)
	}

	c.jar.SetCookies(&url.URL{
		Scheme: "https",
		Host:   domain,
	}, filtered)

	return nil
}

// FindCookie returns the first cookie matching name for the given domain.
func (c *Client) FindCookie(name string, domain string) (*http.Cookie, bool) {
	if c.jar == nil {
		return nil, false
	}

	for _, cookie := range c.GetCookies(domain) {
		if strings.EqualFold(cookie.Name, name) {
			return cookie, true
		}
	}

	return nil, false
}

// GetCookies returns cookies for the given domain.
func (c *Client) GetCookies(domain string) []*http.Cookie {
	if strings.HasPrefix(domain, ".") {
		domain = "www" + domain
	}

	return c.jar.Cookies(&url.URL{
		Scheme: "https",
		Host:   domain,
	})
}

// GetCookieString returns cookies for the given domain as a semicolon-separated string.
func (c *Client) GetCookieString(domain string) string {
	if c.jar == nil {
		return ""
	}

	cookies := c.GetCookies(domain)
	cookieStrings := make([]string, len(cookies))
	for i, cookie := range cookies {
		cookieStrings[i] = cookie.Name + "=" + cookie.Value
	}

	return strings.Join(cookieStrings, "; ")
}

// ClearCookies clears all cookies by creating a new jar.
func (c *Client) ClearCookies() error {
	if c.jar == nil {
		return nil
	}

	jar, err := newCookieJar()
	if err != nil {
		return fmt.Errorf("creating cookie jar: %w", err)
	}

	c.jar = jar

	return nil
}

// ParseCookies parses a raw cookie string (name=value; name2=value2) into http.Cookie slice.
func ParseCookies(cookieStr string) []*http.Cookie {
	cookieStr = strings.TrimSpace(cookieStr)

	if strings.HasPrefix(cookieStr, "[") {
		cookies, err := parseJSONCookies(cookieStr)
		if err != nil {
			return nil
		}
		return cookies
	}

	var cookies []*http.Cookie
	for cookie := range strings.SplitSeq(cookieStr, ";") {
		cookie = strings.TrimSpace(cookie)
		if cookie == "" {
			continue
		}

		parts := strings.SplitN(cookie, "=", 2)
		if len(parts) != 2 {
			continue
		}

		cookies = append(cookies, &http.Cookie{
			Name:  parts[0],
			Value: parts[1],
		})
	}

	return cookies
}

// GetCookieByName finds a cookie by name from a cookie string.
func GetCookieByName(cookieStr, name string) *http.Cookie {
	for _, cookie := range ParseCookies(cookieStr) {
		if cookie.Name == name {
			return cookie
		}
	}

	return nil
}

func parseJSONCookies(cookieStr string) ([]*http.Cookie, error) {
	var serialized []*http.Cookie
	if err := json.Unmarshal([]byte(cookieStr), &serialized); err != nil {
		return nil, fmt.Errorf("parsing cookies json: %w", err)
	}

	return serialized, nil
}
