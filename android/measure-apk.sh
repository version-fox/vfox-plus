#!/bin/bash
# Measures the release APK on a booted emulator: cold-start through the
# launcher, settle, read dumpsys meminfo, write memory.txt, and screenshot the
# cold start for the documentation. Runs in CI
# (.github/workflows/android.yml); the emulator and the APK are already up.
set -euo pipefail

PKG="$(sed -n 's/.*applicationId = "\([^"]*\)".*/\1/p' android/app/build.gradle.kts | head -1)"
[ -n "$PKG" ] || { echo "no applicationId in android/app/build.gradle.kts" >&2; exit 1; }
adb install apk/*.apk
# Cold start through the launcher, like a user tapping the icon.
adb shell monkey -p "$PKG" -c android.intent.category.LAUNCHER 1
sleep 20
DUMP="$(adb shell dumpsys meminfo "$PKG")"
PSS_KB="$(echo "$DUMP" | awk '/^ *TOTAL +[0-9]/{print $2; exit}')"
RSS_KB="$(echo "$DUMP" | awk '{for (i = 1; i <= NF; i++) if ($i == "RSS:") {v = $(i+1); sub(/K$/, "", v); print v; exit}}')"
[ -n "$PSS_KB" ] && [ -n "$RSS_KB" ] || { echo "$DUMP"; exit 1; }
mkdir -p "$RUNNER_TEMP/memory-android"
printf 'memory (apk-cold): RSS %d MB, PSS %d MB in 1 processes\n' \
  "$(( (RSS_KB + 512) / 1024 ))" "$(( (PSS_KB + 512) / 1024 ))" \
  | tee "$RUNNER_TEMP/memory-android/memory.txt"
adb exec-out screencap -p > "$RUNNER_TEMP/screenshot-android.png"
