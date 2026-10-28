#!/usr/bin/env bash
# Onboard a media-server user: Jellyfin account + 1Password item + welcome message to paste in Telegram.
# Usage: ./onboard.sh   (prompts: admin user, admin password, new username)
set -euo pipefail
export OP_ACCOUNT=my                                   # personal account; shell default is the work one

. ./.env                                  # gitignored; local-only
JELLYFIN=https://jellyfin.ms.${DOMAIN:?}
SEERR=https://seerr.ms.${DOMAIN:?}
TELEGRAM=${TELEGRAM_INVITE:?}

read -rp  'Jellyfin admin user: ' admin
read -rsp 'Jellyfin admin password: ' admin_pw; echo
read -rp  'New username: ' username
: "${admin:?}" "${admin_pw:?}" "${username:?}"          # empty prompt = stop here

jf() { curl -sS --fail-with-body -H 'Content-Type: application/json' "$@"; }

token=$(jq -cn --arg u "$admin" --arg p "$admin_pw" '{Username:$u,Pw:$p}' \
  | jf -H 'Authorization: MediaBrowser Client="onboard", Device="cli", DeviceId="onboard", Version="1"' \
       -X POST "$JELLYFIN/Users/AuthenticateByName" --data-binary @- | jq -r .AccessToken)
auth=(-H "Authorization: MediaBrowser Token=\"$token\"")
trap 'jf "${auth[@]}" -X POST "$JELLYFIN/Sessions/Logout" >/dev/null 2>&1 || true' EXIT

users=$(jf "${auth[@]}" "$JELLYFIN/Users")              # curl failure exits here, before any op call
if jq -e --arg n "$username" 'any(.[]; (.Name // "" | ascii_downcase) == ($n | ascii_downcase))' <<<"$users" >/dev/null; then
  echo "user '$username' already exists (Jellyfin names are case-insensitive)" >&2; exit 1
fi

item_id=$(op item create --vault Private --category login --title "Jellyfin $username" --url "$JELLYFIN" \
  "username=$username" --generate-password='letters,digits,20' --format json | jq -r .id)
password=$(op read "op://Private/$item_id/password")

if ! created=$(jq -cn --arg n "$username" --arg p "$password" '{Name:$n,Password:$p}' \
  | jf "${auth[@]}" -X POST "$JELLYFIN/Users/New" --data-binary @-); then
  printf 'Jellyfin rejected the user: %s\n' "$created" >&2     # show the 400 body
  op item delete "$item_id"; exit 1                             # no orphan item
fi

share=$(op item share "$item_id" --view-once)

cat <<EOF
Hi $username! Welcome to the movies server 🎬

Your login (opens once — save it to your password manager):
$share

Watch: $JELLYFIN
Works in the browser, phone apps, and TV apps.

Browse & request: $SEERR
Log in with the same account. Full catalog, request anything, and track your requests — it's all automated. Don't be shy with requests.

Tips:
- Log in on your phone or laptop first. On the TV, use Quick Connect instead of typing the password with the remote.
- iPhone: Swiftfin https://apps.apple.com/us/app/swiftfin/id1604098728 (smoother) or the official Jellyfin app https://apps.apple.com/us/app/jellyfin-mobile/id1480192618 (more stable).
- Android / Android TV: the official Jellyfin app from the Play Store, or VoidTV https://github.com/hritwikjohri/VoidTV-for-jellyfin (lets you request from the TV too).
- Glitches happen — just tell me, I keep track and fix them.

Updates on new movies and shows: $TELEGRAM
EOF
