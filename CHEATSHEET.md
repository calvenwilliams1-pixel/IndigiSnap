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

## Steam Deck Development Environment

### Shell setup (persists across terminals)
- Java 17 (JAVA_HOME=/usr/lib/jvm/java-17-openjdk)
- Android SDK (ANDROID_HOME=/home/deck/android-sdk)
- cb, killindigi functions defined in ~/.bashrc

### Local dev server (port 8090)
    cd ~/IndigiSnap/go-backend
    INDIGISNAP_PORT=8090 INDIGISNAP_BASE_DIR=./IndigiSnap go run ./cmd/server

Port 8080 is used by Steam webhelper, so we use 8090 for local dev.

### Kill server
    source ~/.bashrc
    killindigi

### Local APK build
    cd ~/IndigiSnap
    ./gradlew assembleDebug

### Install on phone
    adb install -r app/build/outputs/apk/debug/app-debug.apk

### Watch phone logs
    adb logcat -c
    adb logcat -v threadtime | grep IndigiSnapDebug

### Android SDK install (first time)
    sdkmanager "platform-tools" "platforms;android-34" "build-tools;34.0.0" "ndk;25.2.9519653"
    echo "sdk.dir=$HOME/android-sdk" > local.properties

