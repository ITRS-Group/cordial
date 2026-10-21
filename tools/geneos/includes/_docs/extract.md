Extract includes from the geneos binary and place them in a versioned directory under the gateway directory.

Functional include files are extracted under `${GENEOS}/gateway/includes/<version>`. These are then linked back to the parent directory using symlinks to the chosen version. These files are all prefixes `itrs.` and are considered managed by the system.

Configuration templates are extracted under `${GENEOS}/gateway/include/config` but are never overwritten as it is assumed that these are edited by the user. These files are all prefixed `config.`.
