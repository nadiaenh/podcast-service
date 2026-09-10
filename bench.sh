#!/usr/bin/env bash
# Times each pipeline stage on one article and reports output sizes. Uses your .env keys.
set -euo pipefail
cd "$(dirname "$0")"

url="${1:-https://go.dev/blog/error-handling-and-go}"
dir="bench-out"
mkdir -p "$dir"
go build -o "$dir/pod" .

for stage in "fetch $url $dir" "transcript $url $dir" "speak $dir"; do
  start=$(date +%s)
  # shellcheck disable=SC2086
  "$dir/pod" $stage >/dev/null
  printf '%-11s %3ds\n' "${stage%% *}" "$(( $(date +%s) - start ))"
done

printf 'source     %8d B\n' "$(wc -c < "$dir/source.md")"
printf 'transcript %8d B\n' "$(wc -c < "$dir/transcript.txt")"
printf 'audio      %8d B\n' "$(wc -c < "$dir/episode.mp3")"
