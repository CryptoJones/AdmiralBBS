#!/usr/bin/env sh
# install-door.sh — fetch, build, and install a resident door game for AdmiralBBS.
#
# Defaults to Chrome Circuit Cowboys, but every source is OVERRIDABLE via env so
# nothing is tied to one forge (GitHub today, Codeberg or another forge tomorrow):
#
#   DOOR_REPO   git URL of the door repo   (default: the Chrome Circuit Cowboys GitHub)
#   DOOR_REF    branch/tag to build        (default: main)
#   DOOR_NAME   door name shown to callers (default: "Chrome Circuit Cowboys")
#   DOOR_ADDR   TCP listen address         (default: 127.0.0.1:4000)
#   DEST        install dir                (default: /opt/admiralbbs)
#   BIN         installed binary name      (default: cowboy)
#
# Example (Codeberg instead of GitHub):
#   DOOR_REPO=https://codeberg.org/CryptoJones/ChromeCircuitCowboys ./scripts/install-door.sh
set -eu

DOOR_REPO="${DOOR_REPO:-https://github.com/CryptoJones/ChromeCircuitCowboys}"
DOOR_REF="${DOOR_REF:-main}"
DOOR_NAME="${DOOR_NAME:-Chrome Circuit Cowboys}"
DOOR_ADDR="${DOOR_ADDR:-127.0.0.1:4000}"
DEST="${DEST:-/opt/admiralbbs}"
BIN="${BIN:-cowboy}"

command -v git >/dev/null 2>&1 || { echo "error: git is required" >&2; exit 1; }
command -v go  >/dev/null 2>&1 || { echo "error: go toolchain is required" >&2; exit 1; }

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

echo "==> Fetching door: $DOOR_REPO ($DOOR_REF)"
git clone --depth 1 --branch "$DOOR_REF" "$DOOR_REPO" "$WORK/door"

echo "==> Building (CGO disabled, pure-Go SQLite)"
( cd "$WORK/door" && CGO_ENABLED=0 go build -o "$WORK/$BIN" . )

echo "==> Installing -> $DEST/$BIN"
install -m 0755 "$WORK/$BIN" "$DEST/$BIN"

cat <<EOF

Done. The door server is at $DEST/$BIN.

  Run the door:    $DEST/$BIN -addr $DOOR_ADDR -db $DEST/data/cowboy.db -tick 2s
  Register it:     start AdmiralBBS with  -door "$DOOR_NAME|tcp|$DOOR_ADDR|0"
  Update checks:   add  -update-url <forge>/api/v1-or-v3/.../releases/latest  to either
                   binary (or set CCC_UPDATE_URL / ADMIRALBBS_UPDATE_URL). No forge is
                   baked in — point it wherever the releases live.
EOF
