# Profiles

Use a profile to deploy and configure a named group of Geneos instances. A profile can create instances, update existing instances, and build Self-Announcing Netprobe (SAN) managed entities from the components installed on a host.

The profile definition is YAML. The installed `profile` instance stores deployment options. It is not a monitoring process.

## Select the Profile File

On Linux, the normal user file is `~/.config/geneos/profiles.yaml`. If `XDG_CONFIG_HOME` is set, the user configuration directory follows that setting. The system configuration location is `/etc/geneos/profiles.yaml`. The command does not search the current directory for this file.

If no profile file is found, `geneos` uses embedded defaults. During initialisation, it writes the defaults to the user configuration directory if the file does not exist. It does not overwrite an existing file.

To select another file, use the global `--profiles` option:

```bash
geneos --profiles ~/config/profiles.yaml profile list
```

Use the same option when you deploy or rebuild that profile. The installed profile instance stores options, not a copy of the YAML definition or its source path.

The supplied configuration is [internal/profile/profiles.defaults.yaml](../../profile/profiles.defaults.yaml). Use it as an example. Put installation-specific changes in your selected profile file, not in the source file.

## Use a Supplied Profile

### Inspect the Definitions

1. List the profiles in the selected file:

   ```bash
   geneos profile list
   ```

    The list includes each profile's description and its source file. Use `--format` (`-f`) to choose `column` (the default), `table`, `csv`, `json`, `pretty` (indented JSON), or `toolkit` output.

2. Read the help for a named profile:

   ```bash
   geneos profile --help motm
   ```

3. Check its definition in your profile file. In particular, check ports, include paths, Gateway Types, and certificate requirements before deployment.

| Profile | Components and purpose |
| --- | --- |
| `all` | A Gateway and Webserver named for the host, a Netprobe named `localhost`, and a License Daemon named `perm`. The License Daemon uses port `7041`. |
| `demo` | A Gateway named `Demo Gateway` with `options=-demo`, a Netprobe, and a Webserver named `Demo Webserver`. The Gateway has best-practice includes. The Webserver port is `8443`, or `8080` when `--insecure` is selected. |
| `motm` | A SAN for Monitor Of The Monitors (MOTM). It requests host monitoring and creates managed entities for installed Gateways, Netprobes, minimal Netprobes, File Agents, License Daemons, and Webservers. |
| `motm-gateway` | A host-named MOTM Gateway with best-practice includes, and a Netprobe. It is the receiving side of an MOTM installation. |

`all` is a starting example, not an inherited base profile. Profiles do not inherit settings from one another.

### Initialise a Demo Installation

Use `init` for a new installation. You need access to suitable release archives or an ITRS download account. The include files referenced by `demo` must be available at the configured paths.

1. Initialise and deploy the profile. Replace the account and absolute directory path:

   ```bash
   geneos init demo /opt/itrs/geneos --username user@example.com
   ```

   Enter the download password when prompted. Use `--archive /opt/itrs/releases` to use previously downloaded release archives. With `init`, `--minimal` selects the minimal Netprobe package for the supplied profiles.

2. Check the installed instances and their status:

   ```bash
   geneos list
   geneos ps
   ```

3. Check generated configuration and logs before you use the installation for monitoring.

`init` requests start-up for the profile instance. Do not assume that its monitoring processes are running: check `geneos ps` and use `geneos start profile demo` if required. For an existing installation, use `geneos deploy profile NAME` to deploy a profile, then use `geneos start profile NAME` to start its instances. See the [init command reference](../../../docs/geneos_init.md) and [deploy command reference](../../../docs/geneos_deploy.md) for installation and download options.

### Deploy the MOTM SAN

Use this procedure on an existing Geneos host. The receiving Gateway must have the Types requested by the profile. For TLS, both sides must trust the certificate chain.

1. Check the Gateway address and the Types in the `motm` definition.
2. Deploy the profile with an issued certificate bundle. The bundle must contain the SAN's certificate, private key, and certificate chain:

   ```bash
   geneos deploy profile motm \
     --gateway motm-a.example.com:7038 \
     --gateway motm-b.example.com:7038 \
     --attribute ENVIRONMENT=Production \
    --certs-bundle /opt/itrs/certs/motm-san.pem
   ```

   Repeat `--gateway` and `--attribute` as required. For a password-protected PKCS#12 bundle, supply the password as described in the [deploy command reference](../../../docs/geneos_deploy.md). Do not put clear-text passwords in the profile file.

3. Start the instances in the profile:

   ```bash
   geneos start profile motm
   ```

4. Confirm that its managed entities appear on the receiving Gateway.

After you add or remove a monitored component, [rebuild the profile](#apply-changes) to update the managed entities.

## Create a Profile

The top-level `profiles` key maps unique profile names to definitions. Each definition contains metadata and component entries.

Use canonical component names where possible. The profile application handlers support `gateway`, `licd`, `netprobe`, `minimal`, `webserver`, `san`, and `floating`. Registered aliases, such as `gateways` and `netprobes`, are also accepted. Do not assume that every component supported by `geneos` has a profile application handler.

Use a YAML list for component instances. A single mapping is also accepted, as shown by the supplied `demo` and `motm` profiles. The list form makes it easier to add another instance.

1. Add this definition to the `profiles` mapping in your selected file. Do not add a second top-level `profiles` key:

   ```yaml
   profiles:
     application:
       _short: Application monitoring
       _description: |
         Deploy an application Gateway and Netprobe.
       gateway:
         - name: application-gateway
           params:
             - gatewayname=Application Gateway
             - port=7100
       netprobe:
         - name: application-probe
           params:
             - port=7105
   ```

2. Confirm that `geneos` can read the file:

   ```bash
   geneos --profiles ~/config/profiles.yaml profile list
   ```

   This lists definitions. It does not validate include files, ports, Types, or certificates.

3. Deploy the profile into your configured installation:

   ```bash
   geneos --profiles ~/config/profiles.yaml deploy profile application
   ```

4. Inspect the resulting configuration:

   ```bash
   geneos show gateway application-gateway
   geneos show netprobe application-probe
   ```

5. Start the profile's instances when the configuration is correct:

   ```bash
   geneos --profiles ~/config/profiles.yaml start profile application
   ```

Choose ports that are free on the host. The Gateway's internal name, `gatewayname`, can differ from the instance name used in commands.

## Common Configuration

### Profile Metadata

| Key | Meaning |
| --- | --- |
| `_short` | Description shown by `geneos profile list`. |
| `_description` | Markdown description shown by `geneos profile --help NAME`. |

Keys that start with `_` are not treated as component entries. The top-level `version` in the supplied file identifies that configuration example; it does not select the software release to install. Use deployment options for release selection.

### Instance Keys

| Key | Format and use |
| --- | --- |
| `name` | Required, non-empty instance name. Supports expansion. Use a unique name within the component type. A package prefix, such as `minimal:probe1`, selects an alternative package where supported. |
| `env` | List of `NAME=VALUE` strings. Sets the process environment for the instance. |
| `params` | List of `KEY=VALUE` strings. Sets instance configuration parameters, such as `port`, `gatewayname`, or `logfile`. The available parameters depend on the component. |
| `imports` | List of `[DEST=]PATH\|URL` strings. Imports files into the instance directory. `DEST` must be relative to, and below, that directory. Without `DEST`, the source file name is used. |
| `cert-bundle` | Certificate bundle source: a file path or PEM data. Used to import instance certificates. |
| `cert-bundle-password` | Password for a password-protected certificate bundle. Supports cordial secret expansion. |

Example instance entry:

```yaml
netprobe:
  - name: application-probe
    env:
      - JAVA_HOME=/usr/lib/jvm/java-17
    params:
      - port=7105
    imports:
      - application.conf=/opt/itrs/config/application.conf
```

For component parameters, see the [show command reference](../../../docs/geneos_show.md) and [set command reference](../../../docs/geneos_set.md). Profiles record their name in each applied instance's `config.profile` setting.

## Value Expansion

Expansion replaces `${...}` text when the profile is applied. Use it in instance names, common string values, SAN Gateway lists, SAN Types and attributes, and SAN variable values. Do not assume that every YAML key is expanded in the same context.

| Form | Result |
| --- | --- |
| `${hostname}` | Host name supplied by the profile lookup table. |
| `${env:NAME}` | Environment variable from the `geneos` process. An unset variable produces an empty value. |
| `${option:EXPRESSION}` | Result of a `goval` expression using selected command options and `NAME=VALUE` arguments. |
| `${config:KEY}` | Value from the current configuration. In a SAN `for-each` entry, this is the matched instance's configuration. |
| `${expr:EXPRESSION}` | Result of a `goval` expression using current configuration values. Process environment variables are available through the `env` map. |
| `${select:KEY1:KEY2:DEFAULT}` | First non-empty string configuration value, or the final literal default. Use an empty final field to return an empty string. |
| `${replace:KEY1:KEY2:/PATTERN/TEXT/}` | Regular-expression replacement on the first non-empty string configuration value. The final delimiter must match the other delimiters. A delimiter other than `/` is allowed, but not `:`. |

Examples from the supplied profiles:

```yaml
name: ${option:minimal?"minimal:":""}localhost
params:
  - port=${option:insecure?8080:8443}
```

Examples within a SAN `for-each` entry:

```yaml
name: ${replace:gatewayname:name:/ /_/} Gateway
variables:
  - name: logfile
    type: string
    value: ${config:home}/${config:logfile}
```

The option names supplied from flags are `attribute`, `base`, `env`, `gateway`, `include`, `insecure`, `minimal`, `type`, and `variable`, where the command defines them. Flag defaults are included. This is not a list of flags accepted by every command. In particular, `--minimal` is an `init` option, not a `deploy` option.

Additional command arguments in `NAME=VALUE` form can supply option values. For example, a profile can use `name: ${option:site}` and be deployed with:

```bash
geneos deploy profile application site=application-gateway
```

These arguments are saved with the profile options. They are not automatically applied as parameters to every component. Likewise, use `${option:gateway}` or `${option:attribute}` in a definition when you want those command options to control its SAN configuration.

Use `${option:...}`, not `${flag:...}`. Arbitrary flags are not available to option expressions. In the current code, `licence` is not in the included flag list, although the supplied `all` profile uses `${option:licence}`. Do not rely on `init --licence` alone to populate that import; set an explicit import source in your profile.

External lookup expansion is disabled: `${file:...}` and `${https://...}` do not read external content. This does not prevent `imports` from using a path or URL. Expansion errors can produce empty values; inspect the applied configuration before start-up.

## Gateway Configuration

Gateway entries accept the common keys, plus `includes` and `variables`.

### Includes

`includes` maps a unique integer priority to an include path. To place an include in groups, put the group names before the path and separate each field with `|`:

```yaml
gateway:
  - name: application-gateway
    includes:
      500: Configuration | /opt/itrs/includes/application-config.xml
      1000: Applications | Production | /opt/itrs/includes/application.xml
```

Lower numbers have higher priority. The main setup file normally has priority `1`. Spaces around `|` are removed. Includes with the same group path are placed in those groups by the Gateway template.

An include entry references a file; it does not import that file. Use `imports` if you need to copy a file into the instance directory. Relative paths are interpreted by the Gateway in its runtime setup context, not relative to the profile YAML file. Check the generated setup file and make sure the paths are accessible.

### Variables

`variables` defines Operating Environment variables in the generated Gateway runtime setup file:

```yaml
variables:
  - name: environment
    type: string
    value: Production
  - name: retryCount
    type: integer
    value: "3"
```

Each item has `name`, `type`, and `value`. If `type` is omitted, it defaults to `string`. Use string values in YAML, including quoted numbers and booleans.

The variable types are `string`, `integer`, `double`, `boolean`, `activeTime`, `externalConfigFile`, and `secret`. A `secret` requires an instance AES key file. It is encrypted and stored as a string variable.

### Gateway Name and Licensing

Use `params` for these settings:

```yaml
params:
  - gatewayname=Application Gateway
  - licdhost=licence.example.com
  - licdport=7041
  - licdsecure=true
```

The profile structure also declares `gateway-name`, `licd-host`, `licd-port`, and `licd-secure`. The current application handler does not apply those fields. They are not equivalent to the parameters above in this implementation.

## SAN Configuration

A `san` entry accepts common keys and the following keys:

| Key | Format and use |
| --- | --- |
| `gateways` | List of `HOST:PORT` strings. `${option:gateway}` accepts repeated `--gateway` values. |
| `types` | List of Gateway Type names for the primary managed entity. The Types must exist on the receiving Gateway. |
| `attributes` | List of `NAME=VALUE` strings for the primary managed entity. Later values replace earlier values with the same name. |
| `variables` | List of `name`, `type`, and `value` mappings. Uses the [variable format](#variables) described above. |
| `managed-entities` | List of additional managed entity definitions. Each can be fixed or generated from installed instances. |

The primary entity uses the SAN's configured `sanname`. The default template emits an entity only if it has at least one Type. Attributes and variables alone do not create an entity.

With no configured Gateways, the default template connects to `localhost:7038` for TLS, or `localhost:7039` without TLS. Certificate and TLS settings determine whether connections are secure; the port number alone does not enable TLS.

### Additional Managed Entities

| Key | Format and use |
| --- | --- |
| `name` | Name of the additional managed entity. Keep generated names unique. |
| `for-each` | Component type whose installed local instances supply the entity configuration. Omit it for a fixed entity. |
| `match` | List of instance-name glob patterns. With `for-each`, the default is `*`. An instance that matches any pattern is selected. |
| `ignore` | List of instance-name glob patterns applied after `match`. A matching instance is excluded. |
| `types`, `attributes`, `variables` | Same formats as the primary entity. Under `for-each`, expansion uses each matched instance's configuration. |

Patterns use glob syntax, not regular expressions. For example, `prod-*` selects names that start with `prod-`; `?` matches one character. Define explicit values for fixed entities. Use `for-each` when values must come from installed component configuration.

### Generate Entities for Selected Gateways

Add this SAN to a profile. It creates one additional entity for each Gateway whose name starts with `prod-`, except names that start with `prod-test`:

```yaml
san:
  - name: ${hostname}-application-san
    gateways:
      - ${option:gateway}
    types:
      - Level 1 - Infrastructure
    attributes:
      - ${option:attribute}
    managed-entities:
      - for-each: gateway
        match:
          - prod-*
        ignore:
          - prod-test*
        name: ${config:name} Gateway
        types:
          - Level 2 - Geneos Gateway
        attributes:
          - ${option:attribute}
          - COMPONENT=Gateway
          - GATEWAY=${config:name}
        variables:
          - name: gatewayName
            type: string
            value: ${config:gatewayname}
          - name: logfile
            type: string
            value: ${config:home}/${config:logfile}
```

1. Install the Gateway instances to monitor before you apply this profile.
2. Deploy the profile with `--gateway HOST:PORT` and any required `--attribute` options.
3. Inspect the SAN setup file. Confirm the selected names, Types, and file paths.
4. Start or restart the named profile and check the receiving Gateway.

`for-each` examines installed instances when the profile is applied. It does not detect later installation changes automatically.

## Floating Netprobe Configuration

A `floating` entry accepts common keys and `gateways`. It does not accept the SAN-specific `types`, `attributes`, `variables`, or `managed-entities` keys.

```yaml
floating:
  - name: floating-probe
    gateways:
      - ${option:gateway}
```

Supply repeated `--gateway HOST:PORT` options at deployment. The generated configuration uses the instance's TLS settings. Configure the corresponding Floating Netprobe on the receiving Gateways.

## Apply Changes

A rebuild reads the selected YAML definition again and reuses the options saved with the installed profile instance. It applies the definition to its component instances. It does not download software releases.

1. Edit the profile in the same file used for deployment.
2. Apply the definition again:

   ```bash
   geneos --profiles ~/config/profiles.yaml rebuild profile application
   ```

3. Check the affected instances with `geneos show`. For a SAN, check the generated managed entities.
4. If a Gateway's generated files must change, rebuild that Gateway explicitly:

   ```bash
   geneos rebuild gateway application-gateway --force
   ```

5. Restart the profile's instances if required:

   ```bash
   geneos --profiles ~/config/profiles.yaml restart profile application
   ```

The component's `config.rebuild` setting controls generated files. The normal Gateway setting is `initial`; SAN and Floating Netprobe use `always`. `--force` permits a rebuild for `initial`, but not for `never`. A profile rebuild does not automatically reload or restart its child processes. See the [rebuild command reference](../../../docs/geneos_rebuild.md).

To change a saved option, set it on the profile instance before rebuilding. For example:

```bash
geneos set profile motm config.options.gateway=motm-c.example.com:7038
geneos rebuild profile motm
```

Use `geneos show profile motm` to inspect the saved options. Passing new SAN flags to `rebuild` is not a substitute for changing those saved values.

Profile application updates the first matching instance of each component and name, or creates it if absent. On existing instances, it clears selected settings before it applies the new values:

| Component | Settings cleared before application |
| --- | --- |
| Gateway | `licdsecure`, `licdport`, includes, variables, and environment. |
| SAN | Gateways, attributes, Types, variables, environment, and additional managed entities. |
| Floating Netprobe | Gateways and environment. |
| Common-only handlers | No predefined clearing list. |

Thus, an omitted list can remove existing settings. Conversely, removing an arbitrary parameter from `params` does not necessarily remove its old value. Back up configuration before you apply a profile to existing instances. An instance with `protected=true` rejects profile application.

## Limits and Fault Isolation

| Symptom or requirement | Check or action |
| --- | --- |
| Profile is not found | Check its name and the selected `--profiles` file. `profile list` lists available definitions; `geneos list profile` lists installed profile instances. |
| Expanded value is empty | Check the option name, process environment, or matched instance parameter. Option-expression errors can produce empty strings. |
| Include file is not found | Check the path from the Gateway runtime setup context. An `includes` entry does not copy a file. |
| SAN entity is absent | Check that it has Types. For generated entities, also check installed instance names, `match`, and `ignore`. |
| A newly installed component is absent from MOTM | Rebuild the `motm` profile, then use `geneos restart profile motm` if required. |
| Gateway setup file does not change | Check `config.rebuild`; explicitly rebuild the Gateway with `--force` if its setting is `initial`. |
| TLS connection fails | Check the trust chain and TLS settings at both ends. Use an instance certificate bundle from the intended trust chain. |

Current implementation limits:

- Use profile application on the local host. Although deployment accepts host selection, the application handlers and SAN `for-each` processing use `geneos.LOCAL`. Do not rely on remote profile deployment to configure the selected host correctly.
- Component entries are processed without a defined dependency order. Install components before a SAN profile that discovers them, then rebuild that profile after installation changes.
- `deploy --signing-bundle` is used when it initialises a local installation. It is not passed through to profile deployment in an existing installation. Use `--certs-bundle` for issued instance certificates there.
- Application is not transactional. An error can leave some instances updated and others unchanged. Check the result before retrying.
- Removing an instance from the YAML does not delete the installed instance. Renaming an entry can create a new instance and leave the old one in place.
- `geneos profile extract` and its `update` alias are registered but do not currently export or update configuration. Use the [supplied YAML source](../../profile/profiles.defaults.yaml) as the reference for default definitions.