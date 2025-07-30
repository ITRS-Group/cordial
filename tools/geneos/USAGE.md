# `geneos` Usage Guide

This comprehensive guide provides detailed examples and use cases for managing your Geneos environment with the `geneos` command.

## Table of Contents

- [Getting Started](#getting-started)
- [Core Operations](#core-operations)
- [Instance Management](#instance-management)
- [Software Management](#software-management)
- [Configuration Management](#configuration-management)
- [Security and Certificates](#security-and-certificates)
- [Remote Host Management](#remote-host-management)
- [Monitoring and Diagnostics](#monitoring-and-diagnostics)
- [Automation and Scripting](#automation-and-scripting)
- [Advanced Topics](#advanced-topics)
- [Troubleshooting Examples](#troubleshooting-examples)
- [Best Practices](#best-practices)

## Getting Started

### Command Structure

All `geneos` commands follow this consistent pattern:

```bash
geneos COMMAND [flags] [TYPE] [NAME...]
```

- **COMMAND**: What to do (`start`, `stop`, `list`, etc.)
- **flags**: Options like `-p` for port, `-l` for logs
- **TYPE**: Component type (`gateway`, `netprobe`, `san`, etc.)
- **NAME**: Instance names (supports wildcards like `prod*`)

### Basic Navigation

```bash
# Get general help
geneos help

# Get help for specific commands
geneos help start
geneos help gateway

# List all available commands
geneos help | grep -E "^  \w+"
```

## Core Operations

### Viewing Your Environment

#### List All Instances
```bash
# Show all configured instances
geneos list
geneos ls                    # Short alias

# Show only specific types
geneos ls gateways
geneos ls netprobes

# Show instances matching patterns
geneos ls "prod*"           # All instances starting with "prod"
geneos ls "*test*"          # All instances containing "test"
```

**Example Output:**
```
Type      Name         Host       Flags Port  Version           Home
gateway   prod-gw1     localhost  PA    7039  active_prod:6.6.0 /opt/geneos/gateway/gateways/prod-gw1
netprobe  prod-np1     localhost  A     7036  active_prod:6.6.0 /opt/geneos/netprobe/netprobes/prod-np1
san       prod-san1    localhost  A     7103  active_prod:6.6.0 /opt/geneos/netprobe/sans/prod-san1
```

**Flag Meanings:**
- `P` = Protected
- `A` = Auto-start enabled
- `D` = Disabled
- `T` = TLS configured

#### Show Running Instances
```bash
# Show what's currently running
geneos ps
geneos status               # Alias

# Show with additional details
geneos ps --long
geneos ps -l

# Show in different formats
geneos ps --json           # JSON output
geneos ps --csv            # CSV output
```

**Example Output:**
```
Type      Name         Host       PID   Ports    User   Group  Starttime             Version
gateway   prod-gw1     localhost  1234  [7039]   geneos geneos 2024-01-15T10:30:00Z  active_prod:6.6.0
netprobe  prod-np1     localhost  1235  [7036]   geneos geneos 2024-01-15T10:30:05Z  active_prod:6.6.0
```

### Starting and Stopping

#### Basic Control
```bash
# Start all instances
geneos start

# Start specific types
geneos start gateways
geneos start netprobes

# Start specific instances
geneos start gateway prod-gw1
geneos start netprobe prod-np1 prod-np2

# Start with log following
geneos start gateway prod-gw1 -l
geneos start gateway prod-gw1 --log
```

#### Advanced Start Options
```bash
# Start with extra command line arguments
geneos start gateway prod-gw1 -x "-skip-cache"

# Start with additional environment variables
geneos start netprobe prod-np1 -e "DEBUG=true" -e "LOG_LEVEL=info"

# Force start protected instances
geneos start gateway prod-gw1 --force
```

#### Stopping and Restarting
```bash
# Stop instances
geneos stop
geneos stop gateways
geneos stop gateway prod-gw1

# Force stop (immediate SIGKILL)
geneos stop gateway prod-gw1 --kill

# Restart instances
geneos restart
geneos restart gateways
geneos restart gateway prod-gw1

# Restart all matching instances (even if stopped)
geneos restart netprobes --all

# Restart with log following
geneos restart gateway prod-gw1 -l
```

## Instance Management

### Creating New Instances

#### Basic Instance Creation
```bash
# Create a Gateway
geneos add gateway MyGateway -p 7039

# Create a Netprobe
geneos add netprobe MyNetprobe -g localhost:7039

# Create a Self-Announcing Netprobe (SAN)
geneos add san MySAN -g gateway.example.com:7039

# Create and start immediately
geneos add gateway MyGateway -p 7039 --start

# Create and follow logs
geneos add gateway MyGateway -p 7039 --log
```

#### Advanced Instance Creation
```bash
# Create Gateway with include files
geneos add gateway MyGW -p 7039 \
  --include "100:/path/to/shared.xml" \
  --include "200:/path/to/app.xml"

# Create SAN with attributes and types
geneos add san MySAN -g gw1:7039 -g gw2:7039 \
  --attribute "ENVIRONMENT=production" \
  --attribute "DATACENTER=east" \
  --type "Infrastructure" \
  --type "Application"

# Create with custom environment variables
geneos add netprobe MyNP -g localhost:7039 \
  --env "JAVA_HOME=/usr/lib/jvm/java-11" \
  --env "CUSTOM_PATH=/opt/custom"

# Create with imported files
geneos add gateway MyGW -p 7039 \
  --import "license.lic" \
  --import "certs/ca-cert.pem"
```

### Copying and Moving Instances

#### Local Operations
```bash
# Copy an instance
geneos copy gateway SourceGW TargetGW

# Move/rename an instance
geneos move gateway OldName NewName
geneos mv gateway OldName NewName     # Alias

# Copy with port change
geneos copy gateway SourceGW TargetGW
geneos set gateway TargetGW -p 7040
```

#### Remote Operations
```bash
# Copy to remote host
geneos copy gateway LocalGW RemoteGW@remotehost

# Move between hosts
geneos move gateway LocalGW gateway@remotehost

# Copy from remote to local
geneos copy gateway RemoteGW@remotehost LocalGW
```

### Instance Protection and Control

#### Protection
```bash
# Protect an instance from accidental changes
geneos protect gateway CRITICAL_GW

# Unprotect an instance
geneos protect gateway CRITICAL_GW --unprotect

# View protection status
geneos list | grep "P"
```

#### Enable/Disable
```bash
# Disable an instance (prevents all starts)
geneos disable gateway MyGW

# Enable a disabled instance
geneos enable gateway MyGW

# View disabled instances
geneos list | grep "D"
```

#### Cleanup and Deletion
```bash
# Clean up log and temporary files
geneos clean gateway MyGW
geneos clean                # Clean all instances

# Delete an instance (must be stopped/disabled first)
geneos delete gateway MyGW

# Force delete a protected instance
geneos delete gateway MyGW --force
```

## Software Management

### Package Information

#### Viewing Installed Packages
```bash
# List all installed packages
geneos package list
geneos package ls          # Alias

# List packages for specific component
geneos package ls gateway
geneos package ls netprobe

# Show in different formats
geneos package ls --json
geneos package ls --csv
```

**Example Output:**
```
Component Host      Version         Links        LastModified          Path
gateway   localhost 6.6.0 (latest)  active_prod  2024-01-15T10:00:00Z  /opt/geneos/packages/gateway/6.6.0
gateway   localhost 6.5.0                        2024-01-10T09:00:00Z  /opt/geneos/packages/gateway/6.5.0
netprobe  localhost 6.6.0 (latest)  active_prod  2024-01-15T10:00:00Z  /opt/geneos/packages/netprobe/6.6.0
```

### Installing Software

#### Basic Installation
```bash
# Install latest version of a component
geneos package install gateway -u email@example.com

# Install specific version
geneos package install gateway --version 6.5.0 -u email@example.com

# Install and update instances
geneos package install gateway --update -u email@example.com

# Install from local archive
geneos package install gateway --archive /path/to/geneos-gateway-6.6.0.tar.gz
```

#### Advanced Installation
```bash
# Install with specific base name
geneos package install gateway --base dev -u email@example.com

# Install and force update protected instances
geneos package install gateway --update --force -u email@example.com

# Install from local directory
geneos package install gateway --local --archive /path/to/archives/

# Install minimal netprobe
geneos package install netprobe --minimal -u email@example.com
```

### Updating Software

#### Basic Updates
```bash
# Update to latest version
geneos package update gateway

# Update to specific version
geneos package update gateway --version 6.6.0

# Update specific base version
geneos package update gateway --base active_prod
```

#### Advanced Updates
```bash
# Update and restart instances
geneos package update gateway --restart

# Update with force (protected instances)
geneos package update gateway --force

# Update specific base to specific version
geneos package update gateway --base dev --version 6.6.0 --force
```

### Uninstalling Software

```bash
# Uninstall a specific version
geneos package uninstall gateway 6.5.0

# Uninstall and update base links
geneos package uninstall gateway 6.5.0 --update

# Force uninstall (stop protected instances)
geneos package uninstall gateway 6.5.0 --update --force
```

## Configuration Management

### Environment Variables

#### Setting Environment Variables
```bash
# Set single environment variable
geneos set netprobe MyNP -e "JAVA_HOME=/usr/lib/jvm/java-11"

# Set multiple environment variables
geneos set netprobe MyNP \
  -e "JAVA_HOME=/usr/lib/jvm/java-11" \
  -e "TNS_ADMIN=/etc/oracle/network/admin" \
  -e "LD_LIBRARY_PATH=/opt/oracle/lib"

# Set variable with spaces
geneos set netprobe MyNP -e "CUSTOM_VAR=value with spaces"

# Set secure environment variable (encrypted)
geneos set netprobe MyNP --secureenv "DB_PASSWORD"
```

#### Removing Environment Variables
```bash
# Remove specific environment variable
geneos unset netprobe MyNP -e "JAVA_HOME"

# Remove multiple environment variables
geneos unset netprobe MyNP -e "JAVA_HOME" -e "TNS_ADMIN"
```

### Instance Parameters

#### Setting Parameters
```bash
# Set basic parameters
geneos set gateway MyGW port=7039 user=geneos

# Append to existing parameters
geneos set gateway MyGW options+="-extra-option"

# Set secure parameters (encrypted)
geneos set gateway MyGW --secure "licdsecure"
```

#### Gateway-Specific Configuration
```bash
# Add include files
geneos set gateway MyGW \
  --include "100:/path/to/shared.xml" \
  --include "200:/path/to/app-specific.xml"

# Set gateway options
geneos set gateway MyGW \
  options="-demo -log debug" \
  port=7039 \
  user=geneos
```

#### SAN-Specific Configuration
```bash
# Set SAN attributes
geneos set san MySAN \
  --attribute "ENVIRONMENT=production" \
  --attribute "COMPONENT=database" \
  --attribute "LOCATION=datacenter1"

# Set SAN types
geneos set san MySAN \
  --type "Infrastructure" \
  --type "Database"

# Set SAN variables
geneos set san MySAN \
  --variable "string:APP_NAME=MyApp" \
  --variable "integer:POLL_INTERVAL=30" \
  --variable "boolean:DEBUG_MODE=false"

# Set gateway connections
geneos set san MySAN \
  --gateway "gw1.example.com:7039" \
  --gateway "gw2.example.com:7039"
```

### Configuration Templates

#### Rebuilding from Templates
```bash
# Rebuild instance configuration from template
geneos rebuild san MySAN

# Force rebuild (override protection)
geneos rebuild san MySAN --force

# Rebuild all SANs
geneos rebuild san
```

### Viewing Configuration

#### Show Instance Configuration
```bash
# Show full configuration
geneos show gateway MyGW

# Show raw configuration (unexpanded variables)
geneos show gateway MyGW --raw

# Show startup command
geneos command gateway MyGW

# Show instance directory
geneos home gateway MyGW
```

## Security and Certificates

### TLS Certificate Management

#### Initialize TLS System
```bash
# Initialize TLS subsystem
geneos tls init

# Force recreate root certificates
geneos tls init --force
```

#### Create and Manage Certificates
```bash
# Create certificate for instance
geneos tls new gateway MyGW
geneos tls new netprobe MyNP

# Renew existing certificate
geneos tls renew gateway MyGW

# Create certificate with custom settings
geneos tls new gateway MyGW --days 365
```

#### View Certificate Information
```bash
# List all certificates
geneos tls list

# List with detailed information
geneos tls list --long

# List including root/signing certificates
geneos tls list --all

# Show certificates for specific type
geneos tls list gateway
```

#### Certificate Synchronization
```bash
# Sync certificates to remote hosts
geneos tls sync

# Sync to specific host
geneos tls sync --host remoteserver
```

### AES Encryption and Credentials

#### Key File Management
```bash
# Create new AES key file
geneos aes new

# Create key file for specific component
geneos aes new --shared gateway
```

#### Encoding/Decoding Values
```bash
# Encode a secret
geneos aes encode "my secret value"

# Encode from file
geneos aes encode --source /path/to/secret.txt

# Decode a value
geneos aes decode "+encs+12345678+abcdef..."
```

#### Credential Storage
```bash
# Store download credentials
geneos login downloads -u email@example.com

# Store Gateway credentials
geneos login gateway:MyGW -u readonly

# Store SMTP credentials
geneos login smtp.example.com -u notifications@company.com

# Remove stored credentials
geneos logout downloads
```

## Remote Host Management

### Adding and Managing Hosts

#### Add Remote Hosts
```bash
# Add simple host
geneos host add server1

# Add with SSH URL
geneos host add server1 ssh://user@server1.example.com

# Add with custom port and path
geneos host add server1 ssh://user@server1.example.com:2222/opt/itrs

# Add with private key
geneos host add server1 ssh://user@server1.example.com \
  --privatekey ~/.ssh/geneos_key

# Add and initialize
geneos host add server1 ssh://user@server1.example.com --init
```

#### View and Manage Hosts
```bash
# List configured hosts
geneos host list

# Show host details
geneos host show server1

# Test host connectivity
geneos host show server1 --test

# Remove a host
geneos host delete server1
```

### Remote Operations

#### Deploy to Remote Hosts
```bash
# Deploy new instance to remote host
geneos deploy gateway RemoteGW --host server1

# Deploy with full configuration
geneos deploy san RemoteSAN --host server1 \
  --gateway "localhost:7039" \
  --attribute "ENVIRONMENT=production"
```

#### Manage Remote Instances
```bash
# List instances on specific host
geneos list --host server1

# Start instances on remote host
geneos start --host server1

# View logs from remote instances
geneos logs --host server1

# Copy instance to remote host
geneos copy gateway LocalGW RemoteGW@server1
```

## Monitoring and Diagnostics

### Log Management

#### Basic Log Viewing
```bash
# View last 10 lines of all logs
geneos logs

# View specific number of lines
geneos logs --lines 50
geneos logs -n 50

# View entire log file
geneos logs --cat

# Follow logs in real-time
geneos logs --follow
geneos logs -f
```

#### Advanced Log Operations
```bash
# Filter logs by content
geneos logs --match "ERROR"
geneos logs --ignore "DEBUG"

# Show stderr logs
geneos logs --stderr

# Show Collection Agent logs (netprobes)
geneos logs --ca

# View logs for specific instances
geneos logs gateway MyGW --lines 100
geneos logs netprobe "prod*" --follow
```

#### Log Examples by Scenario
```bash
# Debug startup issues
geneos logs gateway MyGW --stderr --lines 50

# Monitor for errors across all instances
geneos logs --follow --match "ERROR|SEVERE|FATAL"

# Check recent activity
geneos logs --lines 20

# Troubleshoot specific netprobe
geneos logs netprobe MyNP --ca --follow
```

### Health Monitoring

#### Instance Status Checks
```bash
# Quick health check
geneos ps

# Detailed status with network info
geneos ps --network --long

# Check for failed starts
geneos list | grep -v "running"

# Monitor resource usage
geneos ps --json | jq '.[] | {name: .name, memory: .memory}'
```

#### Automated Monitoring Scripts
```bash
#!/bin/bash
# Health check script
echo "=== Geneos Health Check ==="
echo "Running instances:"
geneos ps --csv | wc -l

echo "Failed instances:"
geneos list | grep -c "D\|stopped"

echo "Recent errors:"
geneos logs --lines 100 --match "ERROR" | wc -l
```

### Taking Snapshots

```bash
# Take snapshot of all dataviews
geneos snapshot

# Take snapshot of specific gateway
geneos snapshot gateway MyGW

# Save snapshots to specific directory
geneos snapshot --output /path/to/snapshots

# Take snapshot with custom filename
geneos snapshot gateway MyGW --file "backup-$(date +%Y%m%d).xml"
```

## Automation and Scripting

### Scripting Best Practices

#### Error Handling
```bash
#!/bin/bash
set -e  # Exit on error

# Check if instance is running
if ! geneos ps gateway MyGW >/dev/null 2>&1; then
    echo "Gateway MyGW is not running, starting..."
    geneos start gateway MyGW
fi

# Wait for startup
sleep 10

# Verify it started
if geneos ps gateway MyGW >/dev/null 2>&1; then
    echo "Gateway MyGW started successfully"
else
    echo "Failed to start Gateway MyGW"
    exit 1
fi
```

#### Non-Interactive Mode
```bash
# Set environment for automation
export GENEOS_NON_INTERACTIVE=true

# Use JSON output for parsing
RUNNING_COUNT=$(geneos ps --json | jq '. | length')
echo "Currently running instances: $RUNNING_COUNT"

# Check specific instance status
if geneos ps gateway MyGW --json | jq -e '.[] | select(.name == "MyGW")' >/dev/null; then
    echo "Gateway MyGW is running"
else
    echo "Gateway MyGW is not running"
fi
```

### Common Automation Tasks

#### Rolling Restart Script
```bash
#!/bin/bash
# Rolling restart of all gateways

GATEWAYS=$(geneos list gateways --json | jq -r '.[].name')

for gw in $GATEWAYS; do
    echo "Restarting gateway: $gw"
    geneos restart gateway "$gw"
    
    # Wait for restart
    sleep 30
    
    # Verify it's running
    if geneos ps gateway "$gw" >/dev/null 2>&1; then
        echo "Gateway $gw restarted successfully"
    else
        echo "ERROR: Gateway $gw failed to restart"
        exit 1
    fi
done
```

#### Update and Restart Script
```bash
#!/bin/bash
# Update gateway software and restart instances

echo "Updating gateway software..."
geneos package update gateway --force

echo "Restarting all gateways..."
geneos restart gateways

echo "Waiting for startup..."
sleep 60

echo "Verifying all gateways are running..."
EXPECTED=$(geneos list gateways --json | jq '. | length')
RUNNING=$(geneos ps gateways --json | jq '. | length')

if [ "$EXPECTED" -eq "$RUNNING" ]; then
    echo "All $RUNNING gateways are running successfully"
else
    echo "ERROR: Expected $EXPECTED gateways, but only $RUNNING are running"
    exit 1
fi
```

#### Backup Script
```bash
#!/bin/bash
# Backup all instance configurations

BACKUP_DIR="/backup/geneos/$(date +%Y%m%d)"
mkdir -p "$BACKUP_DIR"

echo "Creating backup in $BACKUP_DIR"

# Backup each instance
geneos list --json | jq -r '.[].name as $name | .[].type as $type | "\($type) \($name)"' | \
while read type name; do
    echo "Backing up $type $name"
    geneos backup "$type" "$name" --output "$BACKUP_DIR/${type}-${name}.tar.gz"
done

echo "Backup complete"
```

## Advanced Topics

### Custom Base Versions

#### Managing Multiple Versions
```bash
# Install gateway with custom base name
geneos package install gateway --base dev --version 6.6.0

# Install production version
geneos package install gateway --base prod --version 6.5.0

# Create instance using specific base
geneos add gateway DevGW --base dev -p 7040
geneos add gateway ProdGW --base prod -p 7039

# Update dev base to latest
geneos package update gateway --base dev
```

### Template Customization

#### Using Custom Templates
```bash
# Create instance with custom template
geneos add san MySAN --template /path/to/custom.xml.gotmpl

# Deploy with template
geneos deploy san MySAN --template https://example.com/template.xml
```

### Integration with External Tools

#### Exporting Configuration
```bash
# Export instance configuration
geneos show gateway MyGW --json > gateway-config.json

# Export all configurations
geneos list --json > all-instances.json

# Export for backup
geneos backup gateway MyGW --format json
```

#### Health Check Integration
```bash
# Nagios/monitoring integration
#!/bin/bash
CRITICAL_INSTANCES="prod-gw1 prod-gw2"
EXIT_CODE=0

for instance in $CRITICAL_INSTANCES; do
    if ! geneos ps gateway "$instance" >/dev/null 2>&1; then
        echo "CRITICAL: Gateway $instance is not running"
        EXIT_CODE=2
    fi
done

if [ $EXIT_CODE -eq 0 ]; then
    echo "OK: All critical gateways are running"
fi

exit $EXIT_CODE
```

## Troubleshooting Examples

### Common Issues and Solutions

#### Instance Won't Start
```bash
# 1. Check if it's disabled
geneos list gateway MyGW | grep "D"

# 2. Check configuration
geneos show gateway MyGW

# 3. Check startup command
geneos command gateway MyGW

# 4. Check for port conflicts
netstat -ln | grep :7039

# 5. Try starting with verbose output
geneos start gateway MyGW --log --stderr

# 6. Check recent logs
geneos logs gateway MyGW --lines 100 --stderr
```

#### Performance Issues
```bash
# Check resource usage
geneos ps gateway MyGW --long

# Check for file descriptor issues
geneos ps gateway MyGW --files

# Check network connections
geneos ps gateway MyGW --network

# Monitor logs for errors
geneos logs gateway MyGW --follow --match "ERROR|OutOfMemory|Too many"
```

#### Connection Issues
```bash
# Test remote host connectivity
geneos host show remoteserver --test

# Check TLS certificate validity
geneos tls list gateway MyGW

# Verify network connectivity
geneos ps gateway MyGW --network | grep LISTEN

# Check gateway connections for SANs
geneos show san MySAN | jq '.configuration.gateways'
```

#### Configuration Problems
```bash
# Rebuild configuration from template
geneos rebuild san MySAN

# Compare configurations
geneos show gateway GW1 > gw1.json
geneos show gateway GW2 > gw2.json
diff gw1.json gw2.json

# Validate configuration syntax
geneos show gateway MyGW --raw

# Reset to default configuration
geneos migrate gateway MyGW
```

### Recovery Procedures

#### Recovering from Failed Updates
```bash
# List available versions
geneos package list gateway

# Rollback to previous version
geneos package update gateway --version 6.5.0 --force

# Restart affected instances
geneos restart gateways
```

#### Disaster Recovery
```bash
# Restore from backup
geneos restore gateway MyGW --archive /backup/gateway-MyGW.tar.gz

# Recreate instance from configuration
geneos add gateway MyGW --config /backup/gateway-config.json

# Bulk restore
ls /backup/*.tar.gz | while read backup; do
    geneos restore --archive "$backup"
done
```

## Best Practices

### Operational Best Practices

1. **Use Protection for Critical Instances**
   ```bash
   geneos protect gateway PROD_GATEWAY
   ```

2. **Regular Backups**
   ```bash
   # Daily backup script
   geneos backup --all --output "/backup/$(date +%Y%m%d)"
   ```

3. **Monitor Log Sizes**
   ```bash
   # Regular cleanup
   geneos clean --all
   ```

4. **Version Management**
   ```bash
   # Keep production and development versions
   geneos package install gateway --base prod --version 6.5.0
   geneos package install gateway --base dev --version 6.6.0
   ```

### Security Best Practices

1. **Use Encrypted Credentials**
   ```bash
   geneos login downloads -u email@example.com
   geneos set gateway MyGW --secure "password"
   ```

2. **Enable TLS**
   ```bash
   geneos tls init
   geneos tls new gateway MyGW
   geneos tls new netprobe MyNP
   ```

3. **Regular Certificate Rotation**
   ```bash
   # Renew certificates before expiry
   geneos tls list | grep -E "(30|60|90) days"
   geneos tls renew gateway MyGW
   ```

### Performance Best Practices

1. **Resource Monitoring**
   ```bash
   # Regular health checks
   geneos ps --long --network
   ```

2. **Log Management**
   ```bash
   # Regular log cleanup
   geneos clean --all
   ```

3. **Efficient Instance Management**
   ```bash
   # Use wildcards for bulk operations
   geneos restart netprobe "prod-np*"
   ```

### Documentation Best Practices

1. **Document Your Environment**
   ```bash
   # Export current configuration
   geneos list --json > geneos-inventory.json
   geneos show --all > geneos-config-backup.json
   ```

2. **Standard Naming Conventions**
   - Use descriptive names: `prod-gateway-east`, `dev-netprobe-01`
   - Include environment indicators: `prod-`, `dev-`, `test-`
   - Use consistent patterns for similar instances

3. **Regular Configuration Reviews**
   ```bash
   # Review instance configurations
   geneos show gateway MyGW | jq '.configuration'
   
   # Check for inconsistencies
   geneos list --json | jq '.[] | select(.flags | contains("D"))'
   ```

---

This guide covers most common `geneos` operations. For additional help:
- Use `geneos help COMMAND` for specific command details
- Check the [User Guide](USER_GUIDE.md) for workflow-oriented examples
- Reference the [Quick Reference](QUICKREFGUIDE.md) for command syntax
- Visit [ITRS Geneos Documentation](https://docs.itrsgroup.com/docs/geneos/) for product-specific information
