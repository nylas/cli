#!/bin/bash
# auto-format.sh - Auto-format Go files after Edit operations
# Runs as PostToolUse hook for Edit tool

set -e

# Get file path from TOOL_INPUT (JSON format)
FILE_PATH=$(echo "$TOOL_INPUT" | jq -r '.file_path // empty' 2>/dev/null)

# Skip if no file path or not a Go file
if [ -z "$FILE_PATH" ] || [[ ! "$FILE_PATH" =~ \.go$ ]]; then
    exit 0
fi

# Skip if file doesn't exist
if [ ! -f "$FILE_PATH" ]; then
    exit 0
fi

# Run gofmt on the file
if command -v gofmt &> /dev/null; then
    gofmt -w "$FILE_PATH" 2>/dev/null && echo "✓ Auto-formatted: $(basename "$FILE_PATH")"
fi

# Note: File size checks handled by file-size-check.sh (PreToolUse hook)

exit 0
