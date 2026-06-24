#!/bin/sh
# Number Guess — a tiny original demo door game for AdmiralBBS.
# Reads the caller's handle from the door32.sys dropfile (line 7), then plays
# a guess-the-number round over stdin/stdout. No args; uses $DOORFILE if set.
DROP="${DOORFILE:-door32.sys}"
HANDLE="player"
[ -f "$DROP" ] && HANDLE=$(sed -n '7p' "$DROP" | tr -d '\r')

CYAN=$(printf '\033[36m'); GREEN=$(printf '\033[32m'); RESET=$(printf '\033[0m')

printf '%s== NUMBER GUESS ==%s\r\n' "$CYAN" "$RESET"
printf 'Welcome, %s! I am thinking of a number between 1 and 100.\r\n' "$HANDLE"

secret=$(( (${RANDOM:-$$} % 100) + 1 ))
tries=0
while [ "$tries" -lt 7 ]; do
  tries=$((tries + 1))
  printf '%sGuess #%d (or q to quit): %s' "$GREEN" "$tries" "$RESET"
  read -r guess || break
  case "$guess" in
    q|Q) printf 'Bye, %s!\r\n' "$HANDLE"; exit 0 ;;
  esac
  case "$guess" in
    ''|*[!0-9]*) printf 'Numbers only!\r\n'; tries=$((tries - 1)); continue ;;
  esac
  if [ "$guess" -lt "$secret" ]; then
    printf 'Too low.\r\n'
  elif [ "$guess" -gt "$secret" ]; then
    printf 'Too high.\r\n'
  else
    printf '%sGot it in %d tries, %s! The number was %d.%s\r\n' "$CYAN" "$tries" "$HANDLE" "$secret" "$RESET"
    exit 0
  fi
done
printf 'Out of guesses! The number was %d. Come back soon, %s.\r\n' "$secret" "$HANDLE"
exit 0
