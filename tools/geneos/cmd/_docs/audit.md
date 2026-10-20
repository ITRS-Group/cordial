The `geneos` audit feature provides a record of administrative activity. It can help administrators review what happened, when it happened, and which user or resource was involved. The information available depends on the activity and the configuration of your Geneos environment.

## Audit Logs

Audit logs capture administrative actions performed with the `geneos` program. They provide a chronological record of activities, including the user who performed the action, the affected resource, and the outcome. These logs are essential for monitoring, troubleshooting, and ensuring compliance with organizational policies.

There is a global audit log, written to `${GENEOS_HOME}/audit/audit.log` by default. This log contains a record of all commands and the actions they trigger for the majority of commands available in `geneos`. Certain commands that have no side-effects, such as `list`/`ls` and `show`, are typically not recorded.

Additionally, each instance will also have its own audit log, typically located called `TYPE-audit.log` within the instance's directory, where `TYPE` corresponds to the instance type (e.g., `gateway-audit.log` for a gateway instance). This allows for tracking actions specific to that instance, complementing the global audit log. Certain destructive actions, such as `geneos delete` will remove the instance directory, including the audit log and so administrators should refer to the global audit log referred to above.

### Audit Log Structure and Contents

Audit logs are written as single-line JSON per event. Each log entry typically includes the following fields:

- `timestamp` - The date and time when the action occurred.
- `event` - The type of action or event that occurred. Typically this is the `geneos` command name, e.g. "add", "start" or "stop"
- `event_id`-  A unique identifier for the command that triggered the event. This can be used to collate related events.
- `username` -  The user name of the user who performed the action.
- `login` - The login name of the user who performed the action. This is only included when different from the `username` and is derived from the Linux `/proc/self/loginuid` pseudo-file.

Then, depending if the event is a top-level command or a subsequent action, additional fields may be included to capture relevant details.

- `command` - For command events, this field contains the full command line that was executed.

- `instance` - A map containing details about the instance affected by the action. This typically includes the instance type, name, and other relevant attributes.

Examples:

```json
...
```

## Configuration

The audit feature can be configured through the Geneos environment settings. This includes specifying which actions are logged, the retention period for audit records, and access controls for viewing and managing the audit logs. Ensure that the configuration aligns with your organization's security and compliance requirements.

Two levels of configuration are available, global and per-instance. Instance settings override global settings for the specific instance they apply to. The options and names are the same for both, as follows:

- `audit::filename` - Specifies the file path where audit records are stored.
- `audit::max-size` - Specifies the maximum size of the audit log file before it is rotated.

