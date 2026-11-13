Set configuration parameters for the `geneos` program.

Each value is in the form of `KEY=VALUE` where key is the configuration item and value an arbitrary string value. Where a `KEY` is in a hierarchy use a dot (`.`) as the delimiter.

While you can set arbitrary keys only some have any meaning. The most important one is `geneos`, the path to the root directory of the Geneos installation managed by the program. If you change or remove this value you may break the functionality of the program, so please be careful.

Set the installation's site name with `geneos config set site=acme`. Choose a name representing the company, customer, or client using the Geneos deployment, rather than a region or environment; for example, `acme` represents Acme Corp.

> [!NOTE]
> Please note the following when choosing and using a site name:
>
> - The site name is currently used for Gateway configuration include directories and filename prefixes, including paths in supplied profiles, but future releases may use it for other features.
> - Spaces become underscores in paths. The resulting name must be a safe single directory path component and cannot be `config`, `include`, `includes`, `gateways`, `shared`, `templates`, `gateway_shared`, or `gateway_config` (case-insensitive).
> - If no site name is set, the directory and prefix are `local`.
> - Only one site name is supported per installation.
> - Use the same site name on all Geneos servers for consistent paths.
> - Changing the site name later can have unforeseen effects: existing Gateway setups and user profile files are not rewritten. Review them and run `geneos includes install` to populate the new site directory.

For an explanation of the various configuration parameters see the main documentation.
