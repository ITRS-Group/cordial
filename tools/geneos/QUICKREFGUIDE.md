# `geneos` Quick Reference Guide

This is your cheat sheet for the most common `geneos` commands.

## Essential Commands

### Viewing Your Environment
| Command | Description | Example |
|---------|-------------|---------|
| `geneos list` / `geneos ls` | Show all configured instances | `geneos ls gateways` |
| `geneos ps` | Show all running instances | `geneos ps 'prod*'` |
| `geneos logs` | View instance logs | `geneos logs -f gateway MyGW` |
| `geneos show` | Show instance configuration | `geneos show gateway MyGW` |

### Control Commands
| Command | Description | Example |
|---------|-------------|---------|
| `geneos start` | Start instances | `geneos start gateway MyGW -l` |
| `geneos stop` | Stop instances | `geneos stop netprobe "test*"` |
| `geneos restart` | Restart instances | `geneos restart san -F` |
| `geneos reload` | Reload configurations | `geneos reload gateway MyGW` |

### Instance Management
| Command | Description | Example |
|---------|-------------|---------|
| `geneos add` | Create new instance | `geneos add gateway MyGW -p 7039` |
| `geneos copy` | Copy instance | `geneos copy gateway MyGW MyGW2` |
| `geneos delete` | Delete instance | `geneos delete gateway MyGW` |
| `geneos protect` | Protect from changes | `geneos protect gateway MyGW` |

## Quick Workflows

### New Gateway Setup
```bash
geneos add gateway MyGW -p 7039       # Create with port 7039
geneos start gateway MyGW -l          # Start and follow logs
geneos show gateway MyGW              # Verify configuration
```

### New Netprobe Setup  
```bash
geneos add netprobe MyProbe -g localhost:7039  # Connect to gateway
geneos start netprobe MyProbe -l               # Start and follow logs
```

### Daily Health Check
```bash
geneos ps                             # What's running?
geneos logs --tail 20                # Recent activity
geneos package list                  # Software status
```

### Software Management
```bash
geneos package list                  # Available packages
geneos package install gateway      # Install latest gateway
geneos package update               # Update all packages
```

## Common Patterns

### Using Wildcards
```bash
geneos start "prod*"                 # All instances starting with "prod"
geneos stop "*test*"                 # All instances containing "test"
geneos ps gateways                   # All gateways only
```

### Remote Operations
```bash
geneos ps --host myserver           # Status on specific host
geneos start --host myserver        # Start all on specific host
geneos deploy gateway MyGW --host myserver  # Deploy to remote host
```

### Following Logs
```bash
geneos logs -f                       # Follow all logs
geneos logs -f gateway              # Follow all gateway logs
geneos logs -f gateway MyGW         # Follow specific gateway
geneos start gateway MyGW -l        # Start and immediately follow logs
```

## Useful Flags

| Flag | Description | Example |
|------|-------------|---------|
| `-f` | Follow/watch logs | `geneos logs -f` |
| `-l` | Show logs after action | `geneos start MyGW -l` |
| `-F` | Force (override protection) | `geneos stop MyGW -F` |
| `-p PORT` | Set port | `geneos add gateway MyGW -p 7039` |
| `-g HOST:PORT` | Set gateway | `geneos add netprobe MyProbe -g host:7039` |
| `--host HOST` | Target specific host | `geneos ps --host myserver` |

## Command Aliases

| Long Form | Alias | Description |
|-----------|-------|-------------|
| `geneos list` | `geneos ls` | List instances |
| `geneos ps` | `geneos status` | Show running instances |
| `geneos copy` | `geneos cp` | Copy instances |
| `geneos move` | `geneos mv` | Move/rename instances |
| `geneos delete` | `geneos rm` | Delete instances |
| `geneos logs` | `geneos log` | View logs |

## Getting Help

| Command | Description |
|---------|-------------|
| `geneos help` | Main help screen |
| `geneos help COMMAND` | Help for specific command |
| `geneos COMPONENT help` | Component-specific help |
| `geneos help \| grep TERM` | Search help for term |

## Subsystems

| Subsystem | Description | Quick Command |
|-----------|-------------|---------------|
| `aes` | Encryption/passwords | `geneos aes new` |
| `config` | Program configuration | `geneos config show` |
| `host` | Remote host management | `geneos host list` |
| `init` | Environment setup | `geneos init demo` |
| `package` | Software management | `geneos package list` |
| `tls` | Certificate management | `geneos tls list` |

## Emergency Commands

```bash
geneos ps                            # What's running?
geneos stop --force                 # Emergency stop all
geneos logs --tail 100              # Last 100 log lines
geneos command gateway MyGW         # Show startup command
geneos show gateway MyGW --raw      # Raw configuration
```

---

💡 **Tip**: Use quotes around wildcards to prevent shell expansion: `geneos ps 'prod*'`  
🔧 **Pro Tip**: Add `-l` to start commands to immediately see logs: `geneos start MyGW -l`  
📖 **More Help**: For detailed examples, see the [User Guide](USER_GUIDE.md)

