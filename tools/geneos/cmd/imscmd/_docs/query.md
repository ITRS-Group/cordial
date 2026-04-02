# `geneos incident query`

Query for a list of incidents and their details. This command is used to query the `tools/ims-gateway` program for a list of incidents and their details. The command can be used to filter the list of incidents based on various criteria such as status, priority, assignee, etc. The command can also be used to display the details of a specific incident by providing the incident ID.

The command relies on a configuration file, normally locates in `${HOME}/.config/geneos/ims.yaml`, to provide the connection details for the `ims-gateway` program. If the configuration file is not found or is invalid then an error will be returned. You can specify an alternative configuration file using the `--config`/`-C` option.
