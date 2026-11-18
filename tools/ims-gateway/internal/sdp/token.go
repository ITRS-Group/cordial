package sdp

import (
	"context"
	"crypto/sha256"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// tokenKey identifies the settings that a token is for, so that a
// change to any of them, such as the client secret, is a new source
type tokenKey struct {
	clientID   string
	tokenURL   string
	secret     [sha256.Size]byte
	timeout    time.Duration
	skipVerify bool
	trace      bool
}

var (
	tokenSourcesMutex sync.Mutex
	tokenSources      = make(map[tokenKey]*refreshingSource)
)

// sharedTokenSource returns the token source for key that is used by
// every request in the process. A client is created for each request
// but if each had its own source then all the requests that find the
// token has expired would refresh it at the same time, which is rate
// limited by Zoho. The first time a source is needed the saved token is
// read, which is an error if there is none.
//
// The HTTP client in base is used for refreshing the token.
func sharedTokenSource(key tokenKey, conf *oauth2.Config, base *http.Client) (oauth2.TokenSource, error) {
	tokenSourcesMutex.Lock()
	defer tokenSourcesMutex.Unlock()

	if s, ok := tokenSources[key]; ok {
		return s, nil
	}

	token, err := tokenLoader()
	if err != nil {
		return nil, err
	}

	s := &refreshingSource{
		conf:  conf,
		ctx:   context.WithValue(context.Background(), oauth2.HTTPClient, base),
		token: token,
	}
	tokenSources[key] = s
	return s, nil
}

// refreshingSource is an [oauth2.TokenSource] that refreshes the token,
// and saves it, when it has expired. Only one request refreshes it, and
// the others wait for the result.
type refreshingSource struct {
	conf *oauth2.Config
	ctx  context.Context

	mu    sync.Mutex
	token *oauth2.Token
}

// Token returns the current token, refreshed if it has expired
func (s *refreshingSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// each caller is given a copy as oauth2 changes the token it gets
	if s.token.Valid() {
		return new(*s.token), nil
	}

	// a token that has been saved since, for example by the `sdp-auth`
	// command, is used if it is newer
	refresh := s.token.RefreshToken
	if saved, err := tokenLoader(); err == nil && saved.Expiry.After(s.token.Expiry) {
		if saved.Valid() {
			s.token = saved
			return new(*saved), nil
		}
		refresh = saved.RefreshToken
	}

	token, err := s.conf.TokenSource(s.ctx, &oauth2.Token{RefreshToken: refresh}).Token()
	if err != nil {
		return nil, err
	}

	// the refreshed token is valid, so a failure to save it should
	// not fail the request, but must be visible as it means the
	// next process has to refresh the token again
	if err := tokenSaver(token); err != nil {
		log.Error("failed to save refreshed ServiceDesk Plus token", slog.Any("error", err))
	}
	s.token = token
	return new(*token), nil
}

// resetTokenSources discards all of the shared token sources, for tests
func resetTokenSources() {
	tokenSourcesMutex.Lock()
	defer tokenSourcesMutex.Unlock()
	clear(tokenSources)
}
