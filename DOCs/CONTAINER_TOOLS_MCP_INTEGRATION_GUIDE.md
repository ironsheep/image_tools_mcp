# Container Tools MCP Integration Guide

A comprehensive guide for MCP developers to properly integrate their MCP servers into the shared `/opt/container-tools/` infrastructure.

## Overview

Container Tools provides a shared infrastructure where multiple MCP (Model Context Protocol) servers can coexist without interfering with each other. Each MCP:
- Owns its own directory
- Manages its own binaries and content
- Registers its MCP server in shared `mcp.json`
- Installs hooks in user's `settings.json`
- Optionally provides slash commands via `.claude/commands/`

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
│   └── mcp.json                      # Shared MCP server configuration (ALL MCPs)
│
├── todo-mcp/                         # todo-mcp's territory (EXAMPLE)
│   ├── bin/
│   │   ├── todo-mcp                  # Universal launcher script
│   │   └── platforms/                # Platform-specific binaries
│   │       ├── todo-mcp-v1.0.0-darwin-amd64
│   │       ├── todo-mcp-v1.0.0-darwin-arm64
│   │       ├── todo-mcp-v1.0.0-linux-amd64
│   │       ├── todo-mcp-v1.0.0-linux-arm64
│   │       └── todo-mcp-v1.0.0-windows-amd64.exe
│   ├── hooks/                        # Hook scripts for Claude Code
│   │   └── session-start.sh          # Runs on SessionStart
│   ├── .claude/
│   │   └── commands/                 # Slash commands (copied to user projects)
│   │       ├── tmcp.md
│   │       ├── tmcp-status.md
│   │       └── tmcp-init-mastery.md
│   ├── content/                      # Mastery documentation, templates
│   ├── backup/                       # All backup/rollback data
│   │   ├── mcp.json-prior            # Copy of mcp.json before our modifications
│   │   ├── settings.json-prior       # Copy of settings.json before our modifications
│   │   └── prior/                    # Prior installation (for rollback)
│   ├── install.sh                    # In-package installer
│   ├── CHANGELOG.md                  # Version history
│   ├── LICENSE                       # License file
│   ├── README.md                     # Documentation
│   └── VERSION_MANIFEST.txt          # Version and build info
│
└── p2kb-mcp/                         # Another MCP's territory
    ├── bin/
    ├── hooks/
    ├── install.sh
    └── ...
```

### Ownership Rules

| Path | Owner | Can Modify |
|------|-------|------------|
| `/opt/container-tools/bin/` | Shared | Each MCP manages ONLY its own symlink |
| `/opt/container-tools/etc/mcp.json` | Shared | Each MCP MERGES its entry, never replaces |
| `~/.claude/settings.json` | User | Each MCP MERGES its hooks, never replaces |
| `/opt/container-tools/{mcp-name}/` | That MCP | Full ownership - can do anything |

---

## 2. Configuration Files

### Understanding the Two Config Files

Claude Code uses **two separate configuration files** for different purposes:

| File | Purpose | Location | Contents |
|------|---------|----------|----------|
| `mcp.json` | MCP server definitions | `~/.claude/mcp.json` or project | Server commands, args |
| `settings.json` | User preferences, hooks | `~/.claude/settings.json` or `.claude/settings.json` | Hooks, permissions, preferences |

**Critical:** Never confuse these. MCP servers go in `mcp.json`. Hooks go in `settings.json`.

### mcp.json Structure

```json
{
  "mcpServers": {
    "todo-mcp": {
      "command": "/opt/container-tools/todo-mcp/bin/todo-mcp",
      "args": ["--mode", "stdio"]
    },
    "p2kb-mcp": {
      "command": "/opt/container-tools/p2kb-mcp/bin/p2kb-mcp",
      "args": ["--mode", "stdio"]
    }
  }
}
```

### settings.json Structure (Hooks)

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "/opt/container-tools/todo-mcp/hooks/session-start.sh"
          }
        ]
      },
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "/opt/container-tools/p2kb-mcp/hooks/session-start.sh"
          }
        ]
      }
    ]
  }
}
```

---

## 3. Hooks System

### How Claude Code Hooks Work

Claude Code hooks are shell commands that execute at specific lifecycle points. They are configured in `settings.json` (NOT `mcp.json`).

### Hook Event Types

| Event | When Fired | Common Uses |
|-------|------------|-------------|
| `SessionStart` | Claude Code starts or resumes | Initialize state, inject context |
| `PreToolUse` | Before a tool call | Validate, log, block operations |
| `PostToolUse` | After a tool call | Format output, cleanup |
| `PreCompact` | Before context compaction | Save important state |
| `Notification` | When notification is sent | Custom notifications |
| `Stop` | Claude finishes responding | Cleanup, finalization |
| `UserPromptSubmit` | User submits a prompt | Pre-process input |

### Hook Configuration Schema

```json
{
  "hooks": {
    "EVENT_TYPE": [
      {
        "matcher": "TOOL_PATTERN",
        "hooks": [
          {
            "type": "command",
            "command": "/path/to/your/script.sh"
          }
        ]
      }
    ]
  }
}
```

- **matcher**: Tool name pattern (empty string `""` matches all, or use `"Bash"`, `"Edit|Write"`, etc.)
- **type**: Always `"command"` for shell scripts
- **command**: Full path to executable script

### Multi-MCP Coexistence

Claude Code natively supports multiple hooks per event type via arrays:

```json
{
  "hooks": {
    "SessionStart": [
      { "matcher": "", "hooks": [{ "type": "command", "command": "/opt/container-tools/todo-mcp/hooks/session-start.sh" }] },
      { "matcher": "", "hooks": [{ "type": "command", "command": "/opt/container-tools/other-mcp/hooks/session-start.sh" }] }
    ]
  }
}
```

Each MCP adds its entry to the array. All hooks run independently - no dispatcher needed.

### Hook Script Requirements

Hook scripts must:
1. Be executable (`chmod +x`)
2. Use absolute paths (no `~` or relative paths)
3. Exit cleanly (exit 0 for success)
4. Handle missing dependencies gracefully

For `PreToolUse` hooks that can block:
- Exit 0: Allow the operation
- Exit 2: Block the operation (shows hook's stdout to user)

### Example Hook Script

```bash
#!/bin/bash
#
# todo-mcp SessionStart hook
# Provides Claude with mastery context on session start
#

# Output is visible to Claude in the session
cat << 'EOF'
[todo-mcp] Session initialized.
Reminder: Use mcp__todo-mcp__context_resume to recover prior session state.
For mastery documentation, run /tmcp-init-mastery in your project.
EOF

exit 0
```

---

## 4. Installation Contract

### What Your Installer MUST Do

1. **Accept target parameter** (default: `/opt/container-tools`)
   ```bash
   TARGET="${1:-/opt/container-tools}"
   ```

2. **Skip installation if already current** (optimization)
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
   mkdir -p "$TARGET/etc"
   mkdir -p "$TARGET/{your-mcp-name}"
   ```

4. **Backup existing installation**
   ```bash
   if [ -d "$TARGET/{your-mcp}" ]; then
       # Backup mcp.json and settings.json
       mkdir -p "$TARGET/{your-mcp}/backup"
       cp "$TARGET/etc/mcp.json" "$TARGET/{your-mcp}/backup/mcp.json-prior" 2>/dev/null || true
       cp "$HOME/.claude/settings.json" "$TARGET/{your-mcp}/backup/settings.json-prior" 2>/dev/null || true

       # Move existing to temp
       mv "$TARGET/{your-mcp}" "/tmp/{your-mcp}-prior"
   fi

   # Install new version
   cp -r ./package-contents "$TARGET/{your-mcp}"

   # Move prior into backup/prior/
   if [ -d "/tmp/{your-mcp}-prior" ]; then
       mkdir -p "$TARGET/{your-mcp}/backup"
       mv "/tmp/{your-mcp}-prior" "$TARGET/{your-mcp}/backup/prior"
   fi
   ```

5. **Create symlink** (Linux/macOS only)
   ```bash
   if [[ "$OSTYPE" != "msys" && "$OSTYPE" != "cygwin" ]]; then
       ln -sf "../{your-mcp}/bin/{your-mcp}" "$TARGET/bin/{your-mcp}"
   fi
   ```

6. **Merge into mcp.json** (see Section 5)

7. **Install hooks into settings.json** (see Section 6)

### What Your Installer MUST NOT Do

- Replace `mcp.json` or `settings.json` entirely
- Modify or remove other MCPs' directories
- Modify or remove other MCPs' symlinks
- Remove other MCPs' entries from configuration files

---

## 5. mcp.json Management

### Merging Your MCP Entry

```bash
MCP_JSON="$TARGET/etc/mcp.json"
YOUR_MCP="your-mcp-name"
YOUR_COMMAND="$TARGET/$YOUR_MCP/bin/$YOUR_MCP"

# Create if doesn't exist
if [ ! -f "$MCP_JSON" ]; then
    mkdir -p "$(dirname "$MCP_JSON")"
    echo '{"mcpServers":{}}' > "$MCP_JSON"
fi

# Backup first
mkdir -p "$TARGET/$YOUR_MCP/backup"
cp "$MCP_JSON" "$TARGET/$YOUR_MCP/backup/mcp.json-prior"

# Merge your entry (preserves other MCPs)
jq --arg name "$YOUR_MCP" \
   --arg cmd "$YOUR_COMMAND" \
   '.mcpServers[$name] = {"command": $cmd, "args": ["--mode", "stdio"]}' \
   "$MCP_JSON" > "$MCP_JSON.tmp" && mv "$MCP_JSON.tmp" "$MCP_JSON"
```

### Removing Your MCP Entry (Uninstall)

```bash
jq --arg name "$YOUR_MCP" 'del(.mcpServers[$name])' \
   "$MCP_JSON" > "$MCP_JSON.tmp" && mv "$MCP_JSON.tmp" "$MCP_JSON"
```

---

## 6. settings.json Hook Management

### Installing Hooks

```bash
SETTINGS="$HOME/.claude/settings.json"
YOUR_MCP="your-mcp-name"
HOOK_CMD="$TARGET/$YOUR_MCP/hooks/session-start.sh"

# Create settings.json if doesn't exist
if [ ! -f "$SETTINGS" ]; then
    mkdir -p "$(dirname "$SETTINGS")"
    echo '{}' > "$SETTINGS"
fi

# Backup first
cp "$SETTINGS" "$TARGET/$YOUR_MCP/backup/settings.json-prior"

# Add hook if not already present (check by command path)
jq --arg cmd "$HOOK_CMD" '
  # Ensure hooks.SessionStart exists as array
  .hooks.SessionStart //= [] |
  # Check if our hook already exists
  if (.hooks.SessionStart | map(select(.hooks[]?.command == $cmd)) | length) == 0
  then
    # Add our hook entry
    .hooks.SessionStart += [{"matcher": "", "hooks": [{"type": "command", "command": $cmd}]}]
  else
    # Already exists, no change
    .
  end
' "$SETTINGS" > "$SETTINGS.tmp" && mv "$SETTINGS.tmp" "$SETTINGS"
```

### Removing Hooks (Uninstall)

Remove hooks by matching the MCP's path prefix:

```bash
SETTINGS="$HOME/.claude/settings.json"
MCP_PATH="$TARGET/$YOUR_MCP"

jq --arg path "$MCP_PATH" '
  # For each hook event type
  .hooks |= (
    to_entries | map(
      # Filter out entries where command contains our path
      .value |= map(select(.hooks | all(.command | contains($path) | not)))
    ) | from_entries |
    # Remove empty arrays
    with_entries(select(.value | length > 0))
  ) |
  # Remove hooks key entirely if empty
  if .hooks == {} then del(.hooks) else . end
' "$SETTINGS" > "$SETTINGS.tmp" && mv "$SETTINGS.tmp" "$SETTINGS"
```

### Adding Multiple Hook Types

If your MCP needs multiple hooks:

```bash
# Install SessionStart hook
install_hook "SessionStart" "$TARGET/$YOUR_MCP/hooks/session-start.sh"

# Install PreCompact hook
install_hook "PreCompact" "$TARGET/$YOUR_MCP/hooks/pre-compact.sh"

# Helper function
install_hook() {
    local event_type="$1"
    local hook_cmd="$2"

    jq --arg event "$event_type" --arg cmd "$hook_cmd" '
      .hooks[$event] //= [] |
      if (.hooks[$event] | map(select(.hooks[]?.command == $cmd)) | length) == 0
      then .hooks[$event] += [{"matcher": "", "hooks": [{"type": "command", "command": $cmd}]}]
      else .
      end
    ' "$SETTINGS" > "$SETTINGS.tmp" && mv "$SETTINGS.tmp" "$SETTINGS"
}
```

---

## 7. Backup Strategy

### Principle: All Backups Inside `backup/`

All backup data lives inside `{your-mcp}/backup/`:
- `backup/mcp.json-prior` - snapshot of mcp.json before modification
- `backup/settings.json-prior` - snapshot of settings.json before modification
- `backup/prior/` - complete prior installation (for rollback)

Only keep ONE backup depth.

### What to Backup

| Item | Backup Location | When |
|------|-----------------|------|
| Shared mcp.json | `{your-mcp}/backup/mcp.json-prior` | Before modifying |
| User settings.json | `{your-mcp}/backup/settings.json-prior` | Before modifying |
| Your MCP directory | `{your-mcp}/backup/prior/` | Before replacing |

---

## 8. Uninstallation (Rollback Pattern)

### Philosophy: Uninstall = Rollback When Possible

Uninstallation should restore the previous state when possible.

### Rollback Behavior

**If a prior installation exists (`{your-mcp}/backup/prior/`):**
1. Move prior from `backup/prior/` to temp
2. Remove current installation
3. Restore prior from temp
4. Restore prior mcp.json entry (merge, not replace)
5. Restore prior hooks (merge, not replace)
6. Update symlink

**If no prior exists (full removal):**
1. Remove the MCP directory
2. Remove mcp.json entry
3. Remove hooks from settings.json
4. Remove symlink

### Important: Configuration Rollback

When rolling back configuration entries, you must NOT replace entire files.
Other MCPs may have been installed since your backup. Instead:

1. Read your backup file
2. Extract only YOUR entry from it
3. Merge/replace that single entry into the current file

---

## 9. Platform Considerations

### Symlinks

| Platform | Create Symlink? | PATH Recommendation |
|----------|-----------------|---------------------|
| Linux | Yes | Add `/opt/container-tools/bin` to PATH |
| macOS | Yes | Add `/opt/container-tools/bin` to PATH |
| Windows (native) | No | Add `C:\opt\container-tools\{your-mcp}\bin` to PATH |
| Windows (WSL) | Yes | Add `/opt/container-tools/bin` to PATH |

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
}
```

---

## 10. Installer Template

Complete installer script template for MCP developers.

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
        --uninstall) UNINSTALL=true; shift ;;
        --target) TARGET="$2"; shift 2 ;;
        --help|-h) head -20 "$0" | tail -15; exit 0 ;;
        *) TARGET="$1"; shift ;;
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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SETTINGS="$HOME/.claude/settings.json"

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
# HOOK MANAGEMENT
#
install_hook() {
    local event_type="$1"
    local hook_cmd="$2"

    # Create settings.json if needed
    if [ ! -f "$SETTINGS" ]; then
        mkdir -p "$(dirname "$SETTINGS")"
        echo '{}' > "$SETTINGS"
    fi

    # Add hook if not present
    jq --arg event "$event_type" --arg cmd "$hook_cmd" '
      .hooks[$event] //= [] |
      if (.hooks[$event] | map(select(.hooks[]?.command == $cmd)) | length) == 0
      then .hooks[$event] += [{"matcher": "", "hooks": [{"type": "command", "command": $cmd}]}]
      else .
      end
    ' "$SETTINGS" > "$SETTINGS.tmp" && mv "$SETTINGS.tmp" "$SETTINGS"
}

remove_mcp_hooks() {
    local mcp_path="$1"

    [ ! -f "$SETTINGS" ] && return

    jq --arg path "$mcp_path" '
      .hooks //= {} |
      .hooks |= (
        to_entries | map(
          .value |= map(select(.hooks | all(.command | contains($path) | not)))
        ) | from_entries |
        with_entries(select(.value | length > 0))
      ) |
      if .hooks == {} then del(.hooks) else . end
    ' "$SETTINGS" > "$SETTINGS.tmp" && mv "$SETTINGS.tmp" "$SETTINGS"
}

#
# UNINSTALL / ROLLBACK
#
if [ "$UNINSTALL" = true ]; then
    info "Uninstalling $YOUR_MCP from $TARGET..."

    if [ -d "$TARGET/$YOUR_MCP/backup/prior" ]; then
        info "Prior installation found - performing rollback..."

        # 1. Move prior to temp
        $SUDO mv "$TARGET/$YOUR_MCP/backup/prior" "/tmp/$YOUR_MCP-restore"

        # 2. Remove current hooks before removing installation
        remove_mcp_hooks "$TARGET/$YOUR_MCP"

        # 3. Remove current installation
        $SUDO rm -rf "$TARGET/$YOUR_MCP"

        # 4. Restore prior installation
        $SUDO mv "/tmp/$YOUR_MCP-restore" "$TARGET/$YOUR_MCP"
        info "Restored prior installation"

        # 5. Rollback mcp.json entry
        PRIOR_MCP_JSON="$TARGET/$YOUR_MCP/backup/mcp.json-prior"
        CURRENT_MCP_JSON="$TARGET/etc/mcp.json"

        if [ -f "$PRIOR_MCP_JSON" ] && command -v jq &> /dev/null; then
            PRIOR_ENTRY=$($SUDO cat "$PRIOR_MCP_JSON" | jq ".mcpServers[\"$YOUR_MCP\"]")
            if [ "$PRIOR_ENTRY" != "null" ]; then
                $SUDO jq --argjson entry "$PRIOR_ENTRY" \
                   ".mcpServers[\"$YOUR_MCP\"] = \$entry" \
                   "$CURRENT_MCP_JSON" > "/tmp/mcp.json.tmp"
                $SUDO mv "/tmp/mcp.json.tmp" "$CURRENT_MCP_JSON"
                info "Rolled back mcp.json entry"
            fi
        fi

        # 6. Re-install prior version's hooks
        if [ -x "$TARGET/$YOUR_MCP/hooks/session-start.sh" ]; then
            install_hook "SessionStart" "$TARGET/$YOUR_MCP/hooks/session-start.sh"
            info "Restored prior hooks"
        fi

        # 7. Update symlink
        case "$OSTYPE" in
            msys*|cygwin*|win32*) ;;
            *)
                $SUDO rm -f "$TARGET/bin/$YOUR_MCP"
                $SUDO ln -sf "../$YOUR_MCP/bin/$YOUR_MCP" "$TARGET/bin/$YOUR_MCP"
                ;;
        esac

        info "Rollback complete"
    else
        info "No prior installation - performing full removal..."

        # Remove hooks first
        remove_mcp_hooks "$TARGET/$YOUR_MCP"

        # Remove directory
        $SUDO rm -rf "$TARGET/$YOUR_MCP"

        # Remove symlink
        $SUDO rm -f "$TARGET/bin/$YOUR_MCP"

        # Remove mcp.json entry
        if command -v jq &> /dev/null && [ -f "$TARGET/etc/mcp.json" ]; then
            $SUDO jq "del(.mcpServers[\"$YOUR_MCP\"])" \
               "$TARGET/etc/mcp.json" > "/tmp/mcp.json.tmp"
            $SUDO mv "/tmp/mcp.json.tmp" "$TARGET/etc/mcp.json"
            info "Removed $YOUR_MCP from mcp.json"
        fi

        info "Uninstall complete"
    fi
    exit 0
fi

#
# INSTALL
#

# Skip if already up to date
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

# 1. Create structure if needed
if [ ! -d "$TARGET" ]; then
    info "Creating container-tools directory structure..."
    $SUDO mkdir -p "$TARGET/bin"
    $SUDO mkdir -p "$TARGET/etc"
fi
$SUDO mkdir -p "$TARGET/bin"
$SUDO mkdir -p "$TARGET/etc"

# 2. Backup existing installation
PRIOR_TEMP=""
if [ -d "$TARGET/$YOUR_MCP" ]; then
    info "Backing up previous installation..."

    $SUDO mkdir -p "$TARGET/$YOUR_MCP/backup"
    [ -f "$TARGET/etc/mcp.json" ] && $SUDO cp "$TARGET/etc/mcp.json" "$TARGET/$YOUR_MCP/backup/mcp.json-prior"
    [ -f "$SETTINGS" ] && cp "$SETTINGS" "$TARGET/$YOUR_MCP/backup/settings.json-prior"

    $SUDO rm -rf "/tmp/$YOUR_MCP-prior"
    $SUDO mv "$TARGET/$YOUR_MCP" "/tmp/$YOUR_MCP-prior"
    PRIOR_TEMP="/tmp/$YOUR_MCP-prior"
fi

# 3. Install MCP directory
info "Installing $YOUR_MCP..."
$SUDO cp -r "$SCRIPT_DIR" "$TARGET/$YOUR_MCP"

# 4. Move prior into backup/prior/
if [ -n "$PRIOR_TEMP" ] && [ -d "$PRIOR_TEMP" ]; then
    $SUDO mkdir -p "$TARGET/$YOUR_MCP/backup"
    $SUDO rm -rf "$TARGET/$YOUR_MCP/backup/prior"
    $SUDO mv "$PRIOR_TEMP" "$TARGET/$YOUR_MCP/backup/prior"
    info "Prior installation saved to $YOUR_MCP/backup/prior/"
fi

# 5. Ensure binaries are executable
$SUDO chmod +x "$TARGET/$YOUR_MCP/bin/$YOUR_MCP"
$SUDO chmod +x "$TARGET/$YOUR_MCP/bin/platforms"/* 2>/dev/null || true
$SUDO chmod +x "$TARGET/$YOUR_MCP/hooks"/*.sh 2>/dev/null || true

# 6. Create symlink (Linux/macOS only)
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

# 7. Update mcp.json
MCP_JSON="$TARGET/etc/mcp.json"
YOUR_COMMAND="$TARGET/$YOUR_MCP/bin/$YOUR_MCP"

if [ ! -f "$MCP_JSON" ]; then
    info "Creating mcp.json..."
    $SUDO tee "$MCP_JSON" > /dev/null << EOF
{
  "mcpServers": {
    "$YOUR_MCP": {
      "command": "$YOUR_COMMAND",
      "args": ["--mode", "stdio"]
    }
  }
}
EOF
elif command -v jq &> /dev/null; then
    info "Merging into mcp.json..."
    $SUDO jq --arg name "$YOUR_MCP" \
       --arg cmd "$YOUR_COMMAND" \
       '.mcpServers[$name] = {"command": $cmd, "args": ["--mode", "stdio"]}' \
       "$MCP_JSON" > "/tmp/mcp.json.tmp"
    $SUDO mv "/tmp/mcp.json.tmp" "$MCP_JSON"
else
    warn "jq not found - please manually configure mcp.json"
fi

# 8. Install hooks
info "Installing hooks..."
if command -v jq &> /dev/null; then
    # Backup settings.json
    if [ -f "$SETTINGS" ]; then
        $SUDO mkdir -p "$TARGET/$YOUR_MCP/backup"
        cp "$SETTINGS" "$TARGET/$YOUR_MCP/backup/settings.json-prior"
    fi

    # Install SessionStart hook
    if [ -f "$TARGET/$YOUR_MCP/hooks/session-start.sh" ]; then
        install_hook "SessionStart" "$TARGET/$YOUR_MCP/hooks/session-start.sh"
        info "Installed SessionStart hook"
    fi

    # Add other hooks here as needed
    # install_hook "PreCompact" "$TARGET/$YOUR_MCP/hooks/pre-compact.sh"
else
    warn "jq not found - please manually configure hooks in ~/.claude/settings.json"
fi

# 9. Verify installation
echo ""
info "Verifying installation..."
if [ -x "$TARGET/$YOUR_MCP/bin/$YOUR_MCP" ]; then
    VERSION_OUTPUT=$("$TARGET/$YOUR_MCP/bin/$YOUR_MCP" --version 2>&1 | head -1)
    info "Installed: $VERSION_OUTPUT"
else
    error "Installation verification failed"
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

## 11. Checklist for MCP Developers

Before releasing your container-tools package:

**Package Structure:**
- [ ] Package extracts with `{your-mcp}/` directory containing install.sh
- [ ] Contains `{your-mcp}/bin/{your-mcp}` universal launcher
- [ ] Contains `{your-mcp}/bin/platforms/` with versioned binaries
- [ ] Contains `{your-mcp}/hooks/` with hook scripts
- [ ] Contains `{your-mcp}/install.sh` following this guide
- [ ] Contains `{your-mcp}/LICENSE` file
- [ ] Contains `{your-mcp}/CHANGELOG.md` file
- [ ] Contains `{your-mcp}/README.md` file
- [ ] Does NOT contain obsolete hooks-dispatcher or hooks.d structure

**Installation Behavior:**
- [ ] Installer accepts `--target` parameter
- [ ] Installer accepts `--uninstall` flag
- [ ] Installer skips if platform binary MD5 matches
- [ ] Installer backs up mcp.json to `backup/mcp.json-prior`
- [ ] Installer backs up settings.json to `backup/settings.json-prior`
- [ ] Installer backs up previous installation to `backup/prior/`
- [ ] Installer merges (not replaces) mcp.json
- [ ] Installer merges (not replaces) settings.json hooks
- [ ] Installer creates symlink only on Linux/macOS

**Uninstall/Rollback Behavior:**
- [ ] Uninstaller removes hooks from settings.json
- [ ] Uninstaller restores prior installation if available
- [ ] Uninstaller merges prior entries (not full file replace)

**Testing:**
- [ ] Tested: fresh install
- [ ] Tested: update install (previous version exists)
- [ ] Tested: skip-if-identical
- [ ] Tested: rollback (uninstall with prior)
- [ ] Tested: full removal (uninstall without prior)
- [ ] Tested: coexistence with another MCP

---

## 12. Migration from hooks-dispatcher Pattern

If your MCP previously used the hooks-dispatcher pattern:

### What to Remove

1. Delete `etc/hooks-dispatcher.sh` from your package
2. Delete `etc/hooks.d/` directory structure
3. Remove hooks entries from `mcp.json` template

### What to Add

1. Create `{your-mcp}/hooks/` directory
2. Add hook scripts (e.g., `session-start.sh`)
3. Update installer to manage `settings.json`

### Cleanup for Existing Installations

Your installer should clean up old patterns:

```bash
# Remove obsolete dispatcher if we installed it
if [ -f "$TARGET/etc/hooks-dispatcher.sh" ]; then
    # Check if any other MCP still uses hooks.d
    if [ -z "$(ls -A "$TARGET/etc/hooks.d" 2>/dev/null)" ]; then
        $SUDO rm -f "$TARGET/etc/hooks-dispatcher.sh"
        $SUDO rm -rf "$TARGET/etc/hooks.d"
        info "Cleaned up obsolete hooks-dispatcher"
    fi
fi

# Remove old hook format from mcp.json
if [ -f "$TARGET/etc/mcp.json" ] && command -v jq &> /dev/null; then
    if jq -e '.hooks' "$TARGET/etc/mcp.json" > /dev/null 2>&1; then
        $SUDO jq 'del(.hooks)' "$TARGET/etc/mcp.json" > "/tmp/mcp.json.tmp"
        $SUDO mv "/tmp/mcp.json.tmp" "$TARGET/etc/mcp.json"
        info "Removed obsolete hooks from mcp.json"
    fi
fi
```

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 2.0 | 2025-01-XX | Complete rewrite for Claude Code native hooks (settings.json) |
| 1.0 | 2025-01-XX | Initial guide (hooks-dispatcher pattern - deprecated) |

---

## Support

- GitHub: https://github.com/ironsheep/todo-mcp
- Issues: https://github.com/ironsheep/todo-mcp/issues
