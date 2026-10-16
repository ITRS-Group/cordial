# `geneos logout`

The `logout` command removes the credentials for the `DOMAIN` given. If no names are set then the default credentials (`itrsgroup.com`) are removed.

If the `-A` options is given then all credentials are removed, but the underlying file is not deleted.

## Usage

```text
geneos logout [flags] [DOMAIN...]
```

### Options

```text
  -A, --all               remove all credentials
      --allow-root        allow running as root (not recommended)
  -G, --config string     config file (defaults are ~/.config/docs/docs.json, /etc/docs/docs.json)
  -H, --host HOSTNAME     Limit actions to HOSTNAME (not for commands given instance@host parameters) (default "all")
      --profiles string   profiles configuration (defaults are ~/.config/docs/profiles.yaml, /etc/docs/profiles.yaml)
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
