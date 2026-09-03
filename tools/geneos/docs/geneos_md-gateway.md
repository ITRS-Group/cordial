# `geneos md-gateway`

An `md-gateway` component is a Java application managed by `geneos`. It is not downloaded from ITRS Geneos servers; install a local product archive.

The process is started with the JDK bundled in the archive (`jdk/bin/java`). To use a different runtime, set `program`. Additional JVM flags can be set with `java-options`, and heap sizes with `xms` and `xmx` (defaults `512m`).

The listening port used by the application is defined in the YAML configuration. The port shown by `geneos list` is only used for instance uniqueness.

## Package layout

Install the released `.tar.gz` or `.zip` as shipped. A wrapping top-level directory is stripped on install. Contents after install:

```text
lib/md-gateway.jar
plugins/*.jar
jdk/bin/java
config/md-gateway.yaml
config/logback.xml
```

Archive names are recognised without `--override`, including Maven snapshots:

```text
md-gateway-1.0.0.tar.gz
md-gateway-1.0.0-SNAPSHOT-linux-x64.tar.gz
md-gateway-1.0.0-20260826.031023-95-linux-x64.tar.gz
```

Each distinct snapshot timestamp is installed as its own version directory. Use `--override md-gateway:VERSION` only if the filename does not match. These packages are not downloaded from ITRS.

```bash
geneos package install md-gateway ./md-gateway-1.0.0-20260826.031023-95-linux-x64.tar.gz
```

Limit installation to a remote host with `-H HOST`. The archive path is local.

## YAML configuration

Each instance uses `md-gateway.yaml` in the instance home directory. If that file is missing when the instance is created, it is copied from the packaged `config/md-gateway.yaml`. Supply a custom file when creating the instance:

```bash
geneos add md-gateway prod1 --import md-gateway.yaml=./prod1.yaml --start
```

Or install the package and create the instance in one step:

```bash
geneos deploy md-gateway prod1 --import md-gateway.yaml=./prod1.yaml ./md-gateway-1.0.0-20260826.031023-95-linux-x64.tar.gz -l
```

To update the YAML later:

```bash
geneos import md-gateway prod1 md-gateway.yaml=./prod1-v2.yaml
geneos restart md-gateway prod1
```

The application does not reload YAML on signal; restart after every change.

Each operational event is appended as one JSON line to the instance audit log (default `md-gateway-audit.log`). Each record uses a fixed field order: `timestamp`, `event`, `username`, `module`, then event-specific fields (sorted by key). Filter with `jq` on the `event` field:

```json
{"timestamp":"2026-08-28T02:35:10Z","event":"import","username":"jenkins","module":"md-gateway","bytes":4521,"file":"/opt/geneos/md-gateway/prod1/md-gateway.yaml","sha256":"abc..."}
{"timestamp":"2026-08-28T02:40:00Z","event":"start","username":"jenkins","module":"md-gateway","command":"/path/to/java ...","pid":12345}
{"timestamp":"2026-08-28T02:45:00Z","event":"stop","username":"jenkins","module":"md-gateway","pid":12345}
{"timestamp":"2026-08-28T02:46:00Z","event":"restart","username":"jenkins","module":"md-gateway","newPid":12346,"oldPid":12345}
{"timestamp":"2026-08-28T03:00:00Z","event":"update","username":"jenkins","module":"md-gateway","base":"active_prod","version":"1.0.0"}
{"timestamp":"2026-08-28T03:05:00Z","event":"set","username":"jenkins","module":"md-gateway","keys":"xms,xmx"}
{"timestamp":"2026-08-28T03:10:00Z","event":"add","username":"jenkins","module":"md-gateway","port":"18000"}
```

Events are also recorded for `delete`, `enable`, and `disable`. `geneos logs` follows the Java log (`md-gateway.log`), not the audit file.

Configure the audit log path per instance with `audit-log` (basename under the instance home), or set a default with `md-gateway::audit-log-file` in the geneos configuration. When the file reaches 10 MiB it is rotated to `.1` (up to five files). Override size and retention with `md-gateway::audit-max-bytes` and `md-gateway::audit-max-files`.

The archive also contains `geneos/md-gateway-rules.xml` for a Geneos Gateway include; that file is not used by this component.

## Configuration

Instance settings are stored in `md-gateway.json` in the instance working directory. Use `geneos set` and `geneos unset` rather than editing that file directly.

### Instance Parameters

For general instance parameters see `geneos help set`. Parameters specific to `md-gateway`:

* `setup` (Default: `${config:home}/md-gateway.yaml`) — path to the YAML config passed to the Java main class.
* `jar` (Default: `lib/md-gateway.jar`) — jar path relative to the installed package directory.
* `main-class` (Default: `com.itrsgroup.mdgateway.Main`)
* `program` (Default: bundled `jdk/bin/java` under the selected package version)
* `logback` (Default: packaged `config/logback.xml`)
* `audit-log` (Default: `md-gateway-audit.log`) — operational audit log basename in the instance home directory.
* `xms` / `xmx` (Default: `512m`) — JVM heap sizes.
* `java-options` — extra JVM flags, split on spaces.

## Usage

```text
geneos md-gateway
```

### Options

```text
      --allow-root      allow running as root (not recommended)
  -G, --config string   config file (defaults are $HOME/.config/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
