#!/bin/bash
# context.sh - dump all IndigiSnap docs to clipboard (silent)
# Usage: ./context.sh

set -e
cd "$(dirname "$0")"

FILES=(
  RESUME.md
  PROJECT.md
  TODO.md
  DECISIONS_OPEN.md
  ARCHITECTURE.md
  PRIORITIZATION_NOTES.md
  PRODUCTION_PLAN.md
  CAMERA_SPEC.md
)

{
  echo "=== IndigiSnap Context Bundle ==="
  echo "Repo: github.com/calvenwilliams1-pixel/IndigiSnap"
  echo "Local: $(pwd)"
  echo "Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo ""
  echo "Paste everything below into a fresh AI chat to restore full context."
  echo ""

  for f in "${FILES[@]}"; do
    if [ -f "$f" ]; then
      echo "==============================================================="
      echo "FILE: $f"
      echo "==============================================================="
      cat "$f"
      echo ""
    fi
  done

  echo "=== End of Context Bundle ==="
} | wl-copy
