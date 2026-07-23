package jwks

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/llnl/wormhole-airlock/internal/ctls/logs"
	"github.com/llnl/wormhole-airlock/internal/ctls/web"
)

var (
	maxRetry  = 5
	waitRetry = 15 * time.Second
)

type WebKeys interface {
	FetchKey(kid string) (*rsa.PublicKey, error)
	PrintDetails()
	UpdateKeys()
}

type keyMap map[string]*rsa.PublicKey

type keyStore struct {
	keys keyMap
}

type data struct {
	client web.Client
	keys   atomic.Pointer[keyStore]
	ll     logs.Logger
	url    string
}

func Initialize(client web.Client, ll logs.Logger, url string) (WebKeys, error) {
	d := &data{
		client: client,
		ll:     ll,
		url:    url,
	}

	ll.Info("loading keys from " + url)

	keys, err := d.requestJWKS()
	if err != nil {
		return nil, err
	}

	// Store initial keys atomically
	d.keys.Store(&keyStore{keys: keys})

	return d, nil
}

//

func (d *data) FetchKey(kid string) (*rsa.PublicKey, error) {
	keys, err := d.loadKeys()
	if err != nil {
		return nil, err
	}

	pub := keys[kid]
	if pub == nil {
		return nil, fmt.Errorf("no matching public key for %s", kid)
	}

	return pub, nil
}

func (d *data) UpdateKeys() {
	newKeys, err := d.requestJWKS()
	if err != nil {
		d.ll.Warn("skipping public key update due to: " + err.Error())
		return
	}

	// Atomic swap - no lock needed
	d.keys.Store(&keyStore{keys: newKeys})

	d.ll.Debug("successfully updated public keys")
}

func (d *data) PrintDetails() {
	keys, err := d.loadKeys()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("Loaded keys from " + d.url)

	for k := range keys {
		fmt.Printf("\tKid: %s\n", k)
	}
}

//

func (d *data) loadKeys() (keyMap, error) {
	ks := d.keys.Load()
	if ks == nil {
		return nil, errors.New("keys not initialized")
	}

	return ks.keys, nil
}

func (d *data) requestJWKS() (map[string]*rsa.PublicKey, error) {
	keys := make(map[string]*rsa.PublicKey)
	jwks := WebKeySet{}

	var err error

	for range maxRetry {
		err = d.client.GetJSON(d.url, &jwks)
		if err == nil {
			break
		}

		time.Sleep(waitRetry)
	}

	if err != nil {
		return keys, fmt.Errorf("failed to fetch public keys from %s: %w", d.url, err)
	}

	for _, v := range jwks.Keys {
		pub, err := v.establishRSA()
		if err != nil {
			return keys, fmt.Errorf("key %s: %w", v.Kid, err)
		}

		keys[string(v.Kid)] = pub
	}

	return keys, nil
}
