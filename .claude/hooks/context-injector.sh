#!/bin/bash
# Inject relevant context based on user prompt
# This hook runs on UserPromptSubmit to provide contextual reminders

set -euo pipefail

PROMPT="${CLAUDE_USER_PROMPT:-}"

# If prompt mentions testing, remind about test patterns
if echo "$PROMPT" | grep -qi "test\|spec\|coverage"; then
    echo "Testing reminder: Use table-driven tests with t.Run(). See .claude/rules/testing.md"
fi

# If prompt mentions security, remind about security rules
if echo "$PROMPT" | grep -qi "security\|auth\|credential\|secret\|password"; then
    echo "Security check: Run 'make security' before committing. No hardcoded credentials."
fi

# If prompt mentions API or Nylas, remind about v3
if echo "$PROMPT" | grep -qi "api\|endpoint\|nylas"; then
    echo "API Note: Use Nylas v3 ONLY. Reference: https://developer.nylas.com/docs/api/v3/"
fi

# If prompt mentions commit or PR, remind about rules
if echo "$PROMPT" | grep -qi "commit\|push\|pr\|pull request"; then
    echo "Git reminder: Run 'make ci-full' before committing. Never commit secrets."
fi

# If prompt mentions file size or splitting, remind about limits
if echo "$PROMPT" | grep -qi "split\|large file\|too long\|refactor"; then
    echo "File size reminder: See .claude/rules/file-size-limits.md"
fi

exit 0
