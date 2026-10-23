#!/usr/bin/env bash
#
# php2shell-check.sh — report whether PHP is patched against PHP2Shell
# (CVE-2026-109618 / CVE-2026-109619), fixed on 2026-10-21.
#
# USE ONLY ON HOSTS YOU OWN OR ARE AUTHORIZED TO TEST.
# This script is read-only: it reads local PHP version info, or fetches HTTP
# response headers from a host. It never touches the vulnerable code paths.
#
# Usage:
#   ./php2shell-check.sh                 # check PHP on this machine (authoritative)
#   ./php2shell-check.sh host [host...]  # check your own remote hosts via headers
#   ./php2shell-check.sh -f hosts.txt    # hosts from a file, one per line
#
# Exit status: 0 all checked targets patched; 1 at least one vulnerable/EOL;
#              2 at least one undetermined (and none vulnerable).

set -uo pipefail

# --- Fixed (patched) versions per supported branch -------------------------
# A build is patched if it is >= the fix for its branch. Branches 8.1 and
# older are end-of-life: no upstream fix exists.
declare -A FIX=( [8.2]="8.2.35" [8.3]="8.3.36" [8.4]="8.4.27" [8.5]="8.5.12" )
EOL_MAX_MINOR="8.1"   # 8.1.x and anything older has no official patch

worst=0   # tracks overall exit status (0 ok, 2 unknown, 1 vulnerable)
note_status() { # raise `worst` to the more severe of itself and $1
  local s=$1
  if [ "$s" = 1 ]; then worst=1
  elif [ "$s" = 2 ] && [ "$worst" != 1 ]; then worst=2
  fi
}

# Compare dotted versions: echo 0 if $1<$2, 1 if equal, 2 if $1>$2.
vercmp() {
  if [ "$1" = "$2" ]; then echo 1; return; fi
  local lower
  lower=$(printf '%s\n%s\n' "$1" "$2" | sort -V | head -n1)
  [ "$lower" = "$1" ] && echo 0 || echo 2
}

# Given an x.y.z version, print verdict + reason and return status code.
verdict() {
  local v=$1
  if ! [[ $v =~ ^([0-9]+)\.([0-9]+)\.([0-9]+) ]]; then
    echo "UNKNOWN   could not parse version '$v'"; return 2
  fi
  local minor="${BASH_REMATCH[1]}.${BASH_REMATCH[2]}"
  local fix="${FIX[$minor]:-}"

  if [ -z "$fix" ]; then
    # Not a branch we have a fix for: either newer-than-known or EOL/older.
    if [ "$(vercmp "$minor" "8.5")" = 2 ]; then
      echo "UNKNOWN   branch $minor is newer than this script knows; confirm >= its fix release"
      return 2
    fi
    echo "VULNERABLE branch $minor is end-of-life (<= $EOL_MAX_MINOR); no upstream patch — upgrade to 8.2.35+"
    return 1
  fi

  if [ "$(vercmp "$v" "$fix")" = 0 ]; then
    echo "VULNERABLE $v is below the $minor fix ($fix) — upgrade and restart workers"
    return 1
  fi
  echo "PATCHED   $v >= $fix (remember: workers must have been restarted after upgrade)"
  return 0
}

# --- Local check -----------------------------------------------------------
check_local() {
  echo "== Local machine =="
  local found=0

  # CLI binary
  if command -v php >/dev/null 2>&1; then
    local v; v=$(php -r 'echo PHP_VERSION;' 2>/dev/null)
    [ -z "$v" ] && v=$(php -v 2>/dev/null | head -n1 | awk '{print $2}')
    printf '  php (CLI)      %-12s ' "$v"; verdict "$v"; note_status $?
    found=1
  fi

  # FPM binaries (what the web server usually serves — can differ from CLI)
  local fpm
  for fpm in php-fpm php-fpm8.5 php-fpm8.4 php-fpm8.3 php-fpm8.2 \
             php-fpm7.4 php-fpm8.1 php-fpm8.0; do
    if command -v "$fpm" >/dev/null 2>&1; then
      local v; v=$("$fpm" -v 2>/dev/null | head -n1 | awk '{print $2}')
      printf '  %-14s %-12s ' "$fpm" "$v"; verdict "$v"; note_status $?
      found=1
    fi
  done

  if [ "$found" = 0 ]; then
    echo "  No php or php-fpm binary found in PATH."
    echo "  Check your package manager instead, e.g.:"
    echo "    dpkg -l | grep -i php      # Debian/Ubuntu"
    echo "    rpm -qa | grep -i php      # RHEL/Alma/Rocky"
    note_status 2
  fi
  echo
  echo "  NOTE: The CLI version can differ from what the web server runs, and a"
  echo "  patched binary is still exploitable until Apache/PHP-FPM is RESTARTED."
}

# --- Remote header check ---------------------------------------------------
check_remote() {
  local host=$1
  # Strip any scheme the user may have included.
  host=${host#http://}; host=${host#https://}; host=${host%%/*}
  printf '  %-30s ' "$host"

  local hdrs
  hdrs=$(curl -sS -D - -o /dev/null --max-time 20 \
              -A "php2shell-patch-check/1.0" "https://$host/" 2>/dev/null)
  if [ -z "$hdrs" ]; then
    echo "UNKNOWN   no response (unreachable, blocked, or TLS error)"
    note_status 2; return
  fi

  # Pull a PHP version out of Server: / X-Powered-By: banners.
  local banner v
  banner=$(printf '%s\n' "$hdrs" | grep -iE 'x-powered-by:|server:' \
           | tr -d '\r' | paste -sd' ' -)
  v=$(printf '%s\n' "$banner" | grep -oiE 'php/[0-9]+\.[0-9]+\.[0-9]+' \
      | head -n1 | cut -d/ -f2)

  if [ -z "$v" ]; then
    if printf '%s' "$banner" | grep -qi php; then
      echo "UNKNOWN   PHP present but no version in banner (version suppressed)"
    else
      echo "UNKNOWN   no PHP version banner (suppressed, proxied, or not PHP)"
    fi
    note_status 2; return
  fi
  verdict "$v"; note_status $?
}

# --- Argument handling -----------------------------------------------------
hosts=()
if [ $# -eq 0 ]; then
  check_local
else
  if [ "${1:-}" = "-f" ]; then
    [ -z "${2:-}" ] && { echo "usage: $0 -f hosts.txt" >&2; exit 64; }
    while IFS= read -r line; do
      line=${line%%#*}; line=$(echo "$line" | xargs)   # strip comments/space
      [ -n "$line" ] && hosts+=("$line")
    done < "$2"
  else
    hosts=("$@")
  fi

  echo "== Remote hosts (header banners only; authorize before scanning) =="
  echo "   Banners are often hidden or inaccurate — treat UNKNOWN as 'verify on the box'."
  echo
  for h in "${hosts[@]}"; do check_remote "$h"; done
fi

echo
case $worst in
  0) echo "RESULT: all checked targets look patched." ;;
  2) echo "RESULT: could not determine status for some targets — verify directly on the host." ;;
  1) echo "RESULT: at least one target is VULNERABLE or end-of-life. Patch and restart workers." ;;
esac
exit $worst
