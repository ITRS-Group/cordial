# ITRS Layer 2 Programs and Logs Configuration

The `itrs.layer2.programs-and-logs.xml` include file provides shared monitoring configuration for programs and their logs within your Geneos Gateway setup. It defines samplers for logs, processes, and Windows services; technology-specific monitoring can be provided by additional Layer 2 include files.

It is intended for Layer 2 of your configuration hierarchy. Layer 3 - Applications include files can build on these shared samplers and settings where appropriate.

The include file contains three pre-defined samplers for common process and log monitoring, with Windows Services monitoring for Windows hosts.

