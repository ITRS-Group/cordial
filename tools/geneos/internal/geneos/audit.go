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

package geneos

// NotifyAudit records an operational event when the instance component
// type defines an Audit callback.
func NotifyAudit(i Instance, event string, fields map[string]string) {
	if i == nil {
		return
	}
	ct := i.Type()
	if ct == nil || ct.Audit == nil {
		return
	}
	_ = ct.Audit(i, event, fields)
}
