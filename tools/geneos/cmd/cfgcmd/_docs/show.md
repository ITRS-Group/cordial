The show command outputs the current configuration for the `geneos`
program in JSON format. It shows the processed values from the
on-disk copy of your program configuration and not the final
configuration that the running program uses, which includes many
built-in defaults.

If any arguments are given then they are treated as a list of keys to
limit the output to just those keys that match and have a non-nil value.
