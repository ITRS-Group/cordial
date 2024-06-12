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

package main

const (
	_SUBJECT = iota
	_ALERT_SUBJECT
	_CLEAR_SUBJECT
	_SUSPEND_SUBJECT
	_RESUME_SUBJECT
	_SUMMARY_SUBJECT
)

const (
	_FORMAT = iota
	_ALERT_FORMAT
	_CLEAR_FORMAT
	_SUSPEND_FORMAT
	_RESUME_FORMAT
	_SUMMARY_FORMAT
)

var defaultSubject = []string{
	"Geneos Alert",
	"Geneos Alert Fired",
	"Geneos Alert Cancelled",
	"Geneos Alert Suspended",
	"Geneos Alert Resumed",
	"Geneos Alert Throttle Summary",
}

var defaultMsTeamsSubject = []string{
	"Geneos Alert",
	"Geneos Alert Fired",
	"Geneos Alert Cancelled",
	"Geneos Alert Suspended",
	"Geneos Alert Resumed",
	"Geneos Alert Throttle Summary",
}

var defaultFormat = []string{

	`This is an automatically generated mail from Geneos Gateway: %(_GATEWAY)

Action "%(_ACTION)" is being fired against Geneos DataItem %(_VARIABLEPATH)
	
The dataitem value is "%(_VALUE)" and its severity is %(_SEVERITY)`,

	`This is an automatically generated mail from Geneos Gateway: %(_GATEWAY)

Alert "%(_ALERT)" is being fired because Geneos DataItem %(_VARIABLE) in dataview %(_DATAVIEW) in Managed Entity %(_MANAGED_ENTITY) is at %(_SEVERITY) severity.

The cell value is "%(_VALUE)"

This Alert was created at %(_ALERT_CREATED) and has been fired %(_REPEATCOUNT) times.

The item's XPath is %(_VARIABLEPATH)

This alert is controlled by throttle: "%(_THROTTLER)".`,

	`This is an automatically generated mail from Geneos Gateway: %(_GATEWAY).

Alert "%(_ALERT)" is being cancelled because Geneos DataItem %(_VARIABLE) in dataview %(_DATAVIEW) in Managed Entity %(_MANAGED_ENTITY) is at %(_SEVERITY) severity.

The cell value is "%(_VALUE)"

This Alert was created at %(_ALERT_CREATED) and has been fired %(_REPEATCOUNT) times.

The item's XPath is %(_VARIABLEPATH)

This alert is controlled by throttle: "%(_THROTTLER)".`,

	`This is an automatically generated mail from Geneos Gateway: %(_GATEWAY).

Alert "%(_ALERT)" is being suspended because of: "%(_SUSPEND_REASON)". No notifications will be fired for this alert until it is resumed. If the alert is cancelled before it is resumed no further notifications will be fired.

The cell value is "%(_VALUE)"

This Alert was created at %(_ALERT_CREATED) and has been fired %(_REPEATCOUNT) times.

The item's XPath is %(_VARIABLEPATH)

This alert is controlled by throttle: "%(_THROTTLER)".`,

	`This is an automatically generated mail from Geneos Gateway: %(_GATEWAY).

Alert "%(_ALERT)" is being resumed because of: "%(_RESUME_REASON)". Geneos DataItem %(_VARIABLE) in dataview %(_DATAVIEW) in Managed Entity %(_MANAGED_ENTITY) is %(_SEVERITY) severity.

The cell value is "%(_VALUE)"

This Alert was created at %(_ALERT_CREATED) and has been fired %(_REPEATCOUNT) times.

The item's XPath is %(_VARIABLEPATH)

This alert is controlled by throttle: "%(_THROTTLER)".`,

	`This is an automatically generated mail from Geneos Gateway: %(_GATEWAY)

Summary for alert throttle "%(_THROTTLER)"
%(_VALUE) Alerts have been throttled in the last %(_SUMMARY_PERIOD), including:
%(_DROPPED_ALERTS) Alert(s)
%(_DROPPED_CLEARS) Clear(s)
%(_DROPPED_SUSPENDS) Suspend(s)
%(_DROPPED_RESUMES) Resume(s)`,
}
