#!/bin/bash
# context.sh - dump all IndigiSnap docs
# Usage:
#   ./context.sh            auto-copy bundle to clipboard
#   ./context.sh --stdout   print bundle to stdout instead

set -euo pipefail
cd "$(dirname "$0")"

MODE="${1:-clipboard}"

generate() {
  echo "=== IndigiSnap Context Bundle ==="
  echo "Repo: github.com/calvenwilliams1-pixel/IndigiSnap"
  echo "Local: $(pwd)"
  echo "Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo ""
  echo "Paste everything below into a fresh AI chat to restore full context."
  echo ""

  git ls-files '*.md' | sort | while read -r f; do
    echo "==============================================================="
    echo "FILE: $f"
    echo "==============================================================="
    cat "$f"
    echo ""
  done

  echo "=== End of Context Bundle ==="
}

case "$MODE" in
  --stdout|--print|-p)
    generate
    ;;
  clipboard|*)
    generate | wl-copy
    echo "Context bundle copied to clipboard ($(git ls-files '*.md' | wc -l) files)."
    ;;
esac
