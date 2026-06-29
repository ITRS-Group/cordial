A `licd` component is an instance of a Geneos License Daemon. There can only be one running `licd` per system.

## Licence File

When creating the `licd` instance you should import a valid licence file. This is normally supplied by the ITRS Support team named as `HOSTID.lic`, where `HOSTID` is the output of the `hostid` command on the system where the `licd` instance will be running. In the general case, the License Daemon would normally be installed on the standby servers for Geneos deployments where there are primary/standby Gateway configured. In larger environments, the `licd` instance may be installed on a dedicated server to manage licences for multiple Geneos deployments.

If you are using `geneos init all` for a new server that will host the `licd` instance, use the `--licence`/`-L` option with the path to the file and it will be copied into the instance directory with the correct `geneos.lic` file name. If you create the `licd` instance with either `geneos deploy` or `geneos add` then you should use the `--import`/`-I` option but you must specify the destination file name like this: `geneos deploy ... --import geneos.lic=/path/to/HOSTID.lic` or `geneos add ... --import geneos.lic=/path/to/HOSTID.lic`. The `geneos.lic` file is the only licence file that the `licd` instance will use.

To update the licence file for an instance, use the `geneos import` command similar to the above but without the flag, e.g. `geneos import geneos.lic=/path/to/HOSTID.lic`. The `geneos.lic` file will be replaced with the new file but the `licd` instance will not be restarted automatically, so you must also `geneos restart licd` to complete to update.

## Configuration

The License Daemon instance configuration is stored in the instance configuration file `licd.json`. This file is created with the instance and updated when the `geneos set` and `geneos unset` commands are used to change parameters. The configuration file is stored in the instance working directory, which is shown when running the `geneos list` command. This file should not be edited directly but instead the `geneos set` and `geneos unset` commands should be used to change the configuration parameters.

Note: In a future `cordial` release the configuration file may move to a YAML format for better readability but the JSON format would continue to be supported for backwards compatibility.

### Instance Parameters

For general instance parameters, applicable to all component types, please see the documentation for the `geneos set` command, i.e. `geneos help set`.

The License Daemon component has no parameters other than those applicable to all components.
