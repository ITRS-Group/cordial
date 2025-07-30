# `geneos` User Guide

Welcome to the `geneos` command-line tool! This guide will help you get started and become proficient with managing your Geneos environment.

## Table of Contents

- [What is geneos?](#what-is-geneos)
- [Quick Start](#quick-start)
- [Your First 5 Minutes](#your-first-5-minutes)
- [Essential Workflows](#essential-workflows)
- [Common Tasks](#common-tasks)
- [Advanced Usage](#advanced-usage)
- [Troubleshooting](#troubleshooting)
- [Reference](#reference)

## What is geneos?

The `geneos` command-line tool is your one-stop solution for managing ITRS Geneos environments. It replaces older tools like `gatewayctl` and `netprobectl` with a unified, modern interface that makes Geneos administration easier and more intuitive.

### Key Benefits

- **Unified Management**: One tool for all Geneos components (Gateways, Netprobes, Licence Daemons, etc.)
- **Remote Operations**: Manage instances across multiple servers from a single location
- **Automation-Friendly**: Designed for scripting and integration with automation tools
- **Legacy Compatible**: Can work with existing installations and emulate older commands

### What You Can Do

- Install and update Geneos software packages
- Create, configure, and manage instances
- Start, stop, and monitor services
- Handle certificates and secure connections
- Manage configurations and templates
- Deploy across multiple hosts

## Quick Start

### Installation

#### Option 1: Download Pre-built Binary (Recommended)
```bash
# Download the latest version
mkdir -p ${HOME}/bin && cd ${HOME}/bin
curl -OL https://github.com/ITRS-Group/cordial/releases/latest/download/geneos
chmod +x ./geneos

# Add to PATH (add this to your ~/.bashrc or ~/.profile)
export PATH="$HOME/bin:$PATH"
```

#### Option 2: Build from Source
```bash
# Requires Go 1.21.5 or later
go install github.com/itrs-group/cordial/tools/geneos@latest
```

### Verify Installation
```bash
geneos version
geneos --help
```

## Your First 5 Minutes

### 1. Check What's Already There
If you have an existing Geneos installation:

```bash
# Tell geneos where your Geneos installation is (if needed)
geneos config set geneos=/path/to/geneos  # typically /opt/itrs

# List existing instances (safe, read-only)
geneos list

# See what's currently running
geneos ps
```

### 2. Create Your First Demo Environment
If you're starting fresh:

```bash
# Creates a complete demo environment
geneos init demo -u your-email@example.com

# This will:
# - Download required software
# - Create Gateway, Netprobe, and other instances
# - Start everything up
# - Show logs
```

### 3. Basic Control Commands
```bash
# See everything that's configured
geneos list

# See what's running
geneos ps

# Start everything
geneos start

# Follow logs for all components
geneos logs -f

# Stop everything
geneos stop
```

### 4. Get Help Anytime
```bash
# General help
geneos help

# Help for specific commands
geneos help start
geneos help gateway

# Quick reference
geneos help | grep -A 20 "Control Instances"
```

## Essential Workflows

### Starting Your Day: Health Check

```bash
# Quick status overview
geneos ps

# Check if any instances failed to start
geneos list | grep -v "running\|stopped"

# View recent logs for any problems
geneos logs --tail 50
```

### Managing a Gateway

```bash
# Create a new Gateway instance
geneos add gateway MyGateway -p 7039

# Configure it with specific settings
geneos set gateway MyGateway -p 7039 -u geneos

# Start it and watch the logs
geneos start gateway MyGateway -l

# Check its configuration
geneos show gateway MyGateway

# See what command would be run
geneos command gateway MyGateway
```

### Managing Netprobes

```bash
# Add a standard Netprobe
geneos add netprobe MyNetprobe -g localhost:7039

# Add a Self-Announcing Netprobe (SAN)
geneos add san MySAN -g gateway.example.com:7039

# Configure SAN attributes
geneos set san MySAN -a environment=production -a location=datacenter1

# Deploy to a remote host
geneos deploy san MySAN --host remote-server
```

### Software Updates

```bash
# Check available packages
geneos package list

# Install latest Gateway software
geneos package install gateway

# Update all components to latest versions
geneos package update
```

## Common Tasks

### Working with Multiple Instances

```bash
# Use wildcards to match multiple instances
geneos start netprobe "prod*"    # All netprobes starting with "prod"
geneos restart gateway "*test*"  # All gateways with "test" in the name

# Work with specific component types
geneos ps gateways              # Only show gateways
geneos stop san                 # Stop all SANs
geneos logs licd -f             # Follow logs for all licence daemons
```

### Remote Host Management

```bash
# Add a remote host
geneos host add myserver user@hostname --key ~/.ssh/id_rsa

# Deploy an instance to remote host
geneos deploy gateway RemoteGW --host myserver

# Run commands on specific hosts
geneos start --host myserver
geneos ps --host myserver
```

### Configuration Management

```bash
# Set environment variables
geneos set netprobe MyProbe -e JAVA_HOME=/usr/lib/jvm/java-11
geneos set netprobe MyProbe -e "CUSTOM_VAR=value with spaces"

# Remove environment variables
geneos unset netprobe MyProbe -e JAVA_HOME

# Copy configurations between instances
geneos copy gateway SourceGW DestGW

# Backup configurations
geneos backup gateway MyGW

# Import external files (certificates, licenses, etc.)
geneos import licd mylicense.lic
```

### Certificate Management

```bash
# Initialize TLS certificate system
geneos tls init

# Create certificates for secure connections
geneos tls new gateway MyGW
geneos tls new netprobe MyProbe

# List all certificates
geneos tls list

# Sync certificates across hosts
geneos tls sync
```

## Advanced Usage

### Instance Protection

Protect critical instances from accidental changes:

```bash
# Protect an instance
geneos protect gateway CRITICAL_GW

# Try to stop it (will fail)
geneos stop gateway CRITICAL_GW

# Override protection when needed
geneos stop gateway CRITICAL_GW --force
```

### Template-Based Configuration

```bash
# Rebuild configuration from templates
geneos rebuild san MySAN

# Deploy with templates
geneos deploy san NewSAN --template production
```

### Automation and Scripting

```bash
# Non-interactive mode for scripts
export GENEOS_NON_INTERACTIVE=true

# JSON output for parsing
geneos ps --format json

# Exit codes for automation
if geneos ps gateway MyGW > /dev/null 2>&1; then
    echo "Gateway is running"
else
    echo "Gateway is not running"
    geneos start gateway MyGW
fi
```

### Legacy Migration

If you're migrating from older tools:

```bash
# Create symlinks for legacy commands
geneos migrate -X

# This creates links so old commands work:
# gatewayctl -> geneos
# netprobectl -> geneos
# etc.

# Revert if needed
geneos revert -X
```

## Troubleshooting

### Common Issues

#### "geneos: command not found"
**Solution**: Ensure the binary is in your PATH
```bash
# Check if geneos is accessible
which geneos

# If not, add to PATH
export PATH="$HOME/bin:$PATH"

# Make permanent by adding to ~/.bashrc
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc
```

#### "Cannot find Geneos installation"
**Solution**: Configure the base directory
```bash
# Set the geneos base directory
geneos config set geneos=/opt/itrs

# Or set environment variable
export ITRS_HOME=/opt/itrs
```

#### Instance won't start
**Solution**: Check logs and configuration
```bash
# View recent logs
geneos logs instance_name --tail 100

# Check configuration
geneos show instance_name

# Verify the command that would be run
geneos command instance_name

# Check if ports are available
netstat -ln | grep :7039
```

#### Permission denied errors
**Solution**: Check file permissions and ownership
```bash
# Check instance directory permissions
ls -la $(geneos home gateway MyGW)

# Fix ownership if needed (as root)
chown -R geneos:geneos /opt/itrs/gateway/gateways/MyGW
```

### Debugging Commands

```bash
# Verbose output
geneos -v start gateway MyGW

# Show what command would be executed
geneos command gateway MyGW

# Check configuration syntax
geneos show gateway MyGW --raw

# Test connectivity to remote hosts
geneos host show myserver
```

### Getting Help

```bash
# Command-specific help
geneos help [command]

# Component-specific help
geneos gateway help

# List all commands
geneos help | grep -E "^  \w+"

# Show examples
geneos help | grep -A 10 Examples
```

## Reference

### Command Structure

Most commands follow this pattern:
```
geneos COMMAND [flags] [TYPE] [NAMES...]
```

- **COMMAND**: What to do (start, stop, list, etc.)
- **flags**: Options like `-p` for port, `-l` for logs
- **TYPE**: Component type (gateway, netprobe, san, etc.)
- **NAMES**: Instance names, can use wildcards

### Instance Naming

Instances can be referenced as:
- `NAME` - Local instance
- `NAME@HOST` - Instance on specific host
- `TYPE:NAME` - Specific type (useful for SANs/Floating Netprobes)

### Useful Aliases

Many commands have shorter aliases:
- `geneos ls` = `geneos list`
- `geneos ps` = `geneos status`
- `geneos cp` = `geneos copy`
- `geneos mv` = `geneos move`
- `geneos rm` = `geneos delete`

### Environment Variables

Key environment variables:
- `ITRS_HOME` - Base Geneos directory
- `GENEOS_NON_INTERACTIVE` - Disable prompts for automation
- `PATH` - Must include geneos binary location

### File Locations

Default locations:
- Config: `~/.config/geneos.json`
- Instances: `$ITRS_HOME/TYPE/TYPEs/INSTANCE/`
- Packages: `$ITRS_HOME/packages/`
- Logs: `$ITRS_HOME/TYPE/TYPEs/INSTANCE/logs/`

## Next Steps

1. **Explore the Examples**: Try the commands in this guide with your environment
2. **Read Component-Specific Help**: Use `geneos gateway help`, `geneos netprobe help`, etc.
3. **Set Up Automation**: Start incorporating geneos into your operational scripts
4. **Configure Remote Hosts**: Extend management to your entire infrastructure
5. **Implement Security**: Set up TLS certificates for secure communications

For detailed command reference, see:
- [Complete Command Reference](docs/geneos.md)
- [Usage Examples](USAGE.md)
- [Quick Reference Card](QUICKREFGUIDE.md)

---

**Need More Help?**
- Run `geneos help` for built-in assistance
- Check the logs with `geneos logs -f`
- Visit the [ITRS Geneos Documentation](https://docs.itrsgroup.com/docs/geneos/)