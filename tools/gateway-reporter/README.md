# Gateway Reporter Tool

The `geneos-reporter` tool creates ITRS Geneos monitoring coverage reports using Gateway configurations.

This tool works with the Gateway configuration only and does not interact with the running Gateway. This is to mitigate any potential performance impact on production Gateways. The level of information available is limited by the complexities introduced by the dynamically changing monitoring environment in a typical, extensive Geneos deployment.

⚠️ Note: In light of the above, the coverage reports are based on statically configured Netprobes and the Managed Entities attached to them. If you use Self-Announcing Netprobes then they are not included in the reports.

The reports support a limited number of the most common plugins and more can and ill be added over time. If you have specific requirements, please either raise a Github issue under the `cordial` repo or contact the ITRS Professional Services team.


## Getting Started

The tool can run in two modes:

1. As a *[Gateway Validation Hook](https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/geneos_advancedfeatures_tr.html#Gateway_Hooks)*
2. From the command line using the `report` option

In both cases the reporter ONLY works on a merged configuration file create by the Gateway. In the first mode, as a validation hook, the Gateway creates a merged setup file during validation, while for the second mode the user can choose to either read a pre-merged configuration or have the reporter program run a one-off instance of the Gateway to produce a merged file.

### Prerequisites

`geneos-reporter`, like other standalone tools in the `cordial` repo, it is built so that it has no external dependencies when run on a 64-bit Intel/AMD Linux system, the same architecture as the ITRS Geneos Gateway supports.

When run as a validation hook, the Gateway itself prepares a merged setup file before invoking `geneos-reporter` as the validation program. Apart from a suitable YAML configuration file and permissions to write the reports to the configured directory there should be no other pre-requisites.

When run from the command line `geneos-reporter` can either read prepared, merged setup file(s) or it can invoke a one-shot instance of a Gateway to create a merged set-up but in this case it **must** be run on one of the same systems (Primary of Standby) as the Gateway(s) being reported. This is so that the merge process runs in the same environment as the Gateway, for access to include files and Gateway version consistency.

### Installation

The `geneos-reporter` program consists of a single binary, which should have no external dependencies beyond the Gateway being reported on, and an optional configuration file that can customise the operations and output formats.

By default the program is called `geneos-reporter` and the configuration file will use the same name with a `.yaml` suffix. The configuration file can be located in any of the following directories:

* `/etc/itrs`
* `${HOME}/.config/itrs`
* `.` (the current working directory)

If there are configuration files in more than one of the search directories above then they are read in the order above and settings in later files will override the settings from earlier files.

⚠️ Note: If you rename the binary then the configuration file will also need to be renamed to match, e.g. if the tool is renamed `ITRSAuditTool` then the configuration file(s) will only be read if they are called `ITRSAuditTool.yaml`. All names are case sensitive. This naming rule ignores the first file extension (e.g. `.exe` if a Windows binary existed) and any suffix of the form `-VERSION`, where VERSION matches the `cordial` release the program was released with.

The program can be located anywhere, but is normally placed in either a system binary directory (`/usr/local/bin` for example) or your user's own `${HOME}/bin`. Just make sure that the directory is in your `PATH` for execution.

### Running Manually

To run the tool manually you have to use the `report` sub-command with the appropriate options, like this:

```bash
geneos-reporter report /path/to/gateway.setup.xml /path/to/another/gateway.setup.xml
```

There are a number of options as can be seen from the usage message (use `--help` or `-h` to get up-to-date help):

```text
  -o, --out DIRECTORY        Write reports to DIRECTORY. Default `/tmp/gateway-reporter`
  -m, --merge                Create a merged config file. --install must be set
  -i, --install BINARY|DIR   Path to the gateway installation BINARY|DIR
  -p, --prefix name          Report prefix for configurations without a Gateway name
```

In most cases you will not have a pre-merged Gateway configuration file so you can use the options to launch a Gateway to create one for you.

For example:

```bash
$ geneos-reporter report --gateway /opt/itrs/packages/gateway/active_prod/gateway2.linux_64 \
                        --merge /opt/itrs/gateway/gateways/MyGateway/gateway.setup.xml
```

To override a setting without editing the configuration file you can use environment variables for any setting if you use the format `GENEOS_SETTING` where `SETTING` is the upper-case configuration key with levels indicated with underscores. e.g. to update `output.directory` use:

```bash
$ export GENEOS_OUTPUT_DIRECTORY=${HOME}/reports
$ geneos-reporter report --gateway /opt/itrs/packages/gateway/active_prod/gateway2.linux_64 \
                        --merge /opt/itrs/gateway/gateways/MyGateway/gateway.setup.xml
```

When run like this you can pass any number of setups on the command line and each will be processed (and merged, if selected). Using the default configuration each input setup will result in one or more report files with the Gateway name as found in the Operating Environment used as a file prefix.

### Running as a Gateway Validation Hook

Running as a [Gateway Validation Hook](https://docs.itrsgroup.com/docs/geneos/current/Gateway_Reference_Guide/geneos_advancedfeatures_tr.html#Gateway_Hooks) allows you to control reporting through your Geneos Gateways using the commands you are used to and produce reports with the addition of one or two command line options.

To do this you must either wrap the `geneos-reporter` binary in a shell script, rename the program or (the suggested method) use a symbolic link so that the Gateway sees a hook file called `validate-setup` in a directory you choose. For example:

```bash
cd ${HOME}/bin
ln -s geneos-reporter validate-setup
```

Then run a copy of the Gateway with most of the same options as normal but remove `-log FILENAME` and add the following to the command line:

```bash
... -nolog -silent -hooks-dir /path/to/hooks -hub-validation-rules -validate
```

The first two options make the Gateway start-up quieter and also avoid overwriting the normal log file. You should also remember to remove the `-log FILENAME` option, otherwise you will get an error.

The next option "`-hooks-dir /path/to/hooks`" should be set to point to the directory that the `validate-setup` link is in (this could be your `${HOME}/bin`, for example)

The other to options tell the Gateway not to open a listening port and also to ignore some other settings which are not required and also the invoke the `validate-setup` hook with the right `_VALIDATE_TYPE` setting.

Note: If you use the `cordial` Geneos tool `geneos` to manage your Geneos environment then you can do the above using:

```bash
geneos show --validate gateway GATEWAYNAME
```

Note: The above **only** works if you are on `cordial` version v1.6.5 or later.

## Results

When the program finished running you should have one or more output files in the configured directory. By default four kinds files are created:

1. An XLSX file with multiple sheets

    The report produced has a number of sheets, each with different views of the configured environment for the Gateway. The core monitored items for a selected range of plugins are included, for example the file names for FKM plugins and the process "aliases" for the Processes plugin.

2. A ZIP file containing a CSV file per report

    Each file in the ZIP corresponds to a sheet in the above XLSX file.

3. A JSON file which is the parsed data in machine readable format.

    The structure is not defined at this time but should be straight forward to infer from inspection.

4. An XML file which is the merged Gateway setup file used for the report

    This merged configuration is intended for diagnostics but can also be used for further processing or manual inspection to clarify any monitoring coverage identified in other reports above.

The reporting tool only has access to the Gateway configuration through a merged set-up file. It has no access to live, dynamic state of the Gateway and so is limited in what kind of data it can analyse. While the Gateway configuration is an XML file with a published schema (`gateway.xsd`) file this is only the syntactic part of the configuration. The semantics of the configuration are normally only driven by the Gateway and any reporting is subject to attempting to duplicate the same evaluation process on the setup file.

## Configuration Reference

The optional configuration file allows you to tune the processing and output. The configuration file is in YAML format and supports values expansion using a shell-like syntax but with special operators. Some special values are available and are documented below.

### Settings

The following settings are available, along with their default values:

* `site` - Default `ITRS`

  This setting is used in the output report to identify the site the report is generated for. It can also be set using the environment variable `GENEOS_SITE`.

* `output`

  The `output` section controls the location and types of files as well the filtering of columns in reports.

  * `directory` - Default `/tmp/gateway-reporter`

    This is the default directory where reports are written. It is created if it does not exist (and permissions allow). Like other settings, this can also be set as an environment variable `GENEOS_OUTPUT_DIRECTORY`

  * `skip-empty-reports` - Default `true`

    If any reports in either CSV or XLSX files would result in no rows then do not include the file or sheet, respectively.

  * `formats`

    The `formats` section controls which report files are created and the filename, under the `output.directory`, to create. To disable a report use an empty filename (e.g. `""`). If a format is an absolute path then the `directory` value is ignored.

    * Special Values

      The following special values are available in the `formats` section settings:

      * `${gateway}` - The name of the Gateway the report is running against. This is taken from the Operating Environment section of the merged configuration file(s) being processed.

      * `${datetime}` - The date and time in UTC in a simple `YYYYMMDDHHmmss` format, intended for timestamping filenames. The time is when each file is being written to and there is a chance that the timestamp may differ between files in the same report run. This should be addressed in a future release to only use one time per run.

      * Other expansion options are available and are documented here: <https://pkg.go.dev/github.com/itrs-group/cordial/pkg/config#section-readme>

    * `xlsx` - Default `${gateway}-Report-${datetime}.xlsx`

      The `xlsx` file format is an Excel compatible workbook containing a summary sheet and one sheet per report (see below).

    * `csv` - Default `${gateway}-Report-${datetime}.zip`

      The `csv` file format is created as a ZIP file containing one CSV files per report (see below).

    * `json` - Default `${gateway}-Report-${datetime}.json`

      The `json` file format is created for machine processing by downstream systems. If not required it should be disabled in normal use by setting the filename to `""`.

    * `xml` - Default `${gateway}-Merged-${datetime}.xml`

      The `xml` output format is intended for audit and debugging. The file is the merged set-up file that the rest of the report is based on. It can be used as the final arbiter of why something is (or is not) in the report, could be used to re-run the report manually after changing configurations and also for passing to the developers for debugging any potential issues.

  * `plugins`

    The `plugins` section allows control of which supported plugins are considered in-scope for reports. The two configuration items, `single-column` and `two-column` must be given as YAML lists.

    ⚠️ Note that the names used must match those in the Gateway configuration for each plugin and must not be changed from the name given, which is case-sensitive.

    All reports include the Managed Entity, the Type and the Sampler names as the first three columns.

    * `single-column`

      Single-column plugins are those that report one type of data item, such as a filename or process. Some of these are aggregated single data items (see below). The current supported list of plugins is:

      * `control-m`

        The `control-m` report lists the dataviews and the criteria for jobs shown. Each entry is displayed as:

        ```text
        Dataview : Parameter : Criteria
        ```

      * `disk`

        The `disk` reports lists the disk partitions being monitored. For samplers that have Auto-Detect enabled an entry of `[AUTODETECT]` is shown. For samplers that have the Check NFS Partitions set to false an entry if `[NO_NFS]` is shown. Excluded partitions are prefixed with `[x]`. If an Alias is set for a partition then it is shown in parenthesis after the path.

        Variables are shown as they appear in the configuration and are not evaluated.

      * `fkm`

        The `fkm` report lists all the files being monitored for each sampler. The report includes streams with a `stream:` prefix and NT Event Logs with the `NTEVentLog:` prefix. Dynamic Files are not supported.

        File sources from both direct configuration and from Process Descriptors are included.

        Variables and date formats are shown as they appear in the configuration and are not evaluated.

      * `ftm`

        The `ftm` report lists all files being monitored. This includes files given under Additional Paths.

        Variables and date formats are shown as they appear in the configuration and are not evaluated.

      * `gateway-sql`

        The `gateway-sql` report lists the names of the queries for each Gateway-SQL sampler.

        Variables are shown as they appear in the configuration and are not evaluated.

      * `processes`

        The `processes` report lists all the process aliases being monitored. The report includes the data from Process Descriptors as well as those directly configured in the sampler. If the process is configured to use a pattern match then only the alias is shown.

        Variables are shown as they appear in the configuration and are not evaluated.

      * `stateTracker`

        The `stateTracker` report lists all the (non-dynamic) trackers and the files they monitor in the form:

        ```text
        TRACKER-GROUP : TRACKER : FILE
        ```

        Trackers with a ` [*]` suffix indicate a configuration that uses a deprecated format for the tracker name.

        Variables and date formats are shown as they appear in the configuration and are not evaluated.

      * `toolkit`

        The `toolkit` report lists the "Sampler script" (which can include command line arguments) configured for each Toolkit sampler.

        Variables are shown as they appear in the configuration and are not evaluated.

      * `win-services`

        The `win-services` report lists the monitored Windows Services. If there are no filters it reports `[ALL]`. For each defined filter the report lists either both the description and the name in the format `Description [Name]` or, if only one or the other is set then just the setting as is.

        If the Service Name has the `Show Unavailable` checkbox selected then the name is suffixed with a ` *`.

        If a regular expression is used for either the description or name then it is displayed in the form `/pattern/flags`.

      * `x-ping`

        The `x-ping` report lists all the targets being monitored by the X-Ping samplers. IP addresses are rendered as human readable text.

        Variables are shown as they appear in the configuration and are not evaluated.

    * `two-column`

      Two-column plugins are those that report two, possibly unrelated, types of data item, e.g. local and remote ports for `tcp-links`.

      * `sql-toolkit`

        The `sql-toolkit` report shows details of the database connection in the first data column and the name of each query (dataview) and the text of the query itself, up to 32k characters, in the second column. The query has leading and trailing whitespace removed but is otherwise left unformatted.

        Variables are shown as they appear in the configuration and are not evaluated.

      * `tcp-links`

        The `tcp-links` report lists all the local and report port filters for each sampler in the first and second column respectively. As there is no link between the order of the local and report ports - they are all applied - they should not be considered to be aligned in anyway.

        Variables are shown as they appear in the configuration and are not evaluated.

  * `reports`

    The `reports` section gives you some control over the details in each report. There should be one entry under reports for each plugin listed above and also the special report `entities`.

    For each report the basic options are:

    * `filename`

      The `filename` is used as the basename for each file in the CSV output format. The extension `.csv` should not be included. This value is case-sensitive.

    * `sheetname`

      The `sheetname` is used to create a new sheet in the workbook for the XLSX output format. This value is case-sensitive.

    * `columns`

      This is an list of column names to use for each report. They do not control which columns are included, but change the text heading. Any heading with special characters or spaces should be quoted using YAML rules.

    The reports available, and their specific options, are: 

    * `entities`

      This report shows all the configured (and non-disabled) Manage Entities. Two sets of data are shown:

      * `attributes`

        An array of Geneos Attribute names to show. If not set then all Attributes are listed. Attributes include those found through containing Managed Entity Groups and `geneos-reporter` tries to match the rules used by the Gateway for precedence and inheritance.

      * `plugins`

        An array of plugins to list in the entities report. If not given then all plugins found in the configuration will be included. For each plugin found the total number configured for each entity is shown. If there are none then the value will be empty and not zero.

    * Plugins

      For each plugin type listed in the `plugins` section above there can be a `reports` section. At the time of writing the only options supported are the common ones listed above.

### Default Configuration

This is the default configuration built into the program:

```yaml
site: ITRS

output:
  directory: /tmp/gateway-reporter

  # if true and there are no entries for any of the single-column or
  # two-column reports then do not create sheets or CSV files for them
  skip-empty-reports: true

  # The names for the different report formats. Set to an empty string
  # to disable - "". If an absolute path then output directory above is
  # ignored. To put each Gateway into it's own directory use a relative
  # path and repeat `${gateway}`
  #
  # `${gateway}` and `${datetime}` are replaced with the Gateway name
  # and the time of the report (`YYYYMMDDhhmmss`) respectively.
  formats:
    xlsx: ${gateway}-Report-${datetime}.xlsx
    csv: ${gateway}-Report-${datetime}.zip
    json: ${gateway}-Report-${datetime}.json
    xml: ${gateway}-Merged-${datetime}.xml

  # plugins filters which plugins to report on and whether they are one
  # or two columns data types
  #
  # for xlsx output the order here controls the order of the sheets in
  # the workbook
  plugins:
    single-column:
      [
        control-m,
        disk,
        fkm,
        ftm,
        gateway-sql,
        processes,
        stateTracker,
        toolkit,
        win-services,
        x-ping,
      ]
    two-column: 
      [
        sql-toolkit,
        tcp-links,
      ]

  reports:
    # each report is a self-contained section for the output file type
    # selected. they all share common settings:
    #
    # * filename - the base filename (without extension) if a file is
    #   generated
    # * sheetname - the sheet name for XLSX output
    # * columns - an ordered array of column names to use for the output
    #   data
    #
    # The summary report is intended as a header page of information
    # about the report generation, including the date, program version,
    # gateway name and so on.
    summary:
      filename: summary
      sheetname: Summary
    entities:
      filename: entities
      sheetname: Entities
      columns: [ "Managed Entity", "Netprobe Name", "Hostname", "Port" ]
      # attributes: [ENVIRONMENT, DATACENTER, OS]
      # plugins: [fkm, toolkit, x-ping]

    control-m:
      filename: control-m
      sheetname: Control-M
      columns: [ "Managed Entity", Type, Sampler, "View : Parameter : Criteria" ]
    disk:
      filename: disks
      sheetname: Disks
      columns: [ "Managed Entity", Type, Sampler, "Disk (x=eXclude)" ]
    fkm:
      filename: fkm
      sheetname: FKM
      columns: [ "Managed Entity", Type, Sampler, "File / Source" ]
    ftm:
      filename: ftm
      sheetname: FTM
      columns: [ "Managed Entity", Type, Sampler, "File" ]
    gateway-sql:
      filename: gateway-sql
      sheetname: Gateway SQL
      columns: [ "Managed Entity", Type, Sampler, "Query Name" ]
    processes:
      filename: processes
      sheetname: Processes
      columns: [ "Managed Entity", Type, Sampler, "Process Alias" ]
    stateTracker:
      filename: statetracker
      sheetname: State Tracker
      columns: [ "Managed Entity", Type, Sampler, "Group : Tracker : File" ]
    toolkit:
      filename: toolkit
      sheetname: Toolkits
      columns: [ "Managed Entity", Type, Sampler, "Toolkit Script" ]
    win-services:
      filename: win-services
      sheetname: Windows Services
      columns: [ "Managed Entity", Type, Sampler, "Service Description [Service Name]" ]
    x-ping:
      filename: x-ping
      sheetname: X-Ping
      columns: [ "Managed Entity", Type, Sampler, "Remote Target" ]

    sql-toolkit:
      filename: sql-toolkit
      sheetname: SQL Toolkit
      columns: [ "Managed Entity", Type, Sampler, "Database", "Query Name" ]
    tcp-links:
      filename: tcp-links
      sheetname: TCP Links
      columns: [ "Managed Entity", Type, Sampler, "Local Port", "Remote Port" ]
```