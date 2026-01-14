#!/bin/bash
# file-size-check.sh - Enforce 500-line file size limit before Write operations
# Runs as PreToolUse hook for Write tool

set -e

# Get file path from TOOL_INPUT (JSON format)
FILE_PATH=$(echo "$TOOL_INPUT" | jq -r '.file_path // empty' 2>/dev/null)

# Skip if no file path or not a Go file
if [ -z "$FILE_PATH" ] || [[ ! "$FILE_PATH" =~ \.go$ ]]; then
    exit 0
fi

# Get content being written
CONTENT=$(echo "$TOOL_INPUT" | jq -r '.content // empty' 2>/dev/null)

# Count lines in new content
if [ -n "$CONTENT" ]; then
    LINE_COUNT=$(echo "$CONTENT" | wc -l)

    if [ "$LINE_COUNT" -gt 600 ]; then
        echo "⛔ BLOCKED: File would have $LINE_COUNT lines (max: 600)"
        echo "   File: $FILE_PATH"
        echo "   Action: Split the file by responsibility before writing"
        exit 2
    elif [ "$LINE_COUNT" -gt 500 ]; then
        echo "⚠️  WARNING: File has $LINE_COUNT lines (ideal: ≤500)"
        echo "   File: $FILE_PATH"
        echo "   Consider splitting this file soon"
        exit 0
    fi
fi

exit 0
