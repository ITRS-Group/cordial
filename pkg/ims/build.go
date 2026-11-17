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

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/itrs-group/cordial/pkg/config"
)

// BuildIncident sets the fields in incident by evaluating the action
// groups in the `defaults` section of the client configuration cf and
// then those in the section for the profile, `profiles.NAME`. The
// profile used is profile, if not empty, otherwise the value of the
// field profileField, as set by the defaults, and finally `default`.
//
// An error is returned if either section is not a valid list of action
// groups. A profile that is not defined, other than `default`, is
// logged as a warning and only the defaults are used.
func BuildIncident(cf *config.Config, incident Values, profile, profileField string) (err error) {
	var defaults []ActionGroup
	var actionGroups []ActionGroup

	if err = cf.UnmarshalKey("defaults", &defaults, config.NoExpand()); err != nil {
		return fmt.Errorf("invalid `defaults` configuration: %w", err)
	}
	for _, g := range defaults {
		if ProcessActionGroup(cf, g, incident) {
			break
		}
	}

	logFields("incident fields after processing defaults", incident)

	if profile == "" {
		var ok bool
		if profile, ok = incident[profileField]; !ok {
			profile = "default"
		}
	}

	profileKey := cf.Join("profiles", profile)
	if err = cf.UnmarshalKey(profileKey, &actionGroups, config.NoExpand()); err != nil {
		return fmt.Errorf("invalid configuration for profile %q: %w", profile, err)
	}
	if !cf.IsSet(profileKey) {
		if profile == "default" {
			log.Debug("no default profile defined")
		} else {
			log.Warn("profile is not defined in the configuration, only defaults will be used", slog.String("profile", profile))
		}
	}
	for _, ag := range actionGroups {
		log.Debug("processing profile", slog.String("profile", profile), slog.Any("group", ag))
		if ProcessActionGroup(cf, ag, incident) {
			break
		}
	}

	logFields("incident fields after processing profile", incident)

	return nil
}

func logFields(msg string, incident Values) {
	b, _ := json.MarshalIndent(incident, "", "    ")
	log.Debug(msg, slog.String("fields", string(b)))
}
