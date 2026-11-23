package client

import (
	"maps"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	http "github.com/aarock1234/fphttp/v2"
	"golang.org/x/net/publicsuffix"
)

// CookieJar stores cookies by domain. A cookie with a Domain attribute
// matches that domain and its subdomains; a cookie without one matches only
// the host that set it. A cookie whose Domain attribute does not cover the
// host that set it, or names a public suffix such as "com", is ignored. Path
// and Secure attributes are not enforced.
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
// MaxAge is negative, deletes the stored cookie instead. A cookie whose
// Domain attribute does not cover u's host, or names a public suffix, is
// skipped.
func (j *CookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.cookies == nil {
		j.cookies = make(map[string][]*http.Cookie)
	}

	now := time.Now()

	for _, cookie := range cookies {
		key, ok := cookieKey(u, cookie)
		if !ok {
			continue
		}

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

// cookieKey returns the key that stores a cookie received from u, and
// whether u may set it. A Domain attribute must name u's host or one of its
// parent domains, and must not be a public suffix such as "com"; a host may
// still set a cookie for itself.
func cookieKey(u *url.URL, cookie *http.Cookie) (key string, ok bool) {
	host := strings.ToLower(u.Hostname())
	domain := strings.ToLower(strings.TrimPrefix(cookie.Domain, "."))
	domainKey := "." + domain

	switch {
	case domain == "":
		return host, true
	case isIPAddress(host), isPublicSuffix(domain):
		// Neither has subdomains a cookie may cover, so the Domain attribute
		// can only name the host itself, which makes the cookie host-only.
		return host, domain == host
	case domainMatch(host, domainKey):
		return domainKey, true
	default:
		return "", false
	}
}

// isIPAddress reports whether host is an IP address rather than a domain
// name.
func isIPAddress(host string) bool {
	_, err := netip.ParseAddr(host)

	return err == nil
}

// isPublicSuffix reports whether domain is a suffix under which unrelated
// parties register names, such as "com" or "co.uk".
func isPublicSuffix(domain string) bool {
	suffix, _ := publicsuffix.PublicSuffix(domain)

	return suffix == domain
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
