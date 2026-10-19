#!/usr/bin/env bash
# register-mcp.sh — Register wfl as an MCP server in the current project
# for Claude Code, Cursor, and VS Code (GitHub Copilot).
#
# Usage (run from the root of the project you want to register):
#   bash /path/to/wfl/scripts/register-mcp.sh
#
# The script:
#   1. Finds the .wfl file(s) in the current project
#   2. Registers wfl as an MCP server in .claude/settings.json,
#      .cursor/mcp.json, and .vscode/mcp.json
#   3. Prints what it did

set -euo pipefail

# ── Config ────────────────────────────────────────────────────────────────────
WFL_BINARY="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/wfl"
PROJECT_DIR="$(pwd)"

# ── Helpers ───────────────────────────────────────────────────────────────────
red()   { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
blue()  { printf '\033[34m%s\033[0m\n' "$*"; }
bold()  { printf '\033[1m%s\033[0m\n' "$*"; }

require_binary() {
  if ! command -v "$1" &>/dev/null; then
    red "ERROR: '$1' is required but not found in PATH."
    exit 1
  fi
}

# Merge a new mcpServers entry into an existing JSON config file.
# Creates the file if it does not exist.
# Args: <file> <server-name> <command> <args-json-array>
merge_mcp_entry() {
  local file="$1"
  local name="$2"
  local cmd="$3"
  local args_json="$4"

  local dir
  dir="$(dirname "$file")"
  mkdir -p "$dir"

  local new_entry
  new_entry=$(printf '{"command":"%s","args":%s}' "$cmd" "$args_json")

  if [ ! -f "$file" ]; then
    # Create fresh file with just this server.
    printf '{\n  "mcpServers": {\n    "%s": %s\n  }\n}\n' \
      "$name" "$new_entry" > "$file"
    return
  fi

  # File exists — use python3 (available on every mac) to merge safely.
  python3 - "$file" "$name" "$new_entry" <<'PYEOF'
import sys, json

path  = sys.argv[1]
name  = sys.argv[2]
entry = json.loads(sys.argv[3])

with open(path) as f:
    cfg = json.load(f)

cfg.setdefault("mcpServers", {})[name] = entry

with open(path, "w") as f:
    json.dump(cfg, f, indent=2)
    f.write("\n")
PYEOF
}

# ── Checks ────────────────────────────────────────────────────────────────────
require_binary python3

if [ ! -x "$WFL_BINARY" ]; then
  red "ERROR: wfl binary not found or not executable at: $WFL_BINARY"
  red "Build it first:  cd $(dirname "$WFL_BINARY") && go build -o wfl ./cmd/wfl"
  exit 1
fi

# ── Find the workflow file ────────────────────────────────────────────────────
blue "Scanning $PROJECT_DIR for .wfl files..."

IFS=$'\n' read -r -d '' -a WFL_FILES < <(find "$PROJECT_DIR" -name "*.wfl" -not -path "*/.git/*" 2>/dev/null | sort && printf '\0') || true

if [ "${#WFL_FILES[@]}" -eq 0 ]; then
  red "ERROR: No .wfl files found in $PROJECT_DIR"
  red "Create one first:  wfl init features/<name>.wfl --flow-name <FlowName>"
  exit 1
fi

if [ "${#WFL_FILES[@]}" -eq 1 ]; then
  WFL_FILE="${WFL_FILES[0]}"
  blue "Found: $WFL_FILE"
else
  bold "Multiple .wfl files found:"
  for i in "${!WFL_FILES[@]}"; do
    printf "  [%d] %s\n" "$((i+1))" "${WFL_FILES[$i]}"
  done
  printf "Select [1-%d]: " "${#WFL_FILES[@]}"
  read -r choice
  WFL_FILE="${WFL_FILES[$((choice-1))]}"
fi

# Use absolute path.
WFL_FILE="$(cd "$(dirname "$WFL_FILE")" && pwd)/$(basename "$WFL_FILE")"
bold "\nRegistering MCP server for: $WFL_FILE"

ARGS_JSON='["serve", "'"$WFL_FILE"'"]'
SERVER_NAME="wfl"

# ── Claude Code (.claude/settings.json) ──────────────────────────────────────
CLAUDE_CFG="$PROJECT_DIR/.claude/settings.json"
merge_mcp_entry "$CLAUDE_CFG" "$SERVER_NAME" "$WFL_BINARY" "$ARGS_JSON"
green "[Claude Code]  $CLAUDE_CFG"

# ── Cursor (.cursor/mcp.json) ─────────────────────────────────────────────────
CURSOR_CFG="$PROJECT_DIR/.cursor/mcp.json"
merge_mcp_entry "$CURSOR_CFG" "$SERVER_NAME" "$WFL_BINARY" "$ARGS_JSON"
green "[Cursor]       $CURSOR_CFG"

# ── VS Code / Copilot (.vscode/mcp.json) ──────────────────────────────────────
VSCODE_CFG="$PROJECT_DIR/.vscode/mcp.json"
merge_mcp_entry "$VSCODE_CFG" "$SERVER_NAME" "$WFL_BINARY" "$ARGS_JSON"
green "[VS Code]      $VSCODE_CFG"

# ── Summary ───────────────────────────────────────────────────────────────────
cat <<SUMMARY

$(bold "Done. wfl is now registered in this project for:")
  • Claude Code  — reload the project or restart Claude Code
  • Cursor       — reload the window (Cmd+Shift+P → "Reload Window")
  • VS Code      — reload the window (Cmd+Shift+P → "Reload Window")

$(bold "MCP server command:")
  $WFL_BINARY serve $WFL_FILE

$(bold "Test it (Claude Code):")
  Run: wfl validate $WFL_FILE
  Then ask Claude: "list the flows in the wfl workflow file"

SUMMARY
