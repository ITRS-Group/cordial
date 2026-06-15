/*
Copyright © 2026 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ims

import "net/http"

// register endpoints for IMS functions

// Endpoint represents an API endpoint for the IMS integration,
// including the HTTP method, path, and handler function.
type Endpoint struct {
	Method  string // HTTP method (e.g. "POST", "GET")
	Path    string // URL path (e.g. "/create", "/update") relative to application base path and including any path parameters (e.g. "/create/{id}")
	Handler http.HandlerFunc
}

var Endpoints = []Endpoint{}

// RegisterEndpoint registers an API endpoint for the IMS integration,
// including the HTTP method, path, and handler function. The path
// should be relative to the application base path and can include path
// parameters (e.g. "/create/{id}").
func RegisterEndpoint(method, path string, handler http.HandlerFunc) {
	// TODO: validate method and path
	Endpoints = append(Endpoints, Endpoint{
		Method:  method,
		Path:    path,
		Handler: handler,
	})
}
