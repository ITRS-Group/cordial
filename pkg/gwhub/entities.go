package gwhub

import (
	"context"
	"net/http"
	"net/url"
)

// Entities type
//
// Not documented
type Entities []Entity

// Entity type
type Entity struct {
	ID         int               `json:"id"`
	Attributes map[string]string `json:"attributes"`
}

// Entities request
func (h *Hub) Entities(ctx context.Context, response *Entities) (resp *http.Response, err error) {
	return h.Get(ctx, EntitiesEndpoint, nil, &response)
}

// EntitiesAttributes type
type EntitiesAttributes []EntityAttributes

// EntityAttributes type
type EntityAttributes struct {
	Name       string `json:"name"`
	ValueCount int    `json:"valueCount"`
}

// EntitiesAttributes request
func (h *Hub) EntitiesAttributes(ctx context.Context, response *EntitiesAttributes) (resp *http.Response, err error) {
	return h.Get(ctx, EntitiesAttributesEndpoint, nil, &response)
}

// EntityAttributeValues type
type EntityAttributeValues []string

// EntityAttributeValues request
func (h *Hub) EntityAttributeValues(ctx context.Context, target string, response *EntityAttributeValues) (resp *http.Response, err error) {
	endpoint, _ := url.JoinPath(EntitiesAttributesEndpoint, target)
	return h.Get(ctx, endpoint, nil, &response)
}
