package jwks

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/llnl/wormhole-airlock/internal/ctls/util"
)

type WebKey struct {
	Alg string           `json:"alg"`
	Kty string           `json:"kty"`
	Kid util.StringOrInt `json:"kid"`
	E   *exponent        `json:"e"`
	N   *modulus         `json:"n"`
}

type WebKeySet struct {
	Keys []WebKey `json:"keys"`
}

//

func (wk WebKey) establishRSA() (*rsa.PublicKey, error) {
	if wk.N == nil || wk.E == nil {
		return nil, errors.New("failed to identify public key, verify JWKS algorithm type")
	}

	pubKey := &rsa.PublicKey{
		N: wk.N.data,
		E: wk.E.data,
	}

	// Validate the public key
	if pubKey.N == nil || pubKey.N.Sign() <= 0 {
		return nil, errors.New("invalid modulus")
	}

	if pubKey.E <= 0 {
		return nil, errors.New("invalid exponent")
	}

	return pubKey, nil
}

//

type modulus struct {
	data *big.Int
}

func (m *modulus) UnmarshalJSON(data []byte) error {
	buf, err := unmarshal(data)
	if err != nil {
		return err
	}

	m.data = new(big.Int).SetBytes(buf)

	return nil
}

type exponent struct {
	data int
}

func (e *exponent) UnmarshalJSON(data []byte) error {
	buf, err := unmarshal(data)
	if err != nil {
		return err
	}

	e.data = int(new(big.Int).SetBytes(buf).Int64())

	return nil
}

func unmarshal(data []byte) ([]byte, error) {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return []byte{}, err
	}

	return base64.RawURLEncoding.DecodeString(s)
}
