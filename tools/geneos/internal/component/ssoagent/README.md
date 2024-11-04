An `sso-agent` component is an instance of the Geneos SSO Agent to provide Single-Sign On authentication to other Geneos components.

It requires Java to be installed and, unless the default configuration is changed, looks for `/usr/bin/java`.

If you already have a stand-alone instance of the SSO Agent and want to bring it under the control the `geneos` program you can do something like this:

```bash
$ geneos deploy sso-agent --import /path/to/existing/sso-agent/conf --user email@example.com
Password:
```

This will download and install the latest SSO Agent install package from the ITRS download site using the credentials `email@example.com` and prompting you for your password. The `--import` option pulls in the existing configuration files (including the keystore/trust store) from the path given. Be careful to not end the path in a `/` as this copies the contents of a directory without the directory name itself!

Once this is done you can start the `sso-agent` instance and it should work like the stand-alone one.

The deploy a new `sso-agent` instance, use the same command but leave out the `--import ...` arguments. Then you have to edit the `conf/sso-agent.conf` file in the instance directory (`geneos ls` to see the paths or `geneos home sso-agent NAME` to get just the specific path) to set the correct listening port and other mandatory parameters. At this time `geneos` does not manage any of the settings for you, even though the command tells you it has added an instance with a specific port number.

The trust store and key store files (normally the same file) are initialised during deployment and if you have configured `geneos` TLS then the instance certificate and private key are added to the files as well as a freshly created `ssokey` for the process to sign the authentication tokens.
