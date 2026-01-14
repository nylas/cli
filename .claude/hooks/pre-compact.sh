#!/bin/bash
# Save session learnings before context compaction
# This hook runs when Claude's context window is about to be compacted

set -euo pipefail

DATE=$(date +%Y-%m-%d)
DIARY_DIR="$HOME/.claude/memory/diary"
mkdir -p "$DIARY_DIR"

# Count existing sessions for today
SESSION_NUM=$(ls "$DIARY_DIR"/$DATE-session-*.md 2>/dev/null | wc -l || echo "0")
SESSION_NUM=$((SESSION_NUM + 1))

DIARY_FILE="$DIARY_DIR/$DATE-session-$SESSION_NUM.md"

PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
PROGRESS_FILE="$PROJECT_ROOT/claude-progress.txt"

# Notify about compaction with actionable reminder
echo "=================================================="
echo "⚠️  CONTEXT COMPACTION IMMINENT"
echo "=================================================="
echo ""
echo "BEFORE COMPACTION - Save your work:"
echo ""
echo "1. Update claude-progress.txt with:"
echo "   - What was accomplished this session"
echo "   - Current task in progress"
echo "   - Next steps to continue"
echo "   Progress file: $PROGRESS_FILE"
echo ""
echo "2. Use /diary to capture learnings:"
echo "   Diary file ready: $DIARY_FILE"
echo ""
echo "This ensures the next session can continue seamlessly."
echo "=================================================="

exit 0
