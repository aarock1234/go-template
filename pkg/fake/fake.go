// Package fake provides random data generation for testing and development.
// Its functions are safe for concurrent use.
package fake

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"unicode/utf8"
)

// FirstName returns a random first name.
func FirstName() string {
	return pick(firstNames)
}

// LastName returns a random last name.
func LastName() string {
	return pick(lastNames)
}

// Email returns a random email address such as "mary.smith4821@gmail.com".
func Email() string {
	return strings.ToLower(fmt.Sprintf("%s.%s%d@%s", FirstName(), LastName(), between(1000, 10000), pick(emailDomains)))
}

// StudentEmail returns an email address built from a first name, the
// initial of a last name, and a number at domain, such as
// "johnd42@example.edu".
func StudentEmail(firstName, lastName, domain string) string {
	initial := "x"
	if lastName != "" {
		_, size := utf8.DecodeRuneInString(lastName)
		initial = lastName[:size]
	}

	return strings.ToLower(fmt.Sprintf("%s%s%d@%s", firstName, initial, between(10, 100), domain))
}

// pick returns a random element of values, which must not be empty.
func pick(values []string) string {
	return values[rand.IntN(len(values))]
}

// between returns a random integer in [lo, hi).
func between(lo, hi int) int {
	return lo + rand.IntN(hi-lo)
}
