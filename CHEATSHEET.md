# Context Handoff Cheatsheet

## Start A Fresh AI Chat

Primary (fast, needs AI URL fetch):
    cd /home/deck/IndigiSnap && ./urls.sh | wl-copy
    Paste into chat.

Fallback (works always, big paste):
    cd /home/deck/IndigiSnap && ./context.sh | wl-copy
    Paste into chat.

## Just Print To Terminal

    cd /home/deck/IndigiSnap && ./urls.sh
    cd /home/deck/IndigiSnap && ./context.sh

## When The AI Asks What To Do Next

Paste this after the context bundle:

    Workflow rules:
    - You generate commands. I run them on Steam Deck.
    - Terminal commands: python3 << 'PYEOF' ... PYEOF blocks.
    - Edits: File -> Find -> Replace format.
    - One change at a time, verify after each.
    - Do not resolve open decisions silently.
    - Cheap useful features stay in v1.

    Then tell me the next action from TODO.md.

## Script Locations

    /home/deck/IndigiSnap/urls.sh    -- print raw URLs
    /home/deck/IndigiSnap/context.sh -- print all docs

## If wl-copy Is Missing

Install:
    sudo steamos-readonly disable
    sudo pacman -S wl-clipboard
    sudo steamos-readonly enable

Alternative (xclip):
    sudo pacman -S xclip
    ./context.sh | xclip -selection clipboard
