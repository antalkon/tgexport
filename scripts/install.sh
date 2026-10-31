#!/bin/sh
# tgexport installer for macOS and Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/antalkon/tgexport/main/scripts/install.sh | sh
#
# Running it again updates tgexport to the latest release (or does nothing if
# it is already up to date). Options can be passed after "sh -s --":
#
#   --version vX.Y.Z   install a specific release instead of the latest
#   --dir PATH         install directory (default: where tgexport already is,
#                      otherwise ~/.local/bin)
#   --force            reinstall even if the same version is installed
#   --no-modify-path   do not add the install directory to your shell profile
#
# Environment variables TGEXPORT_VERSION and TGEXPORT_INSTALL_DIR work too.
set -eu

REPO="antalkon/tgexport"
BIN="tgexport"
PATH_MARKER="# added by tgexport installer"

VERSION="${TGEXPORT_VERSION:-}"
INSTALL_DIR="${TGEXPORT_INSTALL_DIR:-}"
FORCE=0
MODIFY_PATH=1

say() { printf '%s\n' "$*"; }
err() { printf 'tgexport-install: %s\n' "$*" >&2; exit 1; }
has() { command -v "$1" >/dev/null 2>&1; }

while [ $# -gt 0 ]; do
	case "$1" in
	--version) [ $# -ge 2 ] || err "--version needs a value"; VERSION="$2"; shift 2 ;;
	--dir) [ $# -ge 2 ] || err "--dir needs a value"; INSTALL_DIR="$2"; shift 2 ;;
	--force) FORCE=1; shift ;;
	--no-modify-path) MODIFY_PATH=0; shift ;;
	-h | --help) sed -n '2,16p' "$0" 2>/dev/null | sed 's/^# \{0,1\}//'; exit 0 ;;
	*) err "unknown option: $1" ;;
	esac
done

# --- downloader ------------------------------------------------------------
if has curl; then
	fetch() { curl -fsSL "$1" -o "$2"; }
	latest_tag() { curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest" | sed 's|.*/tag/||'; }
elif has wget; then
	fetch() { wget -qO "$2" "$1"; }
	latest_tag() { wget -S --spider "https://github.com/$REPO/releases/latest" 2>&1 | sed -n 's|.*[Ll]ocation: .*/tag/\([^[:space:]]*\).*|\1|p' | tail -n1; }
else
	err "curl or wget is required"
fi

# --- platform --------------------------------------------------------------
case "$(uname -s)" in
Darwin) OS=darwin ;;
Linux) OS=linux ;;
*) err "unsupported OS $(uname -s); on Windows use scripts/install.ps1" ;;
esac
case "$(uname -m)" in
x86_64 | amd64) ARCH=amd64 ;;
arm64 | aarch64) ARCH=arm64 ;;
*) err "unsupported architecture $(uname -m)" ;;
esac

# --- version ---------------------------------------------------------------
if [ -z "$VERSION" ]; then
	VERSION="$(latest_tag)" || true
	case "$VERSION" in v*) ;; *) err "could not determine the latest release of $REPO" ;; esac
fi
case "$VERSION" in v*) ;; *) VERSION="v$VERSION" ;; esac

CURRENT=""
EXISTING="$(command -v "$BIN" 2>/dev/null || true)"
if [ -n "$EXISTING" ]; then
	# "tgexport 0.1.0 (commit ...)" → "0.1.0"
	CURRENT="$("$EXISTING" version 2>/dev/null | awk '{print $2}' | sed 's/^v//')" || CURRENT=""
	[ -n "$INSTALL_DIR" ] || INSTALL_DIR="$(dirname "$EXISTING")"
fi
[ -n "$INSTALL_DIR" ] || INSTALL_DIR="$HOME/.local/bin"

if [ "$CURRENT" = "${VERSION#v}" ] && [ "$FORCE" -eq 0 ]; then
	say "tgexport $VERSION is already installed at $EXISTING — nothing to do."
	say "Use --force to reinstall."
	exit 0
fi

# --- download & verify -----------------------------------------------------
ARCHIVE="${BIN}_${OS}_${ARCH}.tar.gz"
BASE="https://github.com/$REPO/releases/download/$VERSION"
TMP="$(mktemp -d 2>/dev/null || mktemp -d -t tgexport)"
trap 'rm -rf "$TMP"' EXIT INT TERM

if [ -n "$CURRENT" ]; then
	say "Updating tgexport $CURRENT → ${VERSION#v} ($OS/$ARCH)…"
else
	say "Installing tgexport ${VERSION#v} ($OS/$ARCH)…"
fi
fetch "$BASE/$ARCHIVE" "$TMP/$ARCHIVE" || err "download failed: $BASE/$ARCHIVE"
fetch "$BASE/checksums.txt" "$TMP/checksums.txt" || err "download failed: $BASE/checksums.txt"

EXPECTED="$(awk -v f="$ARCHIVE" '$2 == f {print $1}' "$TMP/checksums.txt")"
[ -n "$EXPECTED" ] || err "$ARCHIVE not listed in checksums.txt"
if has sha256sum; then
	ACTUAL="$(sha256sum "$TMP/$ARCHIVE" | awk '{print $1}')"
elif has shasum; then
	ACTUAL="$(shasum -a 256 "$TMP/$ARCHIVE" | awk '{print $1}')"
else
	err "sha256sum or shasum is required to verify the download"
fi
[ "$EXPECTED" = "$ACTUAL" ] || err "checksum mismatch for $ARCHIVE (expected $EXPECTED, got $ACTUAL)"

tar -xzf "$TMP/$ARCHIVE" -C "$TMP" "$BIN" || err "failed to extract $ARCHIVE"

# --- install ---------------------------------------------------------------
SUDO=""
mkdir -p "$INSTALL_DIR" 2>/dev/null || true
if [ ! -w "$INSTALL_DIR" ]; then
	has sudo || err "$INSTALL_DIR is not writable; rerun with --dir \$HOME/.local/bin"
	say "$INSTALL_DIR needs elevated permissions, using sudo."
	SUDO="sudo"
	$SUDO mkdir -p "$INSTALL_DIR"
fi
# Replace atomically so a running tgexport is never left half-written.
$SUDO cp "$TMP/$BIN" "$INSTALL_DIR/.$BIN.new"
$SUDO chmod 755 "$INSTALL_DIR/.$BIN.new"
$SUDO mv -f "$INSTALL_DIR/.$BIN.new" "$INSTALL_DIR/$BIN"

say "✓ $("$INSTALL_DIR/$BIN" version)"
say "  installed to $INSTALL_DIR/$BIN"

# --- PATH ------------------------------------------------------------------
case ":$PATH:" in
*":$INSTALL_DIR:"*) exit 0 ;;
esac

profile=""
case "$(basename "${SHELL:-sh}")" in
zsh) profile="$HOME/.zshrc" ;;
bash) if [ "$OS" = darwin ]; then profile="$HOME/.bash_profile"; else profile="$HOME/.bashrc"; fi ;;
fish) profile="$HOME/.config/fish/config.fish" ;;
*) profile="$HOME/.profile" ;;
esac

if [ "$MODIFY_PATH" -eq 1 ]; then
	if ! grep -qs "$PATH_MARKER" "$profile"; then
		mkdir -p "$(dirname "$profile")"
		case "$profile" in
		*.fish) line="fish_add_path \"$INSTALL_DIR\" $PATH_MARKER" ;;
		*) line="export PATH=\"$INSTALL_DIR:\$PATH\" $PATH_MARKER" ;;
		esac
		printf '\n%s\n' "$line" >>"$profile"
		say "  added $INSTALL_DIR to PATH in $profile"
	fi
	say "  restart your terminal (or run: export PATH=\"$INSTALL_DIR:\$PATH\") to use tgexport"
else
	say "  note: $INSTALL_DIR is not in your PATH"
fi
