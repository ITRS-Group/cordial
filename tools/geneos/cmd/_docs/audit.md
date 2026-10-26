The `geneos` audit feature provides a record of activity. It can help administrators review what happened, when it happened, and which user or resource was involved. The information available depends on the activity and the configuration of your Geneos environment.

## Audit Logs

Audit logs capture commands and underlying actions performed with the `geneos` program. They provide a chronological record of activities, including the user who performed the action, the affected resource, and the outcome. These logs are useful for monitoring, troubleshooting, and ensuring compliance with organizational policies.

There is a global audit log, written to `${GENEOS_HOME}/audit/audit.log` by default. This log contains a record of all commands and the actions they trigger for the majority of commands available in `geneos`. Certain commands that have no side-effects, such as `list`/`ls` and `show`, are typically not recorded, while others that would be recorded in other places are not captured (by default), e.g. `geneos snapshot`.

Additionally, each instance will also have its own audit log, typically located called `TYPE-audit.log` within the instance's directory, where `TYPE` corresponds to the instance type (e.g., `gateway-audit.log` for a gateway instance). This allows for tracking actions specific to that instance, complementing the global audit log. Certain destructive actions, such as `geneos delete` will remove the instance directory, including the audit log and so administrators should refer to the global audit log referred to above.

### Audit Log Structure and Contents

Audit logs are written as single-line JSON per event. Each log entry typically includes the following fields:

- `timestamp`

  The date and time when the action occurred, in ISO 8601 format.

- `command` or `action`

  The type of event that occurred, one `command` being logged for each invocation of the `geneos` program and `action` lines for the underlying actions triggered by the command. Typically this is the `geneos` command name, e.g. "add", "start" or "stop"

- `command_id`

  A unique identifier for the command that triggered the event. This can be used to correlate related lines in the audit log.

- `username`

  The user name of the user who performed the action.

- `login`

  The login name of the user who performed the action. This is only included when different from the `username` and is derived from the Linux `/proc/self/loginuid` pseudo-file.

Then, depending if the event is a `command` or an `action` additional fields may be included to capture relevant details.

- `commandline` - For command events, this field contains the full command line that was executed. For actions, this field may include the specific command line arguments relevant to the action being performed.

- `instance` - For actions on instances, this field contains a map with details about the instance affected by the action. This typically includes the component type, instance name, the host, and other relevant attributes.

- `result` - For actions, this field contains the outcome of the action, such as "success" or "failure". It may also include additional details or error messages if the action did not complete successfully.

Examples:

```json

```

## Configuration

The audit feature can be configured through the Geneos environment settings. This includes specifying which actions are logged, the retention period for audit records, and access controls for viewing and managing the audit logs. Ensure that the configuration aligns with your organization's security and compliance requirements.

Two levels of configuration are available, global and per-instance. Instance settings override global settings for the specific instance they apply to. The options and names are the same for both, as follows:

- `audit.filename` - Specifies the file path where audit records are stored. Default is `${GENEOS_HOME}/audit/audit.log` for the global settings and `TYPE-audit.log` in the instance home directory for the per-instance settings, where `TYPE` is the instance type.
- `audit.max-size` - Specifies the maximum size of the audit log file in MiB (mebibytes - 1024 * 1024 bytes) before it is rotated. Default is 10MiB
- `audit.max-age` - Specifies the maximum age of the audit log file in days before it is rotated. Default is 30 days.
- `audit.max-backups` - Specifies the maximum number of rotated audit log files to retain. Default is 5.
- `audit.compression` - Specifies whether rotated audit log files should be compressed. This can help save disk space by compressing older log files. Default is to use `gzip`. Valid values are `gzip`, `zstd` and `none`.

## Access Control

Audit files are created with restricted access and the top-level `audit` directory is only accessible to the user running the Geneos process and administrators with appropriate permissions. There is no way to stop the user running the Geneos process from accessing the audit files, so it is important to ensure that this user account is properly secured and monitored.
