package client

import (
	"maps"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	http "github.com/aarock1234/fphttp"
)

// CookieJar stores cookies by domain. A cookie with a Domain attribute
// matches that domain and its subdomains; a cookie without one matches only
// the host that set it. Path and Secure attributes are not enforced.
//
// The zero value is an empty jar ready to use. Its methods are safe for
// concurrent use.
type CookieJar struct {
	mu sync.Mutex

	// cookies is keyed by host for host-only cookies and by a leading-dot
	// domain (".example.com") for domain cookies.
	cookies map[string][]*http.Cookie
}

// SetCookies stores cookies received from u, replacing stored cookies with
// the same name and domain. A cookie that is already expired, or whose
// MaxAge is negative, deletes the stored cookie instead.
func (j *CookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.cookies == nil {
		j.cookies = make(map[string][]*http.Cookie)
	}

	now := time.Now()

	for _, cookie := range cookies {
		key := jarKey(u, cookie)
		stored := j.cookies[key]

		index := slices.IndexFunc(stored, func(c *http.Cookie) bool {
			return c.Name == cookie.Name
		})

		switch {
		case expired(cookie, now) && index >= 0:
			stored = slices.Delete(stored, index, index+1)
		case expired(cookie, now):
			continue
		case index >= 0:
			stored[index] = storedCookie(cookie, now)
		default:
			stored = append(stored, storedCookie(cookie, now))
		}

		if len(stored) == 0 {
			delete(j.cookies, key)

			continue
		}

		j.cookies[key] = stored
	}
}

// Cookies returns copies of the unexpired cookies that match u's host,
// ordered by domain and then by when each cookie was first stored.
func (j *CookieJar) Cookies(u *url.URL) []*http.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()

	host := strings.ToLower(u.Hostname())
	now := time.Now()

	var matched []*http.Cookie

	for _, key := range slices.Sorted(maps.Keys(j.cookies)) {
		if !domainMatch(host, key) {
			continue
		}

		for _, cookie := range j.cookies[key] {
			if expired(cookie, now) {
				continue
			}

			copied := *cookie
			matched = append(matched, &copied)
		}
	}

	return matched
}

// Clear removes every cookie from the jar.
func (j *CookieJar) Clear() {
	j.mu.Lock()
	defer j.mu.Unlock()

	clear(j.cookies)
}

// jarKey returns the key that stores a cookie received from u.
func jarKey(u *url.URL, cookie *http.Cookie) string {
	if cookie.Domain == "" {
		return strings.ToLower(u.Hostname())
	}

	return "." + strings.ToLower(strings.TrimPrefix(cookie.Domain, "."))
}

// storedCookie returns a copy of cookie whose MaxAge is converted to an
// absolute expiry, so it can be checked later.
func storedCookie(cookie *http.Cookie, now time.Time) *http.Cookie {
	stored := *cookie
	if stored.MaxAge > 0 {
		stored.Expires = now.Add(time.Duration(stored.MaxAge) * time.Second)
	}

	return &stored
}

// expired reports whether cookie must not be stored or sent at now.
func expired(cookie *http.Cookie, now time.Time) bool {
	if cookie.MaxAge < 0 {
		return true
	}

	return !cookie.Expires.IsZero() && !cookie.Expires.After(now)
}

// domainMatch reports whether host matches a jar key. A leading-dot key
// matches the domain itself and any of its subdomains.
func domainMatch(host, key string) bool {
	domain, isDomain := strings.CutPrefix(key, ".")
	if !isDomain {
		return host == key
	}

	return host == domain || strings.HasSuffix(host, key)
}
