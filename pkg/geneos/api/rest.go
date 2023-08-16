package api

import "errors"

// Documentation: https://docs.itrsgroup.com/docs/geneos/current/api/rest-api/?v=%252Fv1%252Frest-api.yaml

// Available entry points:
//
// - Create or update dataview
// - Delete dataview
// - Create or update row
// - Delete row
// - Create or update stream
// - Healthcheck

type RESTClient struct {
}

// check we implement all methods
var _ APIClient = (*RESTClient)(nil)

func NewRESTClient(endpoint string, options ...Options) (c APIClient, err error) {
	return
}

func (c *RESTClient) Healthy() bool {
	// get /healthcheck -> 200 OK
	return true
}

func (c *RESTClient) CreateDataview(entity, sampler, name string) (err error) {
	return
}

func (c *RESTClient) UpdateDataview(entity, sampler, name string, values [][]string) (err error) {
	return
}

func (c *RESTClient) DeleteDataview(entity, sampler, name string) (err error) {
	return
}

func (c *RESTClient) CreateRow(entity, sampler, view, name string) (err error) {
	return
}

func (c *RESTClient) UpdateRow(entity, sampler, view, name string, values []string) (err error) {
	return
}

func (c *RESTClient) DeleteRow(entity, sampler, view, name string) (err error) {
	return
}

func (c *RESTClient) CreateStream(entity, sampler, name string) (err error) {
	return
}

func (c *RESTClient) UpdateStream(entity, sampler, name, message string) (err error) {
	return
}

func (c *RESTClient) DataviewExists(entity, sampler, name string) (bool, error) {
	return false, errors.ErrUnsupported
}
