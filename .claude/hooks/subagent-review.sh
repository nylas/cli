#!/bin/bash
# Validates subagent completion quality
# Blocks if subagent found critical issues that need addressing

set -euo pipefail

# Get the subagent output from environment
SUBAGENT_OUTPUT="${CLAUDE_TOOL_OUTPUT:-}"

# Check if subagent output contains critical issues
if [ -n "$SUBAGENT_OUTPUT" ]; then
    # Check for critical/error patterns in the output
    if echo "$SUBAGENT_OUTPUT" | grep -qiE "CRITICAL|FATAL|FAILED.*test|BUILD FAILED"; then
        echo '{"decision": "block", "reason": "Subagent found critical issues that need addressing"}' >&2
        exit 2
    fi

    # Check for test failures
    if echo "$SUBAGENT_OUTPUT" | grep -qiE "FAIL.*Test|Test.*FAILED|\bFAIL\b.*\.go"; then
        echo '{"decision": "block", "reason": "Subagent reported test failures - please fix before completing"}' >&2
        exit 2
    fi

    # Check for compilation errors
    if echo "$SUBAGENT_OUTPUT" | grep -qiE "cannot find package|undefined:|syntax error"; then
        echo '{"decision": "block", "reason": "Subagent found compilation errors - please fix before completing"}' >&2
        exit 2
    fi
fi

# All checks passed
exit 0
