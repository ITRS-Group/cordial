# Geneos Incident Commands

The `geneos` program supports raising incidents in an Incident Management System (IMS) through the `geneos incident` commands. These commands allow you to send data to the IMS Gateway, which can then process this data and create or update incidents in the target IMS platform based on the configuration and the special fields provided in the data.

The `geneos incident` commands are controlled though a standalone `ims.yaml` configuration file, which allows you to specify the target IMS platform and the mapping of the special fields to the fields in the target system. This configuration file is separate from the main Geneos configuration and is used specifically for controlling the behaviour of the incident creation and update process when using the `geneos incident` commands.

