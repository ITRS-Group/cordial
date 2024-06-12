/*
Copyright © 2022 ITRS Group

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

// Standalone pagerduty integration executable
//
// Given a set of Geneos environment variables and a configuration file
// send events to pagerduty using EventV2 API
//
// Some behaviours are hard-wired;
//
// Severity OK is a Resolved
// Severity Warning or Critical is trigger
// Other Severity is mapped to ?
//
// Snooze or Userassignment is an Acknowledge
package main

import (
	_ "embed"
)
