A `tr-gateway` component is a Java application managed by `geneos`. It is not downloaded from ITRS Geneos servers; install a local product archive.

The process is started with the JDK bundled in the archive (`jdk/bin/java`). To use a different runtime, set `program`. Additional JVM flags can be set with `java-options`, and heap sizes with `xms` and `xmx` (defaults `512m`).

The listening port used by the application is defined in the YAML configuration. The port shown by `geneos list` is only used for instance uniqueness.

## Package layout

Install the released `.tar.gz` or `.zip` as shipped. A wrapping top-level directory is stripped on install. Contents after install:

```text
lib/tr-gateway.jar
plugins/*.jar
jdk/bin/java
config/tr-gateway.yaml
config/logback.xml
```

Archive names are recognised without `--override`, including Maven snapshots:

```text
tr-gateway-1.0.0.tar.gz
tr-gateway-1.0.0-SNAPSHOT-linux-x64.tar.gz
tr-gateway-1.0.0-20260826.031023-95-linux-x64.tar.gz
```

Each distinct snapshot timestamp is installed as its own version directory. Use `--override tr-gateway:VERSION` only if the filename does not match. These packages are not downloaded from ITRS.

```bash
geneos package install tr-gateway ./tr-gateway-1.0.0-20260826.031023-95-linux-x64.tar.gz
```

Limit installation to a remote host with `-H HOST`. The archive path is local.

## YAML configuration

Each instance uses `tr-gateway.yaml` in the instance home directory. If that file is missing when the instance is created, it is copied from the packaged `config/tr-gateway.yaml`. Supply a custom file when creating the instance:

```bash
geneos add tr-gateway prod1 --import tr-gateway.yaml=./prod1.yaml --start
```

Or install the package and create the instance in one step:

```bash
geneos deploy tr-gateway prod1 --import tr-gateway.yaml=./prod1.yaml ./tr-gateway-1.0.0-20260826.031023-95-linux-x64.tar.gz -l
```

To update the YAML later:

```bash
geneos import tr-gateway prod1 tr-gateway.yaml=./prod1-v2.yaml
geneos restart tr-gateway prod1
```

The application does not reload YAML on signal; restart after every change.

Each operational event is appended as one JSON line to the instance audit log (default `tr-gateway-audit.log`). Each record uses a fixed field order: `timestamp`, `event`, `username`, `module`, then event-specific fields (sorted by key). Filter with `jq` on the `event` field:

```json
{"timestamp":"2026-08-28T02:35:10Z","event":"import","username":"jenkins","module":"tr-gateway","bytes":4521,"file":"/opt/geneos/tr-gateway/prod1/tr-gateway.yaml","sha256":"abc..."}
{"timestamp":"2026-08-28T02:40:00Z","event":"start","username":"jenkins","module":"tr-gateway","command":"/path/to/java ...","pid":12345}
{"timestamp":"2026-08-28T02:45:00Z","event":"stop","username":"jenkins","module":"tr-gateway","pid":12345}
{"timestamp":"2026-08-28T02:46:00Z","event":"restart","username":"jenkins","module":"tr-gateway","newPid":12346,"oldPid":12345}
{"timestamp":"2026-08-28T03:00:00Z","event":"update","username":"jenkins","module":"tr-gateway","base":"active_prod","version":"1.0.0"}
{"timestamp":"2026-08-28T03:05:00Z","event":"set","username":"jenkins","module":"tr-gateway","keys":"xms,xmx"}
{"timestamp":"2026-08-28T03:10:00Z","event":"add","username":"jenkins","module":"tr-gateway","port":"19000"}
```

Events are also recorded for `delete`, `enable`, and `disable`. `geneos logs` follows the Java log (`tr-gateway.log`), not the audit file.

Configure the audit log path per instance with `audit-log` (basename under the instance home), or set a default with `tr-gateway::audit-log-file` in the geneos configuration. When the file reaches 10 MiB it is rotated to `.1` (up to five files). Override size and retention with `tr-gateway::audit-max-bytes` and `tr-gateway::audit-max-files`.

## Configuration

Instance settings are stored in `tr-gateway.json` in the instance working directory. Use `geneos set` and `geneos unset` rather than editing that file directly.

### Instance Parameters

For general instance parameters see `geneos help set`. Parameters specific to `tr-gateway`:

* `setup` (Default: `${config:home}/tr-gateway.yaml`) — path to the YAML config passed to the Java main class.
* `jar` (Default: `lib/tr-gateway.jar`) — jar path relative to the installed package directory.
* `mainclass` (Default: `com.itrsgroup.trgateway.Main`)
* `program` (Default: bundled `jdk/bin/java` under the selected package version)
* `logback` (Default: packaged `config/logback.xml`)
* `audit-log` (Default: `tr-gateway-audit.log`) — operational audit log basename in the instance home directory.
* `xms` / `xmx` (Default: `512m`) — JVM heap sizes.
* `java-options` — extra JVM flags, split on spaces.
