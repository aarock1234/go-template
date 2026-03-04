package client

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

var _ http.CookieJar = (*CookieJar)(nil)

// CookieJar is a cookie jar that accepts cookie values containing
// double-quote characters. It wraps the standard library's cookiejar
// with relaxed validation.
type CookieJar struct {
	jar *cookiejar.Jar
}

func newCookieJar() (*CookieJar, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	return &CookieJar{jar: jar}, nil
}

// SetCookies stores cookies for the given URL. Cookie values may contain
// double-quote characters that the standard library would reject.
func (j *CookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.jar.SetCookies(u, cookies)
}

// Cookies returns the cookies for the given URL.
func (j *CookieJar) Cookies(u *url.URL) []*http.Cookie {
	return j.jar.Cookies(u)
}
