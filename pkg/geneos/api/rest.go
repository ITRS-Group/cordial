/*
Copyright © 2022 ITRS Group

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package api

import (
	"context"
	"errors"
	"net/url"

	"github.com/itrs-group/cordial/pkg/rest"
)

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
	*rest.Client
}

// check we implement all methods
var _ APIClient = (*RESTClient)(nil)

func NewRESTClient(endpoint string, options ...rest.Options) (c APIClient, err error) {
	options = append(options, rest.BaseURL(endpoint))
	return &RESTClient{Client: rest.NewClient(options...)}, nil
}

func (c *RESTClient) Healthy() bool {
	resp, err := c.Get(context.Background(), "/healthcheck", nil, nil)
	if err != nil || resp.StatusCode != 200 {
		return false
	}
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

func (c *RESTClient) CreateColumn(entity, sampler, view, name string) (err error) {
	return
}

func (c *RESTClient) CreateHeadline(entity, sampler, view, name string) (err error) {
	return
}

func (c *RESTClient) UpdateHeadline(entity, sampler, view, name, value string) (err error) {
	return
}

func (c *RESTClient) DeleteHeadline(entity, sampler, view, name string) (err error) {
	return
}

func (c *RESTClient) CreateStream(entity, sampler, name string) (err error) {
	endpoint, _ := url.JoinPath("managedEntity", entity, "sampler", sampler, "stream", name)
	_, err = c.Put(context.Background(), endpoint, nil, nil)
	return
}

func (c *RESTClient) UpdateStream(entity, sampler, name string, message any) (err error) {
	endpoint, _ := url.JoinPath("managedEntity", entity, "sampler", sampler, "stream", name)
	_, err = c.Put(context.Background(), endpoint, message, nil)
	return
}

func (c *RESTClient) ManagedEntityExists(entity string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (c *RESTClient) SamplerExists(entity, sampler string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (c *RESTClient) DataviewExists(entity, sampler, name string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (c *RESTClient) RowExists(entity, sampler, view, name string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (c *RESTClient) ColumnExists(entity, sampler, view, name string) (bool, error) {
	return false, errors.ErrUnsupported
}

func (c *RESTClient) HeadlineExists(entity, sampler, view, name string) (bool, error) {
	return false, errors.ErrUnsupported
}
