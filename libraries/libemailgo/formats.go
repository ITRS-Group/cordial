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
