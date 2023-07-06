package gwhub

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/itrs-group/cordial/pkg/config"
)

func randValue() (value string) {
	b := make([]byte, 16)
	rand.Read(b)
	value = hex.EncodeToString(b)
	return
}

type authResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Expires      uint64 `json:"expires"`
	TokenType    string `json:"token_type"`
}

func (h *Hub) Login(ctx context.Context, username string, password *config.Plaintext) (err error) {
	v := url.Values{}
	v.Add("response_type", "code")
	v.Add("client_id", "rest")
	state := randValue()
	v.Add("state", state)

	p, _ := url.JoinPath(h.BaseURL, "/authorize")
	req, _ := http.NewRequestWithContext(ctx, "GET", p, nil)
	req.URL.RawQuery = v.Encode()

	resp, err := h.client.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode > 299 {
		panic(resp.Status)
	}
	var authresp authResponse
	d := json.NewDecoder(resp.Body)
	err = d.Decode(&authresp)
	if err != nil {
		panic(err)
	}
	resp.Body.Close()
	h.token = authresp.AccessToken

	v = url.Values{}
	v.Add("response_type", "token")
	v.Add("client_id", "rest")
	state = randValue()
	v.Add("state", state)
	nonce := randValue()
	v.Add("nonce", nonce)
	req, _ = http.NewRequestWithContext(ctx, "GET", p, nil)
	req.URL.RawQuery = v.Encode()
	req.Header.Add("Authorization", "Bearer "+h.token)

	resp, err = h.client.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode > 299 {
		panic(resp.Status)
	}
	d = json.NewDecoder(resp.Body)
	err = d.Decode(&authresp)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%#v", authresp)
	return
}
