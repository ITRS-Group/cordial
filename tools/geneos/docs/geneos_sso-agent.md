# `geneos sso-agent`

An `sso-agent` component is an instance of the Geneos SSO Agent to provide Single-Sign On authentication to other Geneos components.

The sso-agent requires Java to be installed and by default looks for `/usr/bin/java`. To use a different Java runtime, set the `JAVA_HOME` environment variable for the instance using `geneos set sso-agent -e JAVA_HOME=/path/to/java` before starting the instance. This can also be set during the `add` and `deploy` commands.

If you already have a stand-alone instance of the SSO Agent and want to bring it under the control the `geneos` program you can do something like this:

```bash
$ geneos deploy sso-agent --import /path/to/existing/sso-agent/conf --user email@example.com
Password:
```

This will download and install the latest SSO Agent install package from the ITRS download site using the credentials `email@example.com` and prompting you for your password. The `--import` option pulls in the existing configuration files (including the keystore/trust store) from the path given. Be careful to not end the path in a `/`.

Once this is done you can start the `sso-agent` instance and it should continue working like the original, but under the management of the `geneos` command.

To deploy a new `sso-agent` instance, use the same command but leave out the `--import ...` arguments. Then you have to edit the `conf/sso-agent.conf` file in the instance directory (`geneos ls` to see the paths or `geneos home sso-agent NAME` to get just the specific path) to set the correct listening port and other mandatory parameters. Use `geneos rebuild sso-agent NAME` to regenerate other files after editing `sso-agent.conf`.

>[!NOTE]
>Even though the commands above tells you it has added an instance with a specific port number, the actual listening port is set in the `conf/sso-agent.conf` file and must be updated by hand. `geneos` will not change the actuual component configuration file automatically.

The sso-agent trust store and key store files (normally the same file) are initialised during deployment and if you have configured TLS then the instance certificate and private key are added to the files as well as a freshly created `ssokey` for the process to sign the authentication tokens.

If you change the `ssl_alias` or any of the passwords for the keystore or truststore, then any command that rebuilds the instance configuration will also rebuild the keystore and truststore files.

As documented in the official SSO Agent documentation, you can use environment variables instead of clear-text passwords in the `conf/sso-agent.conf` file. The `geneos` program will use the `ITRS_SSO_KEYSTORE_PW` and `ITRS_SSO_TRUSTSTORE_PW` environment variables if they are set in the instance configuration when building the keystore and truststore files, respectively. You can set AES256 encoded values in the instance configuration using `geneos set sso-agent NAME --secureenv ITRS_SSO_KEYSTORE_PW` (or `ITRS_SSO_TRUSTSTORE_PW`) and be prompted for the password. In this case you must comment out the `password` fields in the `conf/sso-agent.conf` file. Other supported variables are passed to the sso-agent in it's starting environment, but are not specially interrpreted by `geneos`.

## Usage

```text
geneos sso-agent
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
