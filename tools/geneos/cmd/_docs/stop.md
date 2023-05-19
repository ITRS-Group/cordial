Stop the matching instances.

Protected instances will not be restarted unless the `--force`/`-F`
option is given.

Normal behaviour is to send, on Linux, a `SIGTERM` to the process and
wait for a short period before trying again until the process is no
longer running. If this fails to stop the process a SIGKILL is sent to
terminate the process without further action. If the `--kill`/`-K`
option is used then the terminate signal is sent immediately without
waiting. Beware that this can leave instance files corrupted or in an
indeterminate state.
