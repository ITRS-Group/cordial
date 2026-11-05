# `geneos tr-gateway`

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
tr-gateway-idx-1.0.0-20261030.041812-1-linux-x64.tar.gz
```

Each distinct snapshot timestamp is installed as its own version directory. Use `--override tr-gateway:VERSION` only if the filename does not match. These packages are not downloaded from ITRS.

Exchange-specific packages, `tr-gateway-EXCHANGE-VERSION...`, install as a `tr-gateway` release with the exchange appended to the version, e.g. `packages/tr-gateway/1.0.0-20261030.041812-1+idx`, so packages for different exchanges with the same version do not collide. To select one exchange, give the package commands `tr-gateway-EXCHANGE` as the type:

```bash
geneos package install tr-gateway-idx ./downloads/
geneos package install tr-gateway-idx -N -S
geneos package update tr-gateway-idx
```

This installs only `tr-gateway-idx-*` archives from a directory, downloads the `tr-gateway-idx` artifact from Nexus, and updates `active_prod` to the latest `+idx` release. Without an exchange, `tr-gateway` keeps its existing behaviour. Instances are still `tr-gateway` instances.

The `active_prod` link is shared by all exchanges, so on a host with more than one exchange installed, give each instance its own release with `geneos set tr-gateway NAME version=1.0.0-20261030.041812-1+idx` rather than relying on `active_prod`.

Releases before v1.30.0 do not recognise exchange package names. To install one with those releases, rename the archive to put the exchange after the version, e.g. `tr-gateway-1.0.0-20261030.041812-1-idx-linux-x64.tar.gz`. It then installs into the same `...+idx` directory, and later releases accept the renamed file too.

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

Local instances write each audit event as one JSON line to the instance audit log (default `tr-gateway-audit.log`). Remote instances write only to any configured global audit destination. Records contain an `action`, a `command_id`, `username`, nested `instance` details, a `result`, and action-specific fields. Filter with `jq` on the `action` field:

```json
{"timestamp":"2026-08-28T02:35:10Z","action":"import","command_id":"<uuid>","username":"jenkins","instance":{"component":"tr-gateway","name":"prod1","host":"localhost"},"result":"success","source":"./prod1.yaml","file":"tr-gateway.yaml"}
{"timestamp":"2026-08-28T02:40:00Z","action":"start","command_id":"<uuid>","username":"jenkins","instance":{"component":"tr-gateway","name":"prod1","host":"localhost"},"result":"success","commandline":"/path/to/java ...","pid":12345}
{"timestamp":"2026-08-28T02:45:00Z","action":"stop","command_id":"<uuid>","username":"jenkins","instance":{"component":"tr-gateway","name":"prod1","host":"localhost"},"result":"success","pid":12345}
{"timestamp":"2026-08-28T02:46:00Z","action":"restart","command_id":"<uuid>","username":"jenkins","instance":{"component":"tr-gateway","name":"prod1","host":"localhost"},"result":"success","newPid":12346,"oldPid":12345}
{"timestamp":"2026-08-28T03:00:00Z","action":"update","command_id":"<uuid>","username":"jenkins","instance":{"component":"tr-gateway","name":"prod1","host":"localhost"},"result":"success","version":"1.0.0","base":"active_prod"}
{"timestamp":"2026-08-28T03:05:00Z","action":"set","command_id":"<uuid>","username":"jenkins","instance":{"component":"tr-gateway","name":"prod1","host":"localhost"},"result":"success","values":"[xms=512m xmx=1024m]"}
{"timestamp":"2026-08-28T03:10:00Z","action":"add","command_id":"<uuid>","username":"jenkins","instance":{"component":"tr-gateway","name":"prod1","host":"localhost"},"result":"success","port":19000}
```

`geneos logs` follows the Java log (`tr-gateway.log`), not the audit file. Configure the audit filename with `audit.file`.

## Configuration

Instance settings are stored in `tr-gateway.json` in the instance working directory. Use `geneos set` and `geneos unset` rather than editing that file directly.

### Instance Parameters

For general instance parameters see `geneos help set`. Parameters specific to `tr-gateway`:

* `setup` (Default: `${config:home}/tr-gateway.yaml`) — path to the YAML config passed to the Java main class.
* `jar` (Default: `lib/tr-gateway.jar`) — jar path relative to the installed package directory.
* `main-class` (Default: `com.itrsgroup.trgateway.Main`)
* `program` (Default: bundled `jdk/bin/java` under the selected package version)
* `logback` (Default: packaged `config/logback.xml`)
* `audit.file` (Default: `tr-gateway-audit.log`) — audit log path relative to the instance home directory.
* `xms` / `xmx` (Default: `512m`) — JVM heap sizes.
* `java-options` — extra JVM flags, split on spaces.

## Usage

```text
geneos tr-gateway
```

### Options

```text
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
