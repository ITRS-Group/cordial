# Getting Started with `geneos`

This guide gets you up and running with the `geneos` command in under 10 minutes.

## What is `geneos`?

`geneos` is a modern command-line tool that replaces older Geneos management scripts (`gatewayctl`, `netprobectl`, etc.) with a single, unified interface for managing your entire Geneos environment.

## Installation (2 minutes)

### Option 1: Download Binary (Recommended)
```bash
# Download and install
mkdir -p ${HOME}/bin
curl -OL https://github.com/ITRS-Group/cordial/releases/latest/download/geneos
chmod +x geneos
mv geneos ${HOME}/bin/

# Add to your PATH (choose one)
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc && source ~/.bashrc  # Bash
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc    # Zsh
```

### Option 2: Build from Source
```bash
go install github.com/itrs-group/cordial/tools/geneos@latest
```

### Verify
```bash
geneos version
```

## First Time Setup (3 minutes)

### If You Have Existing Geneos
```bash
# Point geneos to your installation
geneos config set geneos=/opt/itrs  # or wherever your Geneos is installed

# Check what's already there
geneos list    # See all configured instances  
geneos ps      # See what's currently running
```

### If You're Starting Fresh
```bash
# Create a demo environment (requires ITRS account)
geneos init demo -u your-email@example.com

# This creates and starts a complete Geneos environment
```

## Essential Commands (5 minutes)

### Viewing Your Environment
```bash
geneos list              # Show all instances
geneos ps                # Show running instances only
geneos ps gateways       # Show only gateways
geneos logs -f           # Follow logs for all components
```

### Starting and Stopping
```bash
geneos start             # Start all instances
geneos stop              # Stop all instances
geneos restart           # Restart all instances

geneos start gateway MyGW    # Start specific instance
geneos stop netprobe "prod*" # Stop all netprobes matching pattern
```

### Getting Information
```bash
geneos show gateway MyGW     # Show configuration
geneos command gateway MyGW  # Show startup command
geneos home gateway MyGW     # Show instance directory
```

### Getting Help
```bash
geneos help              # Main help
geneos help start        # Help for start command
geneos gateway help      # Help for gateway component
```

## Common Workflows

### Daily Health Check
```bash
geneos ps                           # Are my services running?
geneos logs --tail 20              # Any recent errors?
```

### Adding a New Gateway
```bash
geneos add gateway MyNewGW -p 7040  # Create with port 7040
geneos start gateway MyNewGW -l     # Start and follow logs
```

### Adding a Netprobe
```bash
geneos add netprobe MyProbe -g localhost:7039  # Connect to gateway
geneos start netprobe MyProbe -l               # Start and follow logs
```

### Software Updates
```bash
geneos package list                 # See available software
geneos package install gateway     # Install latest gateway
geneos package update              # Update all components
```

## What's Next?

- **Learn More**: Read the [User Guide](USER_GUIDE.md) for detailed workflows
- **Get Help**: Use `geneos help` or `geneos help COMMAND` anytime
- **Advanced Features**: Explore remote management with `geneos host help`
- **Automation**: Check out scripting examples in the [Usage Guide](USAGE.md)

## Quick Reference

| Task | Command |
|------|---------|
| List everything | `geneos list` |
| Show running only | `geneos ps` |
| Start all | `geneos start` |
| Stop all | `geneos stop` |
| Follow logs | `geneos logs -f` |
| Get help | `geneos help` |
| Add gateway | `geneos add gateway NAME -p PORT` |
| Add netprobe | `geneos add netprobe NAME -g HOST:PORT` |

## Troubleshooting

**Command not found?**
- Check your PATH: `echo $PATH`
- Verify installation: `which geneos`

**Can't find Geneos installation?**
- Set the path: `geneos config set geneos=/path/to/geneos`

**Instance won't start?**
- Check logs: `geneos logs INSTANCE_NAME`
- Verify config: `geneos show INSTANCE_NAME`

**Need more help?**
- Run `geneos help`
- Check the [User Guide](USER_GUIDE.md)
- View [Troubleshooting section](USER_GUIDE.md#troubleshooting)