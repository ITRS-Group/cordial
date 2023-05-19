# `geneos restart`

Restart instances

```text
geneos restart [flags] [TYPE] [NAME...]
```

Restart the matching instances.

By default this is identical to running `geneos stop` followed by
`geneos start`.

If the `--all`/`-a` option is given then all matching instances are
started regardless of whether they were stopped by the command.

Protected instances will not be restarted unless the `--force`/`-F`
option is given.

Normal behaviour is to send, on Linux, a `SIGTERM` to the process and
wait for a short period before trying again until the process is no
longer running. If this fails to stop the process a `SIGKILL` is sent to
terminate the process without further action. If the `--kill`/`-K`
option is used then the terminate signal is sent immediately without
waiting. Beware that this can leave instance files corrupted or in an
indeterminate state.

If the `--log`/`-l` option is given then the logs of all instances that
are started are followed until interrupted by the user.

### Options

```text
  -a, --all     Start all matching instances, not just those already running
  -F, --force   Force restart of protected instances
  -K, --kill    Force stop by sending an immediate SIGKILL
  -l, --log     Run 'logs -f' after starting instance(s)
```

### Options inherited from parent commands

```text
  -G, --config string   config file (defaults are $HOME/.config/geneos.json, /etc/geneos/geneos.json)
  -H, --host HOSTNAME   Limit actions to HOSTNAME (not for commands given instance@host parameters)
```

## SEE ALSO

* [geneos](geneos.md)	 - Take control of your Geneos environments
