# ITRS Geneos to Pagerduty Event Integration

This program allows you to send ITRS Geneos alerts to Pagerduty as Events which can, in turn, raise Incidents.

## Getting Started

You can either download a pre-built binary or build from source.

## Configuration

The integration takes it settings from the following, in order of priority from first to last:

1. Command line flags
2. Configuration file
3. External Defaults File
4. Internal Defaults

Geneos passes alert information to external programs and scripts using environment variables. These are used by the configuration options to build a Pagerduty Event in [PD-CEF](https://support.pagerduty.com/docs/pd-cef) format.
