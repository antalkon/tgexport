#!/bin/sh
# tgexport uninstaller for macOS and Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/antalkon/tgexport/main/scripts/uninstall.sh | sh
#
# Removes every tgexport binary found in PATH and the default install
# locations, plus the PATH line the installer added to your shell profile.
set -eu

BIN="tgexport"
PATH_MARKER="# added by tgexport installer"

say() { printf '%s\n' "$*"; }

found=0
remove() {
	f="$1"
	[ -f "$f" ] || return 0
	found=1
	if [ -w "$(dirname "$f")" ]; then
		rm -f "$f"
	else
		say "$(dirname "$f") needs elevated permissions, using sudo."
		sudo rm -f "$f"
	fi
	say "✓ removed $f"
}

# Binaries reachable through PATH (there may be several).
old_ifs="$IFS"
IFS=:
for dir in $PATH; do
	[ -n "$dir" ] && remove "$dir/$BIN"
done
IFS="$old_ifs"

# Default locations, in case they are not in PATH in this shell.
for dir in "$HOME/.local/bin" /usr/local/bin; do
	remove "$dir/$BIN"
done

# PATH lines added by the installer.
for profile in "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.profile" "$HOME/.config/fish/config.fish"; do
	if [ -f "$profile" ] && grep -qs "$PATH_MARKER" "$profile"; then
		tmp="$(mktemp)"
		grep -v "$PATH_MARKER" "$profile" >"$tmp" || true
		cat "$tmp" >"$profile"
		rm -f "$tmp"
		say "✓ cleaned PATH entry in $profile"
	fi
done

if [ "$found" -eq 0 ]; then
	say "tgexport is not installed — nothing to remove."
else
	say "tgexport has been uninstalled."
fi
