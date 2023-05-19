# `geneos` Gateway Component

* Gateway general

The `gateway` component type represents a Geneos Gateway.

* Configuration



* Gateway templates

  When creating a new Gateway instance a default `gateway.setup.xml`
  file is created from the template(s) installed in the
  `gateway/templates` directory. By default this file is only created
  once but can be re-created using the `rebuild` command with the `-F`
  option if required. In turn this can also be protected against by
  setting the Gateway configuration setting `configrebuild` to `never`.

* Gateway variables for templates

  Gateways support the setting of Include files for use in templated
  configurations. These are set similarly to the `-e` parameters:

  ```bash
  geneos gateway set example2 -i  100:/path/to/include
  ```

  The setting value is `priority:path` and path can be a relative or
  absolute path or a URL. In the case of a URL the source is NOT
  downloaded but instead the URL is written as-is in the template
  output.