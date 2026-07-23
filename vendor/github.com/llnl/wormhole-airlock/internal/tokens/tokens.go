package tokens

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"

	"github.com/llnl/wormhole-airlock/internal/ctls/rules"
	"github.com/llnl/wormhole-airlock/internal/ctls/util"
	"github.com/llnl/wormhole-airlock/internal/jwks"
)

// ValidationResult stores both successful and failed validation results
type ValidationResult struct {
	Valid       bool
	Claims      Claims
	ErrorString string
	Timestamp   time.Time
}

type Claims struct {
	jwt.RegisteredClaims

	Subject string   `json:"sub"`
	Groups  []string `json:"groups"`
}

type JWTValidator interface {
	CacheExp(claims Claims) time.Duration
	CacheKey(encoded string) string
	CheckHeader(encoded string) (Header, error)
	Parse(h Header, encoded string) (Claims, error)
}

type Header struct {
	Alg string           `json:"alg"`
	Kid util.StringOrInt `json:"kid"`
	Typ string           `json:"typ"`
}

type data struct {
	decoder *base64.Encoding
	keys    jwks.WebKeys
	leeway  time.Duration
	opts    []jwt.ParserOption
	v       *validator.Validate
}

func Initialize(leeway time.Duration, keys jwks.WebKeys) JWTValidator {
	return data{
		decoder: base64.RawURLEncoding,
		keys:    keys,
		leeway:  leeway,
		opts: []jwt.ParserOption{
			jwt.WithLeeway(leeway),
		},
		v: rules.NewValidator(),
	}
}

//

func (d data) CacheExp(claims Claims) time.Duration {
	tarExp := claims.ExpiresAt.Add(d.leeway)

	return time.Until(tarExp)
}

func (data) CacheKey(encoded string) string {
	return fmt.Sprintf("jwt_%x", sha256.Sum256([]byte(encoded)))
}

func (d data) CheckHeader(encoded string) (Header, error) {
	// Split JWT into parts (header.payload.signature)
	parts := strings.SplitN(encoded, ".", 3)
	if len(parts) != 3 {
		return Header{}, errors.New("invalid JWT format: expected 3 parts")
	}

	// Decode only the header portion
	headerBytes, err := d.decoder.DecodeString(parts[0])
	if err != nil {
		return Header{}, fmt.Errorf("invalid JWT header encoding: %w", err)
	}

	// Unmarshal header JSON
	var h Header
	if err := json.Unmarshal(headerBytes, &h); err != nil {
		return Header{}, fmt.Errorf("invalid JWT header JSON: %w", err)
	}

	// Manual validation (faster than validator library)
	if err := validateHeader(&h); err != nil {
		return Header{}, err
	}

	return h, nil
}

func (d data) Parse(h Header, encoded string) (Claims, error) {
	claims := Claims{}

	pub, err := d.keys.FetchKey(string(h.Kid))
	if err != nil {
		return Claims{}, err
	}

	jt, err := jwt.ParseWithClaims(
		encoded,
		&claims,
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			return pub, nil
		},
		d.opts...,
	)

	if err == nil && jt.Valid {
		return claims, nil
	}

	switch {
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return Claims{}, errors.New("invalid token signature")
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return Claims{}, errors.New("token not yet valid")
	case errors.Is(err, jwt.ErrTokenExpired):
		return Claims{}, errors.New("token has expired")
	}

	// Catch all error for a token that is not valid.
	return Claims{}, fmt.Errorf("invalid token: %w", err)
}

//

func validateHeader(h *Header) error {
	// Validate algorithm: must start with "RS" and be exactly 5 chars (RS256, RS384, RS512)
	if len(h.Alg) != 5 || !strings.HasPrefix(h.Alg, "RS") {
		return fmt.Errorf("invalid algorithm: %s", h.Alg)
	}

	// Validate type
	if h.Typ != "JWT" {
		return fmt.Errorf("invalid type: %s (expected JWT)", h.Typ)
	}

	// Validate kid exists
	if len(h.Kid) == 0 {
		return errors.New("missing kid (key ID)")
	}

	return nil
}
