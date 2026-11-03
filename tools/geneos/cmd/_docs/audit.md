`geneos` audit logging records selected command invocations and the actions they perform. Audit events are written as one JSON object per line (JSONL) and can be correlated by `command_id`.

## Log locations

The global audit log is written to `<Geneos home>/audit/audit.log` by default. The global `audit.file` setting changes this path. The global writer is initialized only when a local Geneos home is available.

Actions on local instances are written to the global log and to an instance audit log. The instance log defaults to `TYPE-audit.log` relative to the instance home, where `TYPE` is the component type; the instance's `audit.file` setting overrides that path. Actions on remote instances are written to the global log only. Deleting an instance removes its local audit file with the instance directory, so use the global log when investigating deletion.

Whether a command or its actions are logged is controlled by the command's audit annotations. Read-only commands are commonly excluded; lifecycle and configuration commands commonly record actions. Not every command produces both event types.

## Event fields

Each event includes:

* `timestamp` - Event time in RFC3339 format, including fractional seconds when present.
* `command_id` - UUID shared by audit events emitted during one `geneos` process invocation.
* `username` - Operating-system user running `geneos`; `login` is included when the login UID name differs.
* `command` or `action` - The event name. Command events use `command` (for example, `geneos start`); action events use `action` (for example, `start`).

Command events include `commandline`, the invoked arguments as a single string. Sensitive values recognized by the logger are redacted. Instance action events include an `instance` object with `component`, `name`, and `host`, plus `result` (`success` or the error). Additional action-specific fields may also be present. The logger does not emit a separate `event_id` field.

Example command and action events:

```json
{"timestamp":"2026-11-03T12:34:56.123456789Z","command":"geneos start","command_id":"7d87c2fb-9a9b-44fa-9acd-04f70d07df2b","username":"geneos","commandline":"geneos start gateway MARKET"}
{"timestamp":"2026-11-03T12:34:56.234567890Z","action":"start","command_id":"7d87c2fb-9a9b-44fa-9acd-04f70d07df2b","username":"geneos","instance":{"component":"gateway","name":"MARKET","host":"localhost"},"result":"success"}
```

## Configuration

Global and per-instance audit writers support these settings:

* `audit.file` - Log path. Global default: `<Geneos home>/audit/audit.log`; instance default: `TYPE-audit.log` relative to the instance home.
* `audit.max-size` - Maximum log size in MiB before size-based rotation.
* `audit.max-age` - Maximum age in days for rotated logs; zero disables age-based cleanup.
* `audit.max-backups` - Number of rotated logs to retain; zero retains all backups unless `max-age` removes them.
* `audit.compression` - Compression for rotated logs: `none`, `gzip`, or `zstd`.

The global writer also rotates at midnight and on a seven-day interval. The current effective global defaults come from timberjack's zero-value behavior: 100 MiB, no age or backup-count cleanup, and no compression. The current per-instance writer defaults to 10 MiB, 30 days, and five backups; compression is none unless explicitly configured.

Command/action selection is currently controlled by per-command annotations. The parser checks whether `audit.commands` and `audit.actions` are set but does not read their boolean values, so these settings do not override the per-command annotations.

## File access

The global `audit` directory is created with mode `0750`. Timberjack creates log files with mode `0640` by default, subject to the process umask; group members may therefore have read access. Secure the Geneos account and its groups accordingly. Audit logs can contain operational details and should be protected as sensitive data.
