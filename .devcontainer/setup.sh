#!/bin/bash
set -e

echo "🔧 Setting up Android SDK..."

export ANDROID_HOME=$HOME/android-sdk
export ANDROID_SDK_ROOT=$ANDROID_HOME
mkdir -p $ANDROID_HOME/cmdline-tools

cd /tmp
echo "⬇️  Downloading Android command-line tools..."
wget -q https://dl.google.com/android/repository/commandlinetools-linux-11076708_latest.zip
unzip -q commandlinetools-linux-*.zip
mv cmdline-tools $ANDROID_HOME/cmdline-tools/latest
rm commandlinetools-linux-*.zip

# Add to bashrc so it persists across terminal sessions
cat >> ~/.bashrc << 'BASHRC'

# Android SDK
export ANDROID_HOME=$HOME/android-sdk
export ANDROID_SDK_ROOT=$ANDROID_HOME
export PATH=$PATH:$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools
BASHRC

export PATH=$PATH:$ANDROID_HOME/cmdline-tools/latest/bin

echo "📜 Accepting licenses..."
yes | $ANDROID_HOME/cmdline-tools/latest/bin/sdkmanager --licenses > /dev/null 2>&1 || true

echo "📦 Installing SDK components..."
$ANDROID_HOME/cmdline-tools/latest/bin/sdkmanager \
  "platform-tools" \
  "platforms;android-34" \
  "build-tools;34.0.0" \
  "ndk;25.2.9519653" > /dev/null

echo "✅ Android SDK setup complete"
echo "ANDROID_HOME=$ANDROID_HOME"
echo "NDK version: 25.2.9519653"
