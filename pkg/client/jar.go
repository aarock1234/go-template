package client

import (
	"net/url"

	ehttp "github.com/enetx/http"
	"github.com/enetx/http/cookiejar"
	http "github.com/saucesteals/fhttp"
)

var _ http.CookieJar = &cookieJar{}

type cookieJar struct {
	jar *cookiejar.Jar
}

func newCookieJar() (*cookieJar, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	return &cookieJar{
		jar: jar,
	}, nil
}

func (c *cookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	var enetxCookies []*ehttp.Cookie
	for _, cookie := range cookies {
		enetxCookies = append(enetxCookies, toEnetxCookie(cookie))
	}

	c.jar.SetCookies(u, enetxCookies)
}

func (c *cookieJar) Cookies(u *url.URL) []*http.Cookie {
	var cookies []*http.Cookie
	for _, cookie := range c.jar.Cookies(u) {
		cookies = append(cookies, toNetHTTPCookie(cookie))
	}

	return cookies
}

func toNetHTTPCookie(cookie *ehttp.Cookie) *http.Cookie {
	return &http.Cookie{
		Name:  cookie.Name,
		Value: cookie.Value,
		// Quoted:     cookie.Quoted,
		Path:       cookie.Path,
		Domain:     cookie.Domain,
		Expires:    cookie.Expires,
		RawExpires: cookie.RawExpires,
		MaxAge:     cookie.MaxAge,
		Secure:     cookie.Secure,
		HttpOnly:   cookie.HttpOnly,
		SameSite:   http.SameSite(cookie.SameSite),
		// Partitioned: cookie.Partitioned,
		Raw:      cookie.Raw,
		Unparsed: cookie.Unparsed,
	}
}

func toEnetxCookie(cookie *http.Cookie) *ehttp.Cookie {
	return &ehttp.Cookie{
		Name:  cookie.Name,
		Value: cookie.Value,
		// Quoted:     cookie.Quoted,
		Path:       cookie.Path,
		Domain:     cookie.Domain,
		Expires:    cookie.Expires,
		RawExpires: cookie.RawExpires,
		MaxAge:     cookie.MaxAge,
		Secure:     cookie.Secure,
		HttpOnly:   cookie.HttpOnly,
		SameSite:   ehttp.SameSite(cookie.SameSite),
		// Partitioned: cookie.Partitioned,
		Raw:      cookie.Raw,
		Unparsed: cookie.Unparsed,
	}
}
