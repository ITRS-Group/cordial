# `geneos config unset`

Unset a program parameter

```text
geneos config unset [KEY...] [flags]
```

Unset removes the program configuration value for any arguments given on
the command line.

No validation is done and there if you mistype a key name it is still
considered valid to remove an non-existing key.

## SEE ALSO

* [geneos config](geneos_config.md)	 - Configure Command Behaviour
