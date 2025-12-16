# Container Tools MCP Integration Guide

A comprehensive guide for MCP developers to properly integrate their MCP servers into the shared `/opt/container-tools/` infrastructure.

## Overview

Container Tools provides a shared infrastructure where multiple MCP (Model Context Protocol) servers can coexist without interfering with each other. Each MCP:
- Owns its own directory
- Manages its own binaries and content
- Registers hooks through a shared dispatcher
- Merges its configuration into shared mcp.json

This document defines the contracts and patterns all MCPs must follow.

---

## 1. Directory Structure

### Complete Container Tools Layout

```
/opt/container-tools/
├── bin/                              # Symlinks to MCP launchers (Linux/macOS only)
│   ├── todo-mcp → ../todo-mcp/bin/todo-mcp
│   └── p2kb-mcp → ../p2kb-mcp/bin/p2kb-mcp
│
├── etc/
│   ├── mcp.json                      # Shared MCP configuration (ALL MCPs)
│   ├── mcp.json-prior                # Backup before last modification
│   ├── hooks-dispatcher.sh           # Universal hook dispatcher
│   └── hooks.d/                      # Hook scripts directory
│       ├── app-start/
│       │   ├── todo-mcp.sh
│       │   └── p2kb-mcp.sh
│       ├── compact-start/
│       │   └── todo-mcp.sh
│       └── compact-end/
│           └── todo-mcp.sh
│
├── todo-mcp/                         # todo-mcp's territory (EXAMPLE)
│   ├── bin/
│   │   ├── todo-mcp                  # Universal launcher script
│   │   └── platforms/                # Platform-specific binaries
│   │       ├── todo-mcp-v0.6.9.0-darwin-amd64
│   │       ├── todo-mcp-v0.6.9.0-darwin-arm64
│   │       ├── todo-mcp-v0.6.9.0-linux-amd64
│   │       ├── todo-mcp-v0.6.9.0-linux-arm64
│   │       └── todo-mcp-v0.6.9.0-windows-amd64.exe
│   ├── templates/                    # MCP-specific templates
│   ├── content/                      # MCP-specific content
│   ├── backup/
│   │   └── mcp.json-prior            # Copy of mcp.json before our modifications
│   ├── install.sh                    # In-package installer
│   ├── README.md
│   └── VERSION_MANIFEST.txt
│
└── p2kb-mcp/                         # Another MCP's territory
    ├── bin/
    │   ├── p2kb-mcp
    │   └── platforms/
    ├── install.sh
    └── ...
```

### Ownership Rules

| Path | Owner | Can Modify |
|------|-------|------------|
| `/opt/container-tools/bin/` | Shared | Each MCP manages ONLY its own symlink |
| `/opt/container-tools/etc/mcp.json` | Shared | Each MCP MERGES its entry, never replaces |
| `/opt/container-tools/etc/hooks.d/` | Shared | Each MCP manages ONLY its own hook scripts |
| `/opt/container-tools/etc/hooks-dispatcher.sh` | First installer | Created once, never modified |
| `/opt/container-tools/{mcp-name}/` | That MCP | Full ownership - can do anything |

---

## 2. Installation Contract

### What Your Installer MUST Do

1. **Accept target parameter** (default: `/opt/container-tools`)
   ```bash
   TARGET="${1:-/opt/container-tools}"
   ```

2. **Skip installation if already current** (optimization)
   - Detect current platform (os/arch)
   - Find matching binary in source package
   - Find matching binary in destination (if exists)
   - Compare MD5 checksums
   - If identical, print "Already up to date" and exit cleanly
   ```bash
   detect_platform() {
       local os=$(uname -s | tr '[:upper:]' '[:lower:]')
       local arch=$(uname -m)
       case "$arch" in
           x86_64|amd64) arch="amd64" ;;
           aarch64|arm64) arch="arm64" ;;
       esac
       echo "${os}-${arch}"
   }

   PLATFORM=$(detect_platform)
   SOURCE_BIN=$(find "$SCRIPT_DIR/bin/platforms" -name "*-${PLATFORM}" -o -name "*-${PLATFORM}.exe" | head -1)
   DEST_BIN=$(find "$TARGET/{your-mcp}/bin/platforms" -name "*-${PLATFORM}" -o -name "*-${PLATFORM}.exe" 2>/dev/null | head -1)

   if [ -n "$SOURCE_BIN" ] && [ -n "$DEST_BIN" ]; then
       SOURCE_MD5=$(md5sum "$SOURCE_BIN" | awk '{print $1}')
       DEST_MD5=$(md5sum "$DEST_BIN" | awk '{print $1}')
       if [ "$SOURCE_MD5" = "$DEST_MD5" ]; then
           echo "Already up to date (${PLATFORM} binary unchanged)"
           exit 0
       fi
   fi
   ```

3. **Create structure if first-time install**
   ```bash
   mkdir -p "$TARGET/bin"
   mkdir -p "$TARGET/etc/hooks.d"
   mkdir -p "$TARGET/{your-mcp-name}"
   ```

4. **Backup before modifying shared files**
   - Copy `etc/mcp.json` to `{your-mcp}/backup/mcp.json-prior`
   - Only ONE backup depth (overwrite previous backup)

5. **Replace your own directory entirely**
   ```bash
   # Backup existing installation
   if [ -d "$TARGET/{your-mcp}" ]; then
       mv "$TARGET/{your-mcp}" "$TARGET/{your-mcp}-prior"
   fi
   # Install new version
   cp -r ./package-contents "$TARGET/{your-mcp}"
   ```

6. **Create symlink (Linux/macOS only)**
   ```bash
   if [[ "$OSTYPE" != "msys" && "$OSTYPE" != "cygwin" ]]; then
       ln -sf "../{your-mcp}/bin/{your-mcp}" "$TARGET/bin/{your-mcp}"
   fi
   ```

7. **Merge into mcp.json** (see Section 4)

8. **Install hooks** (see Section 3)

9. **Install hooks dispatcher if missing** (see Section 3)

### What Your Installer MUST NOT Do

- Replace `/opt/container-tools/etc/mcp.json` entirely
- Modify or remove other MCPs' directories
- Modify or remove other MCPs' symlinks
- Modify or remove other MCPs' hook scripts
- Modify the hooks-dispatcher.sh (except to create it if missing)

---

## 3. Hooks System

### The Problem

Claude Code's `mcp.json` allows only ONE script per hook type:
```json
"hooks": {
  "app-start": "/single/script/only"
}
```

With multiple MCPs, each needing hooks, this creates conflicts.

### The Solution: hooks.d Dispatcher Pattern

A single dispatcher script runs ALL registered hooks for a given type.

### Hooks Dispatcher Script

Location: `/opt/container-tools/etc/hooks-dispatcher.sh`

```bash
#!/bin/bash
#
# Container Tools Hook Dispatcher
# Runs all hook scripts for a given hook type
#
# Usage: hooks-dispatcher.sh <hook-type>
# Example: hooks-dispatcher.sh app-start
#

set -e

HOOK_TYPE="$1"
HOOKS_DIR="/opt/container-tools/etc/hooks.d/${HOOK_TYPE}"

if [ -z "$HOOK_TYPE" ]; then
    echo "Usage: $0 <hook-type>" >&2
    exit 1
fi

if [ ! -d "$HOOKS_DIR" ]; then
    # No hooks registered for this type - that's okay
    exit 0
fi

# Run all executable scripts in alphabetical order
for script in "$HOOKS_DIR"/*.sh; do
    if [ -f "$script" ] && [ -x "$script" ]; then
        if [ -n "$CONTAINER_TOOLS_DEBUG" ]; then
            echo "[hooks-dispatcher] Running: $script" >&2
        fi

        # Run hook, capture errors but don't stop other hooks
        if ! "$script"; then
            echo "[hooks-dispatcher] Warning: $script failed" >&2
        fi
    fi
done

exit 0
```

### Installing the Dispatcher (First MCP Only)

Your installer should create the dispatcher if it doesn't exist:

```bash
DISPATCHER="$TARGET/etc/hooks-dispatcher.sh"

if [ ! -f "$DISPATCHER" ]; then
    cat > "$DISPATCHER" << 'DISPATCHER_SCRIPT'
#!/bin/bash
# [paste the dispatcher script above]
DISPATCHER_SCRIPT
    chmod +x "$DISPATCHER"
fi
```

### Installing Your Hooks

Each MCP installs its hooks as individual scripts:

```bash
# Create hook directory for this type
mkdir -p "$TARGET/etc/hooks.d/app-start"

# Install your hook script
cat > "$TARGET/etc/hooks.d/app-start/{your-mcp}.sh" << 'HOOK'
#!/bin/bash
# {your-mcp} app-start hook
# Called when Claude Code starts

# Your initialization logic here
echo "[{your-mcp}] Initializing..."
HOOK

chmod +x "$TARGET/etc/hooks.d/app-start/{your-mcp}.sh"
```

### Hook Naming Convention

- Script name: `{your-mcp-name}.sh`
- Examples: `todo-mcp.sh`, `p2kb-mcp.sh`
- Execution order: Alphabetical by filename
- To control order, prefix with numbers: `01-todo-mcp.sh`, `02-p2kb-mcp.sh`

### Known Hook Types

| Hook Type | When Fired | Common Uses |
|-----------|------------|-------------|
| `app-start` | Claude Code starts | Initialize state, check dependencies |
| `compact-start` | Before context compaction | Save important state |
| `compact-end` | After context compaction | Restore state, log event |

---

## 4. mcp.json Configuration

### Structure

```json
{
  "mcpServers": {
    "todo-mcp": {
      "command": "/opt/container-tools/todo-mcp/bin/todo-mcp",
      "args": ["--mode", "stdio"]
    },
    "p2kb-mcp": {
      "command": "/opt/container-tools/p2kb-mcp/bin/p2kb-mcp",
      "args": []
    }
  },
  "hooks": {
    "app-start": "/opt/container-tools/etc/hooks-dispatcher.sh app-start",
    "compact-start": "/opt/container-tools/etc/hooks-dispatcher.sh compact-start",
    "compact-end": "/opt/container-tools/etc/hooks-dispatcher.sh compact-end"
  }
}
```

### Merging Your Entry (with jq)

```bash
MCP_JSON="$TARGET/etc/mcp.json"
YOUR_MCP="your-mcp-name"
YOUR_COMMAND="$TARGET/$YOUR_MCP/bin/$YOUR_MCP"

# Backup first
mkdir -p "$TARGET/$YOUR_MCP/backup"
cp "$MCP_JSON" "$TARGET/$YOUR_MCP/backup/mcp.json-prior"

# Merge your MCP server entry
jq --arg name "$YOUR_MCP" \
   --arg cmd "$YOUR_COMMAND" \
   '.mcpServers[$name] = {"command": $cmd, "args": ["--mode", "stdio"]}' \
   "$MCP_JSON" > "$MCP_JSON.tmp" && mv "$MCP_JSON.tmp" "$MCP_JSON"

# Ensure hooks point to dispatcher
jq '.hooks["app-start"] = "/opt/container-tools/etc/hooks-dispatcher.sh app-start" |
    .hooks["compact-start"] = "/opt/container-tools/etc/hooks-dispatcher.sh compact-start" |
    .hooks["compact-end"] = "/opt/container-tools/etc/hooks-dispatcher.sh compact-end"' \
   "$MCP_JSON" > "$MCP_JSON.tmp" && mv "$MCP_JSON.tmp" "$MCP_JSON"
```

### Merging Without jq (Fallback)

If `jq` is not available, your installer should:

1. Warn the user
2. Provide the exact JSON snippet to add manually
3. Optionally create a new mcp.json if none exists

```bash
if ! command -v jq &> /dev/null; then
    echo "Warning: jq not found. Manual configuration required."
    echo ""
    echo "Add this to $MCP_JSON under 'mcpServers':"
    echo '  "your-mcp": {'
    echo '    "command": "'$YOUR_COMMAND'",'
    echo '    "args": ["--mode", "stdio"]'
    echo '  }'
fi
```

### Creating mcp.json (First-Time Install)

If no mcp.json exists, create the full structure:

```bash
if [ ! -f "$MCP_JSON" ]; then
    cat > "$MCP_JSON" << EOF
{
  "mcpServers": {
    "$YOUR_MCP": {
      "command": "$YOUR_COMMAND",
      "args": ["--mode", "stdio"]
    }
  },
  "hooks": {
    "app-start": "/opt/container-tools/etc/hooks-dispatcher.sh app-start",
    "compact-start": "/opt/container-tools/etc/hooks-dispatcher.sh compact-start",
    "compact-end": "/opt/container-tools/etc/hooks-dispatcher.sh compact-end"
  }
}
EOF
fi
```

---

## 5. Backup Strategy

### Principle: Single Depth with `-prior` Suffix

Only keep ONE backup of each item. The filesystem provides timestamps.

### What to Backup

| Item | Backup Location | When |
|------|-----------------|------|
| Your MCP directory | `{your-mcp}-prior/` | Before replacing with new version |
| Shared mcp.json | `{your-mcp}/backup/mcp.json-prior` | Before modifying |

### Backup Flow

```bash
# 1. Backup mcp.json to YOUR territory
mkdir -p "$TARGET/{your-mcp}/backup"
if [ -f "$TARGET/etc/mcp.json" ]; then
    cp "$TARGET/etc/mcp.json" "$TARGET/{your-mcp}/backup/mcp.json-prior"
fi

# 2. Backup your previous installation
if [ -d "$TARGET/{your-mcp}" ]; then
    rm -rf "$TARGET/{your-mcp}-prior"  # Remove old backup
    mv "$TARGET/{your-mcp}" "$TARGET/{your-mcp}-prior"
fi

# 3. Install new version
cp -r ./new-version "$TARGET/{your-mcp}"
```

### Rollback Procedure

```bash
# Restore previous MCP version
rm -rf "$TARGET/{your-mcp}"
mv "$TARGET/{your-mcp}-prior" "$TARGET/{your-mcp}"

# Restore mcp.json if needed
cp "$TARGET/{your-mcp}/backup/mcp.json-prior" "$TARGET/etc/mcp.json"
```

---

## 6. Uninstallation (Rollback Pattern)

### Philosophy: Uninstall = Rollback

Uninstallation should restore the previous state when possible, not just delete everything.
This allows users to safely try new versions and roll back if issues occur.

### Rollback Behavior

**If a prior installation exists (`{your-mcp}-prior/`):**
1. Restore the prior installation directory
2. Restore the prior mcp.json entry (from `backup/mcp.json-prior`)
3. Remove the current version's hooks, replace with prior's if available

**If no prior exists (first-time install being removed):**
1. Remove the MCP directory entirely
2. Remove the mcp.json entry
3. Remove hooks

### Important: mcp.json Rollback

When rolling back the mcp.json entry, you must NOT replace the entire mcp.json file.
Other MCPs may have been installed since your backup was made. Instead:

1. Read your `backup/mcp.json-prior` file
2. Extract only YOUR MCP's entry from it
3. Merge/replace that single entry into the current mcp.json

This preserves other MCPs' configurations while rolling back only your entry.

### What NOT to Remove

- Other MCPs' directories, symlinks, or hooks
- The hooks-dispatcher.sh (other MCPs may use it)
- The hooks.d directories (other MCPs may have hooks there)
- Other entries in mcp.json

### Uninstall/Rollback Script Pattern

```bash
#!/bin/bash
# Uninstall/Rollback {your-mcp}

TARGET="${1:-/opt/container-tools}"
YOUR_MCP="{your-mcp}"

echo "Uninstalling $YOUR_MCP from $TARGET..."

# Check if we have a prior installation to roll back to
if [ -d "$TARGET/$YOUR_MCP-prior" ]; then
    echo "Prior installation found - performing rollback..."

    # 1. Remove current installation
    rm -rf "$TARGET/$YOUR_MCP"

    # 2. Restore prior installation
    mv "$TARGET/$YOUR_MCP-prior" "$TARGET/$YOUR_MCP"
    echo "Restored prior installation"

    # 3. Rollback mcp.json entry (merge prior entry into current mcp.json)
    PRIOR_MCP_JSON="$TARGET/$YOUR_MCP/backup/mcp.json-prior"
    CURRENT_MCP_JSON="$TARGET/etc/mcp.json"

    if [ -f "$PRIOR_MCP_JSON" ] && command -v jq &> /dev/null; then
        # Extract our entry from the prior mcp.json
        PRIOR_ENTRY=$(jq ".mcpServers[\"$YOUR_MCP\"]" "$PRIOR_MCP_JSON")

        if [ "$PRIOR_ENTRY" != "null" ]; then
            # Merge prior entry into current mcp.json (preserves other MCPs)
            jq --argjson entry "$PRIOR_ENTRY" \
               ".mcpServers[\"$YOUR_MCP\"] = \$entry" \
               "$CURRENT_MCP_JSON" > "/tmp/mcp.json.tmp"
            mv "/tmp/mcp.json.tmp" "$CURRENT_MCP_JSON"
            echo "Rolled back mcp.json entry to prior version"
        fi
    else
        echo "Warning: Could not rollback mcp.json entry (jq not found or no prior backup)"
    fi

    # 4. Update symlink to point to restored version
    rm -f "$TARGET/bin/$YOUR_MCP"
    ln -sf "../$YOUR_MCP/bin/$YOUR_MCP" "$TARGET/bin/$YOUR_MCP"

    echo "Rollback complete - restored prior version"
else
    echo "No prior installation - performing full removal..."

    # Remove your directory
    rm -rf "$TARGET/$YOUR_MCP"

    # Remove your symlink
    rm -f "$TARGET/bin/$YOUR_MCP"

    # Remove your hooks
    find "$TARGET/etc/hooks.d" -name "$YOUR_MCP.sh" -delete 2>/dev/null

    # Remove your entry from mcp.json
    if command -v jq &> /dev/null && [ -f "$TARGET/etc/mcp.json" ]; then
        jq "del(.mcpServers[\"$YOUR_MCP\"])" \
           "$TARGET/etc/mcp.json" > "$TARGET/etc/mcp.json.tmp"
        mv "$TARGET/etc/mcp.json.tmp" "$TARGET/etc/mcp.json"
    else
        echo "Warning: Please manually remove '$YOUR_MCP' from $TARGET/etc/mcp.json"
    fi

    echo "Uninstall complete"
fi
```

---

## 7. Platform Considerations

### Symlinks

| Platform | Create Symlink in bin/? | PATH Recommendation |
|----------|------------------------|---------------------|
| Linux | Yes | Add `/opt/container-tools/bin` to PATH |
| macOS | Yes | Add `/opt/container-tools/bin` to PATH |
| Windows (native) | No | Add `C:\opt\container-tools\{your-mcp}\bin` to PATH |
| Windows (WSL) | Yes (it's Linux) | Add `/opt/container-tools/bin` to PATH |

### Detection

```bash
create_symlink() {
    case "$OSTYPE" in
        msys*|cygwin*|win32*)
            echo "Windows detected - skipping symlink"
            echo "Add $TARGET/{your-mcp}/bin to your PATH"
            return
            ;;
    esac

    ln -sf "../{your-mcp}/bin/{your-mcp}" "$TARGET/bin/{your-mcp}"
    echo "Created symlink: $TARGET/bin/{your-mcp}"
}
```

---

## 8. Installer Template

Complete installer script template for MCP developers.

This template includes:
- **Skip-if-identical**: Compares platform binary MD5 checksums and skips install if unchanged
- **Rollback uninstall**: Restores prior installation when available instead of just deleting

```bash
#!/bin/bash
#
# {your-mcp} installer for container-tools
#
# Usage:
#   ./install.sh [OPTIONS] [target-dir]
#
# Options:
#   --target DIR    Install to DIR (default: /opt/container-tools)
#   --uninstall     Remove/rollback {your-mcp} from container-tools
#   --help          Show this help
#
# Default target: /opt/container-tools
#

set -e

YOUR_MCP="{your-mcp}"
VERSION="{version}"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Parse arguments
UNINSTALL=false
TARGET="/opt/container-tools"

while [[ $# -gt 0 ]]; do
    case $1 in
        --uninstall)
            UNINSTALL=true
            shift
            ;;
        --target)
            TARGET="$2"
            shift 2
            ;;
        --help|-h)
            head -20 "$0" | tail -15
            exit 0
            ;;
        *)
            TARGET="$1"
            shift
            ;;
    esac
done

# Check for sudo if needed
need_sudo() {
    if [ -w "$TARGET" ] 2>/dev/null || [ -w "$(dirname "$TARGET")" ] 2>/dev/null; then
        echo ""
    else
        echo "sudo"
    fi
}
SUDO=$(need_sudo)

# Get script directory (where the package was extracted)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

#
# PLATFORM DETECTION
#
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) arch="unknown" ;;
    esac
    echo "${os}-${arch}"
}

#
# UNINSTALL / ROLLBACK
#
if [ "$UNINSTALL" = true ]; then
    info "Uninstalling $YOUR_MCP from $TARGET..."

    # Check if we have a prior installation to roll back to
    if [ -d "$TARGET/$YOUR_MCP-prior" ]; then
        info "Prior installation found - performing rollback..."

        # 1. Remove current installation
        $SUDO rm -rf "$TARGET/$YOUR_MCP"

        # 2. Restore prior installation
        $SUDO mv "$TARGET/$YOUR_MCP-prior" "$TARGET/$YOUR_MCP"
        info "Restored prior installation"

        # 3. Rollback mcp.json entry (merge prior entry into current mcp.json)
        PRIOR_MCP_JSON="$TARGET/$YOUR_MCP/backup/mcp.json-prior"
        CURRENT_MCP_JSON="$TARGET/etc/mcp.json"

        if [ -f "$PRIOR_MCP_JSON" ] && command -v jq &> /dev/null; then
            # Extract our entry from the prior mcp.json
            PRIOR_ENTRY=$($SUDO jq ".mcpServers[\"$YOUR_MCP\"]" "$PRIOR_MCP_JSON")

            if [ "$PRIOR_ENTRY" != "null" ]; then
                # Merge prior entry into current mcp.json (preserves other MCPs)
                $SUDO jq --argjson entry "$PRIOR_ENTRY" \
                   ".mcpServers[\"$YOUR_MCP\"] = \$entry" \
                   "$CURRENT_MCP_JSON" > "/tmp/mcp.json.tmp"
                $SUDO mv "/tmp/mcp.json.tmp" "$CURRENT_MCP_JSON"
                info "Rolled back mcp.json entry to prior version"
            fi
        else
            warn "Could not rollback mcp.json entry (jq not found or no prior backup)"
        fi

        # 4. Update symlink to point to restored version
        case "$OSTYPE" in
            msys*|cygwin*|win32*) ;;
            *)
                $SUDO rm -f "$TARGET/bin/$YOUR_MCP"
                $SUDO ln -sf "../$YOUR_MCP/bin/$YOUR_MCP" "$TARGET/bin/$YOUR_MCP"
                ;;
        esac

        info "Rollback complete - restored prior version"
    else
        info "No prior installation - performing full removal..."

        # Remove our directories
        $SUDO rm -rf "$TARGET/$YOUR_MCP"

        # Remove our symlink
        $SUDO rm -f "$TARGET/bin/$YOUR_MCP"

        # Remove our hooks
        $SUDO find "$TARGET/etc/hooks.d" -name "$YOUR_MCP.sh" -delete 2>/dev/null || true

        # Remove our entry from mcp.json
        if command -v jq &> /dev/null && [ -f "$TARGET/etc/mcp.json" ]; then
            $SUDO jq "del(.mcpServers[\"$YOUR_MCP\"])" \
               "$TARGET/etc/mcp.json" > "/tmp/mcp.json.tmp"
            $SUDO mv "/tmp/mcp.json.tmp" "$TARGET/etc/mcp.json"
            info "Removed $YOUR_MCP from mcp.json"
        else
            warn "Please manually remove '$YOUR_MCP' from $TARGET/etc/mcp.json"
        fi

        info "Uninstall complete"
    fi
    exit 0
fi

#
# INSTALL
#

# Check if already up to date (skip-if-identical optimization)
PLATFORM=$(detect_platform)
SOURCE_BIN=$(find "$SCRIPT_DIR/bin/platforms" -name "*-${PLATFORM}" -o -name "*-${PLATFORM}.exe" 2>/dev/null | head -1)
DEST_BIN=$(find "$TARGET/$YOUR_MCP/bin/platforms" -name "*-${PLATFORM}" -o -name "*-${PLATFORM}.exe" 2>/dev/null | head -1)

if [ -n "$SOURCE_BIN" ] && [ -n "$DEST_BIN" ]; then
    SOURCE_MD5=$(md5sum "$SOURCE_BIN" 2>/dev/null | awk '{print $1}')
    DEST_MD5=$(md5sum "$DEST_BIN" 2>/dev/null | awk '{print $1}')
    if [ -n "$SOURCE_MD5" ] && [ "$SOURCE_MD5" = "$DEST_MD5" ]; then
        info "Already up to date (${PLATFORM} binary unchanged)"
        exit 0
    fi
fi

echo ""
echo "========================================="
echo -e "${BLUE}Installing $YOUR_MCP v${VERSION}${NC}"
echo "========================================="
echo ""
info "Target: $TARGET"

# 1. Create directory structure if first-time install
if [ ! -d "$TARGET" ]; then
    info "Creating container-tools directory structure..."
    $SUDO mkdir -p "$TARGET/bin"
    $SUDO mkdir -p "$TARGET/etc/hooks.d/app-start"
    $SUDO mkdir -p "$TARGET/etc/hooks.d/compact-start"
    $SUDO mkdir -p "$TARGET/etc/hooks.d/compact-end"
fi

# 2. Backup existing mcp.json to our territory
if [ -f "$TARGET/etc/mcp.json" ]; then
    $SUDO mkdir -p "$TARGET/$YOUR_MCP/backup"
    $SUDO cp "$TARGET/etc/mcp.json" "$TARGET/$YOUR_MCP/backup/mcp.json-prior"
    info "Backed up mcp.json to $YOUR_MCP/backup/"
fi

# 3. Backup existing installation with -prior suffix (depth of 1)
if [ -d "$TARGET/$YOUR_MCP" ]; then
    info "Backing up previous installation..."
    $SUDO rm -rf "$TARGET/$YOUR_MCP-prior"
    $SUDO mv "$TARGET/$YOUR_MCP" "$TARGET/$YOUR_MCP-prior"
fi

# 4. Install MCP directory
info "Installing $YOUR_MCP..."
$SUDO cp -r "$SCRIPT_DIR" "$TARGET/$YOUR_MCP"

# 5. Install hooks dispatcher if missing
if [ ! -f "$TARGET/etc/hooks-dispatcher.sh" ]; then
    info "Installing hooks dispatcher..."
    $SUDO cp "$PACKAGE_ROOT/etc/hooks-dispatcher.sh" "$TARGET/etc/"
    $SUDO chmod +x "$TARGET/etc/hooks-dispatcher.sh"
fi

# 6. Install our hooks
info "Installing hooks..."
$SUDO cp "$PACKAGE_ROOT/etc/hooks.d/app-start/$YOUR_MCP.sh" "$TARGET/etc/hooks.d/app-start/" 2>/dev/null || true
$SUDO chmod +x "$TARGET/etc/hooks.d/app-start/$YOUR_MCP.sh" 2>/dev/null || true

# 7. Create symlink (Linux/macOS only)
case "$OSTYPE" in
    msys*|cygwin*|win32*)
        warn "Windows detected - skipping symlink"
        warn "Add $TARGET/$YOUR_MCP/bin to your PATH"
        ;;
    *)
        $SUDO ln -sf "../$YOUR_MCP/bin/$YOUR_MCP" "$TARGET/bin/$YOUR_MCP"
        info "Created symlink: $TARGET/bin/$YOUR_MCP"
        ;;
esac

# 8. Update mcp.json
MCP_JSON="$TARGET/etc/mcp.json"
YOUR_COMMAND="$TARGET/$YOUR_MCP/bin/$YOUR_MCP"
DISPATCHER="$TARGET/etc/hooks-dispatcher.sh"

if [ ! -f "$MCP_JSON" ]; then
    info "Creating mcp.json..."
    $SUDO tee "$MCP_JSON" > /dev/null << EOF
{
  "mcpServers": {
    "$YOUR_MCP": {
      "command": "$YOUR_COMMAND",
      "args": ["--mode", "stdio"]
    }
  },
  "hooks": {
    "app-start": "$DISPATCHER app-start",
    "compact-start": "$DISPATCHER compact-start",
    "compact-end": "$DISPATCHER compact-end"
  }
}
EOF
elif command -v jq &> /dev/null; then
    info "Merging into mcp.json..."
    $SUDO jq --arg name "$YOUR_MCP" \
       --arg cmd "$YOUR_COMMAND" \
       --arg dispatcher "$DISPATCHER" \
       '.mcpServers[$name] = {"command": $cmd, "args": ["--mode", "stdio"]} |
        .hooks["app-start"] = "\($dispatcher) app-start" |
        .hooks["compact-start"] = "\($dispatcher) compact-start" |
        .hooks["compact-end"] = "\($dispatcher) compact-end"' \
       "$MCP_JSON" > "/tmp/mcp.json.tmp"
    $SUDO mv "/tmp/mcp.json.tmp" "$MCP_JSON"
else
    warn "jq not found - please manually configure mcp.json"
    warn "Add $YOUR_MCP entry pointing to: $YOUR_COMMAND"
fi

# 9. Verify installation
echo ""
info "Verifying installation..."
if [ -x "$TARGET/$YOUR_MCP/bin/$YOUR_MCP" ]; then
    VERSION_OUTPUT=$("$TARGET/$YOUR_MCP/bin/$YOUR_MCP" --version 2>&1 | head -1)
    info "Installed: $VERSION_OUTPUT"
else
    error "Installation verification failed - launcher not executable"
fi

# Summary
echo ""
echo "========================================="
echo -e "${GREEN}Installation complete!${NC}"
echo "========================================="
echo ""
echo "Installed to: $TARGET/$YOUR_MCP/"
echo ""
echo -e "${BLUE}Next steps:${NC}"
case "$OSTYPE" in
    msys*|cygwin*|win32*)
        echo "  1. Add $TARGET/$YOUR_MCP/bin to your PATH"
        ;;
    *)
        echo "  1. Add $TARGET/bin to your PATH (if not already)"
        ;;
esac
echo "  2. Restart Claude Code to load the new MCP"
echo ""
echo -e "${BLUE}Test:${NC}"
echo "  $TARGET/$YOUR_MCP/bin/$YOUR_MCP --version"
echo ""
echo -e "${BLUE}Rollback (if needed):${NC}"
echo "  ./install.sh --uninstall"
echo ""
```

---

## 9. Checklist for MCP Developers

Before releasing your container-tools package:

**Package Structure:**
- [ ] Package extracts to `{your-mcp}/` directory (not nested)
- [ ] Contains `bin/{your-mcp}` universal launcher
- [ ] Contains `bin/platforms/` with versioned binaries
- [ ] Contains `install.sh` following this guide

**Installation Behavior:**
- [ ] Installer accepts `--target` parameter
- [ ] Installer accepts `--uninstall` flag
- [ ] Installer skips if platform binary MD5 matches (already up to date)
- [ ] Installer creates structure on first-time install
- [ ] Installer backs up mcp.json to your backup directory
- [ ] Installer backs up previous installation with `-prior` suffix
- [ ] Installer merges (not replaces) mcp.json
- [ ] Installer creates hooks dispatcher if missing
- [ ] Installer installs hooks to `hooks.d/{type}/{your-mcp}.sh`
- [ ] Installer creates symlink only on Linux/macOS
- [ ] Installer provides Windows PATH instructions

**Uninstall/Rollback Behavior:**
- [ ] Uninstaller restores prior installation if available
- [ ] Uninstaller merges prior mcp.json entry (not full file replace)
- [ ] Uninstaller removes content only when no prior exists
- [ ] Uninstaller removes only your mcp.json entry (not others)

**Testing:**
- [ ] Tested: fresh install (no container-tools exists)
- [ ] Tested: update install (previous version exists)
- [ ] Tested: skip-if-identical (reinstall same version)
- [ ] Tested: rollback (uninstall with prior)
- [ ] Tested: full removal (uninstall without prior)
- [ ] Tested: coexistence (another MCP already installed)

---

## 10. Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2025-01-XX | Initial guide |

---

## Support

- GitHub: https://github.com/ironsheep/todo-mcp
- Issues: https://github.com/ironsheep/todo-mcp/issues
