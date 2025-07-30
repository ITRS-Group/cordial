# Documentation Flag Corrections

This document summarizes the corrections made to ensure the documentation examples match the actual `geneos` command flags and options.

## Fixed Issues

### 1. **`--tail` Flag → `--lines` Flag**

**Issue**: Several examples used `--tail` flag which doesn't exist in the `logs` command.

**Actual Flag**: `--lines` or `-n`

**Files Fixed**:
- `USER_GUIDE.md`
- `GETTING_STARTED.md`
- `QUICKREFGUIDE.md`

**Examples**:
```bash
# WRONG:
geneos logs --tail 50

# CORRECT:
geneos logs --lines 50
geneos logs -n 50
```

### 2. **`--format json` → `--json`**

**Issue**: Example used `--format json` which doesn't exist in the `ps` command.

**Actual Flag**: `--json` or `-j`

**Files Fixed**:
- `USER_GUIDE.md`

**Examples**:
```bash
# WRONG:
geneos ps --format json

# CORRECT:
geneos ps --json
geneos ps -j
```

### 3. **`--key` → `--privatekey`**

**Issue**: Example used `--key` flag which doesn't exist in the `host add` command.

**Actual Flag**: `--privatekey` or `-i`

**Files Fixed**:
- `USER_GUIDE.md`

**Examples**:
```bash
# WRONG:
geneos host add myserver user@hostname --key ~/.ssh/id_rsa

# CORRECT:
geneos host add myserver ssh://user@hostname --privatekey ~/.ssh/id_rsa
geneos host add myserver ssh://user@hostname -i ~/.ssh/id_rsa
```

## Verified Correct Flags

The following flags were verified to be correct in the documentation:

### Control Commands
- `--force` / `-F` - Correctly used across multiple commands
- `--log` / `-l` - Correctly used in start, restart commands
- `--start` / `-S` - Correctly used in add, deploy commands

### Instance Management
- `--port` / `-p` - Correctly used in add command
- `--gateway` / `-g` - Correctly used for SAN/netprobe gateway connections
- `--env` / `-e` - Correctly used for environment variables
- `--attribute` / `-a` - Correctly used for SAN attributes

### Package Management
- `--username` / `-u` - Correctly used in init demo command
- `--version` / `-V` - Correctly used in package commands
- `--base` / `-b` - Correctly used for base versions

### Host Management
- `--host` / `-H` - Global flag correctly used across commands

## Verification Process

Each correction was verified by:

1. Running `go run main.go help COMMAND` to check actual flags
2. Comparing with documentation examples
3. Testing flag aliases (short vs long form)
4. Verifying command syntax and usage patterns

## Command Reference Sources

All corrections were verified against the actual command help output from:
- `geneos help logs`
- `geneos help ps` 
- `geneos help host add`
- `geneos help start`
- `geneos help add`
- Other relevant commands

This ensures the documentation accurately reflects the current implementation of the `geneos` command.