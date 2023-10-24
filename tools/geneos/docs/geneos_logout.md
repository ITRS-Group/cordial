# `geneos logout`

The logout command removes the credentials for the `DOMAIN` given. If no names are set then the default credentials (`itrsgroup.com`) are removed.

If the `-A` options is given then all credentials are removed, but the underlying file is not deleted.

```text
geneos logout [flags] [DOMAIN...]
```

### Options

```text
  -A, --all   remove all credentials
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
