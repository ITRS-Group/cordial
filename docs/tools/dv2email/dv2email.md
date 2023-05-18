# dv2email

Email a Dataview following Geneos Action/Effect conventions

```text
dv2email [flags]
```
## Details

Email a Dataview following Geneos Action/Effect conventions.

When called without a sub-command and no arguments the program
processes environment variables setup as per Geneos Action/Effect
conventions and constructs an HTML Email of the dataview the data
item is from.

Settings for the Gateway REST connection and defaults for the EMail
gateway can be located in dv2email.yaml (either in the working
directory or in the user's .config/dv2email directory)
	
### Options

```text
  -f, --config string   config file (default is $HOME/.config/geneos/dv2email.yaml)
  -d, --debug           enable extra debug output
  -i, --inline-css      inline CSS for better mail client support (default true)
```

