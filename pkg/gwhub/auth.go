package gwhub

import (
	"context"
	"net/url"

	"github.com/itrs-group/cordial/pkg/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// Auth with client ID and Secret. If clientid is empty just return,
// allowing callers to call with config values even when not set.
func (h *Hub) Auth(ctx context.Context, clientid string, clientsecret *config.Plaintext) {
	if clientid == "" {
		return
	}
	params := make(url.Values)
	params.Set("grant_type", "client_credentials")
	tokenauth, _ := url.JoinPath(h.BaseURL + "/oauth2/token")
	conf := &clientcredentials.Config{
		ClientID:       clientid,
		ClientSecret:   clientsecret.String(),
		EndpointParams: params,
		TokenURL:       tokenauth,
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, h.client)
	h.client = conf.Client(ctx)
}
