// Package rules maintains established validation for shared aspects relating
// to a range of inbound requests, headers, and potential configurations.
package rules

import (
	"regexp"

	"github.com/go-playground/validator/v10"

	"github.com/llnl/wormhole-airlock/internal/ctls/util"
)

const (
	// Observe 8KB size maximum fallback for any check. Most can enforce
	// lower values if identified (nginx.org/en/docs/http/ngx_http_core_module.html).
	maxHeaderKB = 8192

	TagAuthToken   = "authToken"
	TagHeaderName  = "headerName"
	TagHeaderValue = "headerValue"
	TagKID         = "kid"
)

var (
	authTokenRegexp   = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-_.~]*$`) // #nosec G101
	headerNameRegexp  = regexp.MustCompile(`^[a-zA-Z0-9!#$%&'*+\-.^_` + "`" + `|~]+$`)
	headerValueRegexp = regexp.MustCompile(`^[\x09\x20-\x7E\x80-\xFF]*$`)
	kidRegexp         = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-_\.]{0,128}$`)
)

// NewValidator generates a new validator with the following custom rules registered:
//   - authToken (string): optionally ensures a valid generic authorization token has been
//     supplied that contains no potentially malicious characters and generic maximum length.
//   - headerName (string): optional header per RFC 7230.
//   - headerValue (string): optional header value, no control characters except HTAB.
//   - kid (string): optionally ensure key ID found in JWT header.
func NewValidator() *validator.Validate {
	v := validator.New()

	_ = v.RegisterValidation(TagAuthToken, checkAuthToken)
	_ = v.RegisterValidation(TagHeaderName, checkHeaderName)
	_ = v.RegisterValidation(TagHeaderValue, checkHeaderValue)
	_ = v.RegisterValidation(TagKID, checkKID)

	return v
}

var checkAuthToken validator.Func = func(fl validator.FieldLevel) bool {
	return optionalStrField(fl, authTokenRegexp)
}

var checkHeaderName validator.Func = func(fl validator.FieldLevel) bool {
	return optionalStrField(fl, headerNameRegexp)
}

var checkHeaderValue validator.Func = func(fl validator.FieldLevel) bool {
	return optionalStrField(fl, headerValueRegexp)
}

var checkKID validator.Func = func(fl validator.FieldLevel) bool {
	return optionalStrField(fl, kidRegexp)
}

//

func maximumHeader(s string) bool {
	// This is not strictly true for all potentially instance, but does at least set a
	// ceiling for otherwise uncapped values.
	return len(s) > maxHeaderKB
}

func optionalStrField(fl validator.FieldLevel, re *regexp.Regexp) bool {
	if s, ok := fl.Field().Interface().(string); ok {
		return stringTests(s, re)
	} else if s, ok := fl.Field().Interface().(util.StringOrInt); ok {
		return stringTests(string(s), re)
	}

	return false
}

func stringTests(s string, re *regexp.Regexp) bool {
	if maximumHeader(s) {
		return false
	} else if s == "" {
		return true
	}

	return re.MatchString(s)
}
