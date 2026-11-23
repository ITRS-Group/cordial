# ITRS Layer 2 Database Health

## Overview

`itrs.layer2.database-health.xml` provides baseline and extended SQL Toolkit
monitoring for MySQL, MariaDB, Oracle Database, PostgreSQL, Microsoft SQL
Server, SAP ASE, and IBM Db2. The checks cover database availability,
instance state, connection pressure, contention, capacity, and supported
replication or high-availability features.

The include supplies reusable Types, samplers, database connections, and
connection variables. It deliberately does not contain a Managed Entity
hierarchy, database credentials, or rules. Keep the functional include
read-only and put Gateway-specific configuration in
`config.layer2.database-health.xml` or the local Gateway setup.

For the example site name `acme`, the files are installed as:

| File | Purpose |
| --- | --- |
| `gateway/includes/itrs.layer2.database-health.xml` | Upgradeable database monitoring definitions |
| `gateway/acme/acme-config.layer2.database-health.xml` | Site-specific environments and overrides |

If no site name is configured, the site directory and file prefix are `local`.
The functional file is replaced during an include update; the site
configuration is preserved.

## Prerequisites

Before using this include, configure the Netprobe and the client libraries for
the selected database as described in
[Configure the Netprobe and database client libraries](https://docs.itrsgroup.com/docs/geneos/7.12.0/collection/sql-toolkit/index.html#configure-thenetprobe-and-database-client-libraries).
The required database client must be available to the Netprobe process, and
the Netprobe must be able to connect to the database endpoint.

The database administrator must also provision a dedicated, read-only
monitoring account with the minimum grants needed by the enabled queries.
Validate the account, queries, driver, database version, and required
permissions in the target environment before production use. Extended
samplers usually need more privileges than baseline samplers.

Use a protected password supplied through an encrypted or external Geneos
credential mechanism. Do not put a clear-text database password in the
include, a site file, or this example. Restrict the monitoring account to the
required endpoint and views, and do not collect application-table contents,
SQL text, secrets, or client identities unless there is an approved
operational requirement.

## Add the Include

Add `itrs.layer2.database-health.xml` to the Gateway. Also add the matching
site configuration file if you use it for local environments and overrides:

```bash
geneos includes extract --update
```

This installs missing bundled includes and preserves existing site
configuration. To add or refresh the Gateway references manually, use the
Gateway Setup Editor or the include XML below.

```xml
<includeGroup name="ITRS Layer 2 - Database Health">
    <include>
        <priority>1000</priority>
        <required>true</required>
        <location>../../includes/itrs.layer2.database-health.xml</location>
    </include>
    <include>
        <priority>500</priority>
        <required>true</required>
        <location>../../acme/acme-config.layer2.database-health.xml</location>
    </include>
</includeGroup>
```

Give the site configuration a numerically lower priority than the functional
include so that its settings take precedence. Use unique priorities that fit
the Gateway's include hierarchy.

## Configure a Database

Use a Netprobe that can reach the database. Create a logical Managed Entity for
each database endpoint or independently configured instance, then attach the
Baseline Type and, where required and permitted, the Extended Type.

Attach an Environment containing the connection settings to each Type
assignment. The following generalized MariaDB example follows the structure of
a Gateway with a remote Netprobe. Replace the names, addresses, database, and
monitoring account with approved values.

Paste each example into its corresponding top-level section in the Gateway
Setup Editor's **Paste XML** function. The blocks are intentionally separate so
that they can be validated and pasted independently.

### Probe

```xml
<probe name="db-netprobe">
    <hostname>192.0.2.25</hostname>
    <port>7036</port>
    <connectionSecurity>secure</connectionSecurity>
</probe>
```

### Managed Entity

```xml
<managedEntity name="database-host">
    <probe ref="db-netprobe"/>
    <addTypes>
        <type ref="Layer 2 - MariaDB - Baseline">
            <environment ref="Example MariaDB"/>
        </type>
        <type ref="Layer 2 - MariaDB - Extended">
            <environment ref="Example MariaDB"/>
        </type>
    </addTypes>
</managedEntity>
```

### Environment

```xml
<environment name="Example MariaDB">
    <var name="DB_SERVER">
        <string>db.example.com</string>
    </var>
    <var name="DB_NAME">
        <string>example_database</string>
    </var>
    <var name="DB_PORT">
        <integer>3306</integer>
    </var>
    <var name="DB_USER">
        <string>monitor_user</string>
    </var>
    <!-- Add DB_PASSWORD here using an encrypted or external password in GSE. -->
</environment>
```

In the Gateway Setup Editor, add `DB_PASSWORD` to `Example MariaDB` as an
encrypted or external password. Do not substitute a clear-text string for the
password. For a Gateway managed from configuration files, use the credential
representation approved for that installation.

Geneos cannot attach the same named Sampler or Type more than once to one
Managed Entity. To monitor two databases on the same server, create one
logical Managed Entity per endpoint or database and attach the relevant Type
once to each. Keep endpoint or database identity in the Managed Entity name or
attributes so that rules and cross-server views do not rely on ambiguous
display names.

## Select a Type

Each database has a baseline Type and an optional extended Type.

| Baseline Type | Extended Type | Purpose |
| --- | --- | --- |
| <code>Layer&nbsp;2&nbsp;-&nbsp;MySQL&nbsp;-&nbsp;Baseline</code> | <code>Layer&nbsp;2&nbsp;-&nbsp;MySQL&nbsp;-&nbsp;Extended</code> | MySQL instance health; optional replication and lock detail |
| <code>Layer&nbsp;2&nbsp;-&nbsp;MariaDB&nbsp;-&nbsp;Baseline</code> | <code>Layer&nbsp;2&nbsp;-&nbsp;MariaDB&nbsp;-&nbsp;Extended</code> | MariaDB and Galera health; optional replication and wait detail |
| <code>Layer&nbsp;2&nbsp;-&nbsp;Oracle&nbsp;-&nbsp;Baseline</code> | <code>Layer&nbsp;2&nbsp;-&nbsp;Oracle&nbsp;-&nbsp;Extended</code> | Oracle instance, capacity, and session health; optional PDB, RAC, and Data Guard detail |
| <code>Layer&nbsp;2&nbsp;-&nbsp;PostgreSQL&nbsp;-&nbsp;Baseline</code> | <code>Layer&nbsp;2&nbsp;-&nbsp;PostgreSQL&nbsp;-&nbsp;Extended</code> | PostgreSQL activity and contention; optional replication and statement statistics |
| <code>Layer&nbsp;2&nbsp;-&nbsp;SQL&nbsp;Server&nbsp;-&nbsp;Baseline</code> | <code>Layer&nbsp;2&nbsp;-&nbsp;SQL&nbsp;Server&nbsp;-&nbsp;Extended</code> | SQL Server state and capacity; optional Always On and wait detail |
| <code>Layer&nbsp;2&nbsp;-&nbsp;SAP&nbsp;ASE&nbsp;-&nbsp;Baseline</code> | <code>Layer&nbsp;2&nbsp;-&nbsp;SAP&nbsp;ASE&nbsp;-&nbsp;Extended</code> | SAP ASE inventory, connection, and space health; optional process-wait detail |
| <code>Layer&nbsp;2&nbsp;-&nbsp;IBM&nbsp;Db2&nbsp;-&nbsp;Baseline</code> | <code>Layer&nbsp;2&nbsp;-&nbsp;IBM&nbsp;Db2&nbsp;-&nbsp;Extended</code> | Db2 member, connection, lock, and tablespace health; optional log, buffer-pool, and HADR detail |

Start with the Baseline Type. Enable the Extended Type only when the target
supports the queries, the account has the required permissions, and the extra
topology or diagnostic visibility is operationally useful. Some extended
queries are deliberately topology-specific: an empty result can mean "not
applicable" on one role and "missing expected member" on another.

## Connection Variables

All Types include a product-specific defaults Environment. The defaults contain
sample intervals and connection values except `DB_PASSWORD`. Attach a local
Environment to override the values needed by the target database.

| Database | Connection variables |
| --- | --- |
| All databases | `DB_NAME`, `DB_USER`, `DB_PASSWORD` |
| MySQL / MariaDB | `DB_SERVER`, `DB_PORT` |
| Oracle Database | No additional connection variables |
| PostgreSQL | `DB_SERVER`, `DB_PORT` |
| SQL Server | `DB_SERVER`, `DB_PORT` |
| SAP ASE | `INTERFACE` |
| IBM Db2 | No additional connection variables |

Oracle and IBM Db2 use database or client-configuration names rather than a
server and port in this include. SAP ASE uses an entry from the ASE interfaces
file through `INTERFACE`. Oracle, SAP ASE, and IBM Db2 connectivity therefore
depends on the corresponding client configuration being correct for the
Netprobe.

`DB_SERVER` is the address of the database as seen from the Netprobe; it is
not necessarily the address of the Netprobe itself. Every connection is
configured to close after its queries complete.

### Replacing a Database Connection

If a database requires connectivity settings beyond the supplied variables,
copy the complete connection from `itrs.layer2.database-health.xml` into a
local, higher-priority include and redefine it at the **identical** XML path,
group, and name. The required path is:

`<staticVars>` / `<databaseConnections>` / `<databaseConnectionGroup>` named
`Layer 2 - Database Health` / `<databaseConnection>` with the original product
connection name.

Keep the group's and connection's names exactly as supplied. Gateway include
merging then replaces the default connection rather than creating another one.
For example, use this approach to add certificate validation, SSL modes,
wallet or trust-store settings, or other SQL Toolkit connection options.
Copy the whole connection, not only the option to change, and do not rename it.
Validate the replacement in a non-production Gateway first.

The include uses these sampling defaults:

| Variable | Default | Used by |
| --- | --- | --- |
| `NORMAL_SAMPLE` | 20 seconds | Baseline samplers |
| `SLOW_SAMPLE` | 300 seconds | Extended samplers |
| `FAST_SAMPLE` | 5 seconds | Available for custom rules or samplers; not used by the include |

Baseline queries have a 30-second timeout and extended queries have a
60-second timeout. Increase intervals for expensive queries or constrained
database services rather than collecting unrestricted diagnostic detail.

## What the Samplers Collect

The Baseline and Extended Types create separate dataviews for the named query
sets below. Availability is demonstrated by a successful connection and
fresh query result; the result does not prove that every application database
or transaction is healthy.

### MySQL

The baseline sampler collects:

* **System information:** server identity, version, and read-only settings.

    ```sql
    SHOW GLOBAL VARIABLES
    WHERE Variable_name IN
      ('version', 'version_comment', 'server_uuid', 'server_id',
       'read_only', 'super_read_only')
    ```

* **Connections and failures:** uptime, current and running threads, the
  connection high-water mark, and cumulative connection failures.

    ```sql
    SHOW GLOBAL STATUS
    WHERE Variable_name IN
      ('Uptime', 'Threads_connected', 'Threads_running',
       'Max_used_connections', 'Connections', 'Aborted_connects')
    ```

* **Connection limit:** the configured server connection limit.

    ```sql
    SHOW GLOBAL VARIABLES
    WHERE Variable_name IN ('max_connections')
    ```

* **InnoDB contention:** buffer-pool reads and requests, row-lock waits and
  wait time, and slow-query count.

    ```sql
    SHOW GLOBAL STATUS
    WHERE Variable_name IN
      ('Innodb_buffer_pool_read_requests', 'Innodb_buffer_pool_reads',
       'Innodb_row_lock_waits', 'Innodb_row_lock_time',
       'Slow_queries')
    ```

* **Schema inventory:** visible schemas and their default character sets and
  collations.

    ```sql
    SELECT SCHEMA_NAME, DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME
    FROM information_schema.SCHEMATA
    ORDER BY SCHEMA_NAME
    ```

The extended sampler collects:

* **Group replication members.**

    ```sql
    SELECT MEMBER_ID, MEMBER_HOST, MEMBER_PORT, MEMBER_ROLE, MEMBER_STATE
    FROM performance_schema.replication_group_members
    ```

* **Replica status.**

    ```sql
    SHOW REPLICA STATUS
    ```

* **Lock waits.**

    ```sql
    SELECT REQUESTING_ENGINE_LOCK_ID,
           COUNT(*) AS waiters
    FROM performance_schema.data_lock_waits
    GROUP BY REQUESTING_ENGINE_LOCK_ID
    ORDER BY waiters DESC
    ```

Replica and Performance Schema visibility may require additional grants. See
the MySQL documentation for
[SHOW STATUS](https://dev.mysql.com/doc/refman/8.4/en/show-status.html),
[status variables](https://dev.mysql.com/doc/refman/8.4/en/server-status-variables.html),
[SHOW VARIABLES](https://dev.mysql.com/doc/refman/8.4/en/show-variables.html),
[Group Replication members](https://dev.mysql.com/doc/refman/8.4/en/performance-schema-replication-group-members-table.html),
and [SHOW REPLICA STATUS](https://dev.mysql.com/doc/refman/8.4/en/show-replica-status.html).

### MariaDB

The baseline sampler collects:

* **System information:** server identity, version, read-only settings, and
  Galera configuration.

    ```sql
    SHOW GLOBAL VARIABLES
    WHERE Variable_name IN
      ('version', 'version_comment', 'server_id', 'read_only',
       'wsrep_on', 'wsrep_provider')
    ```

* **Connections and failures.**

    ```sql
    SHOW GLOBAL STATUS
    WHERE Variable_name IN
      ('Uptime', 'Threads_connected', 'Threads_running',
       'Max_used_connections', 'Connections', 'Aborted_connects')
    ```

* **Connection limit.**

    ```sql
    SHOW GLOBAL VARIABLES
    WHERE Variable_name IN ('max_connections')
    ```

* **InnoDB contention.**

    ```sql
    SHOW GLOBAL STATUS
    WHERE Variable_name IN
      ('Innodb_buffer_pool_read_requests', 'Innodb_buffer_pool_reads',
       'Innodb_row_lock_waits', 'Innodb_row_lock_time',
       'Slow_queries')
    ```

* **Galera status:** component state, membership, connection, readiness, and
  local state.

    ```sql
    SHOW GLOBAL STATUS
    WHERE Variable_name IN
      ('wsrep_cluster_status', 'wsrep_cluster_size', 'wsrep_connected',
       'wsrep_ready', 'wsrep_local_state_comment')
    ```

The extended sampler collects:

* **Replica status.**

    ```sql
    SHOW SLAVE STATUS
    ```

* **Aggregate Performance Schema wait summary.**

    ```sql
    SELECT EVENT_NAME, COUNT_STAR, SUM_TIMER_WAIT
    FROM performance_schema.events_waits_summary_global_by_event_name
    WHERE COUNT_STAR > 0
    ORDER BY SUM_TIMER_WAIT DESC
    LIMIT 20
    ```

MariaDB and MySQL replication interfaces and privileges are not
interchangeable; validate the statements against the installed MariaDB
version. See the MariaDB documentation for
[SHOW STATUS](https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/show/show-status),
[SHOW VARIABLES](https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/show/show-variables),
[status variables](https://mariadb.com/docs/server/server-management/variables-and-modes/server-status-variables),
[replica status](https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/show/show-replica-status),
and [Galera status](https://mariadb.com/docs/galera-cluster/reference/galera-cluster-status-variables).

### Oracle Database

The baseline sampler collects:

* **System information:** instance identity, version, startup, uptime,
  cluster-database mode, role, and status.

    ```sql
    SELECT 'instance_name' AS name, instance_name AS value FROM v$instance
    UNION ALL
    SELECT 'version', version FROM v$instance
    UNION ALL
    SELECT 'startup_time',
           TO_CHAR(startup_time, 'YYYY-MM-DD HH24:MI:SS') FROM v$instance
    UNION ALL
    SELECT 'uptime_seconds',
           TO_CHAR(ROUND((SYSDATE - startup_time) * 86400),
                   'FM9999999999999990') FROM v$instance
    UNION ALL
    SELECT 'cluster_database_mode', parallel FROM v$instance
    UNION ALL
    SELECT 'instance_role', instance_role FROM v$instance
    UNION ALL
    SELECT 'status', status FROM v$instance
    ```

* **Instance state.**

    ```sql
    SELECT instance_name, host_name, status, database_status, active_state,
           instance_role, logins, shutdown_pending
    FROM v$instance
    ```

* **Resource headroom:** process and session use against configured limits.

    ```sql
    SELECT resource_name, current_utilization, max_utilization, limit_value
    FROM v$resource_limit
    WHERE resource_name IN ('processes', 'sessions')
    ```

* **Tablespace capacity.**

    ```sql
    SELECT tablespace_name, used_space, tablespace_size, used_percent
    FROM dba_tablespace_usage_metrics
    ```

* **Session contention.**

    ```sql
    SELECT 'user_sessions' AS metric,
           COUNT(*) AS sessions,
           COUNT(CASE WHEN status = 'ACTIVE' THEN 1 END) AS active_sessions,
           COUNT(CASE WHEN blocking_session IS NOT NULL THEN 1 END) AS blocked_sessions
    FROM v$session
    WHERE type = 'USER'
    ```

The extended sampler collects:

* **Pluggable database state.**

    ```sql
    SELECT name, open_mode, restricted, recovery_status
    FROM v$pdbs
    ```

* **RAC instances.**

    ```sql
    SELECT inst_id, instance_name, host_name, status, database_status,
           active_state, instance_role
    FROM gv$instance
    ORDER BY inst_id
    ```

* **Data Guard transport and apply state.**

    ```sql
    SELECT inst_id, process, status, thread#, sequence#, delay_mins
    FROM gv$managed_standby
    WHERE process IN ('RFS', 'LNS', 'MRP0')
    ORDER BY inst_id, process
    ```

Use narrowly scoped `SELECT` grants on the required data-dictionary views; do
not use `SYSDBA`, `DBA`, or `SELECT ANY DICTIONARY` solely to enable this
include. See the Oracle documentation for
[V$INSTANCE](https://docs.oracle.com/en/database/oracle/oracle-database/19/refrn/V-INSTANCE.html),
[V$RESOURCE_LIMIT](https://docs.oracle.com/en/database/oracle/oracle-database/19/refrn/V-RESOURCE_LIMIT.html),
[DBA_TABLESPACE_USAGE_METRICS](https://docs.oracle.com/en/database/oracle/oracle-database/19/refrn/DBA_TABLESPACE_USAGE_METRICS.html),
and [V$SESSION](https://docs.oracle.com/en/database/oracle/oracle-database/19/refrn/V-SESSION.html).

### PostgreSQL

The baseline sampler collects:

* **System information:** version, connected database, cluster label, startup
  time, uptime, recovery mode, and WAL level.

    ```sql
    SELECT 'version' AS name, current_setting('server_version') AS value
    UNION ALL
    SELECT 'database', current_database()
    UNION ALL
    SELECT 'cluster_name', current_setting('cluster_name')
    UNION ALL
    SELECT 'startup_time', pg_postmaster_start_time()::text
    UNION ALL
    SELECT 'uptime_seconds',
           floor(extract(epoch FROM clock_timestamp() - pg_postmaster_start_time()))::text
    UNION ALL
    SELECT 'in_recovery', pg_is_in_recovery()::text
    UNION ALL
    SELECT 'wal_level', current_setting('wal_level')
    ```

* **Database activity and errors:** per-database commits, rollbacks, deadlocks,
  temporary-space use, current backends, and statistics reset time.

    ```sql
    SELECT datname, numbackends, xact_commit, xact_rollback,
           deadlocks, temp_bytes, stats_reset
    FROM pg_stat_database
    WHERE datname IS NOT NULL
    ORDER BY datname
    ```

* **Activity and contention:** active connections, lock waiters, idle
  transactions, and the oldest transaction.

    ```sql
    SELECT 'client_backends' AS metric,
           COUNT(*) AS connections,
           COUNT(*) FILTER (WHERE state = 'active') AS active,
           COUNT(*) FILTER (WHERE wait_event_type = 'Lock') AS lock_waiters,
           COUNT(*) FILTER (WHERE state = 'idle in transaction') AS idle_in_transaction,
           COALESCE(MAX(EXTRACT(EPOCH FROM (clock_timestamp() - xact_start)))
             FILTER (WHERE xact_start IS NOT NULL), 0) AS oldest_transaction_seconds
    FROM pg_stat_activity
    WHERE backend_type = 'client backend'
      AND pid <> pg_backend_pid()
    ```

* **Connection limit.**

    ```sql
    SELECT 'connection_limit' AS metric,
           current_setting('max_connections')::integer AS max_connections
    ```

The extended sampler collects:

* **Streaming replicas.**

    ```sql
    SELECT application_name, client_addr, state, sync_state,
           pg_wal_lsn_diff(pg_current_wal_lsn(), replay_lsn) AS replay_lag_bytes
    FROM pg_stat_replication
    ORDER BY application_name
    ```

* **WAL receiver state.**

    ```sql
    SELECT status, sender_host, sender_port, received_tli,
           latest_end_lsn, latest_end_time
    FROM pg_stat_wal_receiver
    ```

* **Statement statistics.**

    ```sql
    SELECT queryid, calls, total_exec_time, mean_exec_time
    FROM pg_stat_statements
    ORDER BY total_exec_time DESC
    LIMIT 20
    ```

The monitoring role needs `CONNECT` on each directly tested database.
Replication visibility and statement statistics require their specific
privileges and extensions. See the PostgreSQL documentation for
[monitoring statistics](https://www.postgresql.org/docs/16/monitoring-stats.html),
[predefined roles](https://www.postgresql.org/docs/16/predefined-roles.html),
[system information functions](https://www.postgresql.org/docs/16/functions-info.html),
and [recovery and start-time functions](https://www.postgresql.org/docs/16/functions-admin.html).

### SQL Server

The baseline sampler collects:

* **System information:** server identity, version, clustering, HADR
  capability, and manager state.

    ```sql
    SELECT info.name, info.value
    FROM (VALUES
      ('server_name', CONVERT(nvarchar(128), SERVERPROPERTY('ServerName'))),
      ('version', CONVERT(nvarchar(128), SERVERPROPERTY('ProductVersion'))),
      ('product_level', CONVERT(nvarchar(128), SERVERPROPERTY('ProductLevel'))),
      ('update_level', CONVERT(nvarchar(128), SERVERPROPERTY('ProductUpdateLevel'))),
      ('edition', CONVERT(nvarchar(128), SERVERPROPERTY('Edition'))),
      ('failover_clustered', CONVERT(nvarchar(128), SERVERPROPERTY('IsClustered'))),
      ('availability_groups_enabled', CONVERT(nvarchar(128), SERVERPROPERTY('IsHadrEnabled'))),
      ('hadr_manager_status', CONVERT(nvarchar(128), SERVERPROPERTY('HadrManagerStatus')))
    ) AS info(name, value)
    ```

* **Startup and uptime.**

    ```sql
    SELECT 'startup_time' AS name,
           CONVERT(nvarchar(128), sqlserver_start_time, 126) AS value
    FROM sys.dm_os_sys_info
    UNION ALL
    SELECT 'uptime_seconds',
           CONVERT(nvarchar(128), (ms_ticks - sqlserver_start_time_ms_ticks) / 1000)
    FROM sys.dm_os_sys_info
    ```

* **Database state:** access mode, read-only state, recovery model, and log
  reuse reason.

    ```sql
    SELECT name, state_desc, user_access_desc, is_read_only,
           recovery_model_desc, log_reuse_wait_desc
    FROM sys.databases
    ```

* **Transaction-log capacity.**

    ```sql
    SELECT DB_NAME() AS database_name, total_log_size_in_bytes,
           used_log_space_in_bytes, used_log_space_in_percent
    FROM sys.dm_db_log_space_usage
    ```

* **Running and blocked requests.**

    ```sql
    SELECT 'requests' AS metric,
           COUNT(*) AS request_count,
           COUNT(CASE WHEN blocking_session_id <> 0 THEN 1 END) AS blocked_requests,
           COALESCE(MAX(total_elapsed_time), 0) AS longest_elapsed_ms
    FROM sys.dm_exec_requests
    WHERE session_id <> @@SPID
      AND database_id = DB_ID()
    ```

The extended sampler collects:

* **Local Availability Group replica state.**

    ```sql
    SELECT ag.name AS availability_group,
           ar.replica_server_name, ars.role_desc, ars.operational_state_desc,
           ars.connected_state_desc, ars.synchronization_health_desc
    FROM sys.availability_groups AS ag
    JOIN sys.availability_replicas AS ar ON ar.group_id = ag.group_id
    JOIN sys.dm_hadr_availability_replica_states AS ars ON ars.replica_id = ar.replica_id
    WHERE ars.is_local = 1
    ```

* **Local Availability Group database state.**

    ```sql
    SELECT ag.name AS availability_group,
           ar.replica_server_name, d.name AS database_name,
           drs.synchronization_state_desc, drs.synchronization_health_desc,
           drs.database_state_desc
    FROM sys.dm_hadr_database_replica_states AS drs
    JOIN sys.availability_replicas AS ar ON ar.replica_id = drs.replica_id
    JOIN sys.availability_groups AS ag ON ag.group_id = drs.group_id
    JOIN sys.databases AS d ON d.database_id = drs.database_id
    WHERE drs.is_local = 1
    ```

* **Wait statistics.**

    ```sql
    SELECT wait_type, waiting_tasks_count, wait_time_ms
    FROM sys.dm_os_wait_stats
    WHERE waiting_tasks_count > 0
    ORDER BY wait_time_ms DESC
    ```

Dynamic Management View access depends on the SQL Server version and requires
the appropriate server-state permission. A missing or masked result is not a
zero value. See Microsoft documentation for
[SERVERPROPERTY](https://learn.microsoft.com/en-us/sql/t-sql/functions/serverproperty-transact-sql),
[sys.dm_os_sys_info](https://learn.microsoft.com/en-us/sql/relational-databases/system-dynamic-management-objects/sys-dm-os-sys-info-transact-sql),
[sys.databases](https://learn.microsoft.com/en-us/sql/relational-databases/system-catalog-views/sys-databases-transact-sql),
[sys.dm_db_log_space_usage](https://learn.microsoft.com/en-us/sql/relational-databases/system-dynamic-management-objects/sys-dm-db-log-space-usage-transact-sql),
and [sys.dm_exec_requests](https://learn.microsoft.com/en-us/sql/relational-databases/system-dynamic-management-objects/sys-dm-exec-requests-transact-sql).

### SAP ASE

The baseline sampler collects:

* **System information:** server name, version, and maximum connections.

    ```sql
    SELECT 'server_name' AS name, @@servername AS value
    UNION ALL
    SELECT 'version', LEFT(@@version, 255)
    UNION ALL
    SELECT 'maximum_connections', CONVERT(varchar(20), @@max_connections)
    ```

* **Database inventory.**

    ```sql
    SELECT name, dbid, status, status2, status3, crdate
    FROM master..sysdatabases
    ORDER BY dbid
    ```

* **Connection pressure.**

    ```sql
    SELECT COUNT(*) AS connections,
           SUM(CASE WHEN blocked <> 0 THEN 1 ELSE 0 END) AS blocked_sessions
    FROM master..sysprocesses
    WHERE spid <> @@spid
    ```

* **Database space.**

    ```sql
    EXEC master..sp_spaceused
    ```

The extended sampler collects:

* **Process waits.**

    ```sql
    SELECT w.WaitEventID, e.Description,
           COUNT(*) AS waiting_processes,
           SUM(w.WaitTime) AS total_wait_time
    FROM master..monProcessWaits AS w
    JOIN master..monWaitEventInfo AS e
      ON e.WaitEventID = w.WaitEventID
    GROUP BY w.WaitEventID, e.Description
    ORDER BY SUM(w.WaitTime) DESC
    ```

The MDA process-wait query may require a monitoring role and enabled MDA
tables. Do not grant `sa_role`, `oper_role`, or application-table access
solely for this include. See the SAP documentation for
[Adaptive Server Enterprise system catalogs](https://help.sap.com/docs/SAP_ASE)
and [SAP ASE administration](https://help.sap.com/docs/SAP_ASE).

### IBM Db2

The baseline sampler collects:

* **System information:** instance, service, build, and fix-pack levels.

    ```sql
    SELECT INST_NAME, SERVICE_LEVEL, BLD_LEVEL, FIXPACK_NUM
    FROM TABLE(SYSPROC.ENV_GET_INST_INFO()) AS instance_info
    ```

* **Member and database state:** activation, connections, lock waits, and last
  backup.

    ```sql
    SELECT MEMBER, DB_STATUS, DB_ACTIVATION_STATE, DB_CONN_TIME,
           APPLS_CUR_CONS, APPLS_IN_DB2, NUM_LOCKS_WAITING, LAST_BACKUP
    FROM TABLE(MON_GET_DATABASE(-2)) AS database_metrics
    ORDER BY MEMBER
    ```

* **Connection states by member.**

    ```sql
    SELECT COORD_MEMBER, APPLICATION_STATE, COUNT(*) AS connections
    FROM TABLE(MON_GET_CONNECTION(NULL, -2, 0)) AS connection_metrics
    GROUP BY COORD_MEMBER, APPLICATION_STATE
    ORDER BY COORD_MEMBER, APPLICATION_STATE
    ```

* **Tablespace capacity and state.**

    ```sql
    SELECT MEMBER, TBSP_NAME, TBSP_STATE, TBSP_TYPE,
           TBSP_TOTAL_SIZE_KB, TBSP_USED_SIZE_KB, TBSP_USABLE_SIZE_KB
    FROM TABLE(MON_GET_TABLESPACE('', -2)) AS tablespace_metrics
    ORDER BY MEMBER, TBSP_NAME
    ```

The extended sampler collects:

* **Transaction-log use.**

    ```sql
    SELECT MEMBER, TOTAL_LOG_USED_KB, TOTAL_LOG_AVAILABLE_KB
    FROM TABLE(MON_GET_TRANSACTION_LOG(-2)) AS transaction_log
    ORDER BY MEMBER
    ```

* **Buffer-pool activity.**

    ```sql
    SELECT MEMBER, BP_NAME, POOL_DATA_P_READS,
           POOL_ASYNC_DATA_READS, POOL_DATA_WRITES
    FROM TABLE(MON_GET_BUFFERPOOL('', -2)) AS buffer_pool
    ORDER BY MEMBER, BP_NAME
    ```

* **HADR state.**

    ```sql
    SELECT STANDBY_ID, HADR_ROLE, HADR_STATE, HADR_CONNECT_STATUS,
           PRIMARY_MEMBER_HOST, STANDBY_MEMBER_HOST, HADR_SYNCMODE
    FROM TABLE(MON_GET_HADR(-2)) AS hadr
    ORDER BY STANDBY_ID
    ```

Grant `EXECUTE` only on the monitor functions required by enabled queries. A
SQL Toolkit connection tests one database; it does not prove the health of
every database catalogued by the instance. See IBM Db2 documentation for
[MON_GET_DATABASE](https://www.ibm.com/docs/en/db2/11.5.x?topic=functions-mon-get-database-get-database-metrics),
[MON_GET_CONNECTION](https://www.ibm.com/docs/en/db2/11.5.x?topic=functions-mon-get-connection-get-connection-metrics),
[MON_GET_TABLESPACE](https://www.ibm.com/docs/en/db2/11.5.x?topic=functions-mon-get-tablespace-get-tablespace-metrics),
and [MON_GET_HADR](https://www.ibm.com/docs/en/db2/11.5.x?topic=functions-mon-get-hadr-get-high-availability-disaster-recovery-hadr-metrics).

## Topology and Multiple Instances

Use Layer 1 host and filesystem monitoring alongside this include. Database
metrics do not replace CPU, memory, disk, network, backup, or recovery
monitoring.

For a clustered or replicated service:

* monitor every expected server or member with a separate logical Managed
  Entity;

* store or otherwise define the expected role and membership;

* monitor the client-visible writable endpoint in addition to member health;

* treat a healthy standby or empty role-specific view according to that
  server's expected role rather than applying a universal severity.

When a query returns multiple databases, tablespaces, PDBs, members, or
channels, preserve the object identity as the row identity. Maintain an
explicit expected inventory; a missing row is not the same as a healthy zero.
For end-to-end application availability, add a small approved synthetic
transaction rather than assuming that catalog access proves application
readiness.

## Alerting

The include creates an empty `Layer 2 - Database Health` rule group and does
not install universal warning or critical thresholds. Create rules in a local,
higher-priority include after measuring the workload and defining the expected
database topology.

Useful starting points include:

* warn when connection or storage usage is sustained at or above 80%, and
  treat sustained use at or above 90% as critical;

* alert on repeated connection or query failures, stale samples, unexpected
  restarts, and unexpected database or member state;

* alert when expected databases, replicas, members, or channels are missing;

* investigate sustained blocking, old transactions, and replication lag
  using service-specific durations and counts;

* calculate rates for cumulative counters and discard deltas after a reset,
  restart, failover, missing sample, or decreasing counter;

* use sustained conditions, hysteresis, and maintenance windows to avoid
  flapping.

Treat query errors, missing required data, and stale samples as unknown or
failure conditions, not as zero. Do not apply universal cache-hit, wait-time,
slow-query, or query-duration thresholds. Establish the normal range first and
correlate database symptoms with host resources, application errors, backups,
replication, and the service recovery objectives.

## Upgrades and Customisation

Do not edit `itrs.layer2.database-health.xml`. To change sample intervals or
connection settings, override variables in a local Environment. To customise a
sampler or rule, copy and redefine the complete item in a higher-priority
include; a redefinition replaces the original item rather than merging with
it.

Install or update the bundled includes with the usual Cordial include
workflow. Test the updated definitions and database permissions in a
non-production Gateway first. The local site configuration is not overwritten,
so review its overrides when a future release adds or changes variables,
samplers, or queries.
