#!/bin/bash
set -e

# Usage: wait_for_paths.sh source destination wait_seconds

src=$1
dst=$2
seconds=$3

CMD="showpaths -srcIA $src -dstIA $dst"
count=1
while ! $CMD &>/dev/null; do
    ((count++))
    sleep 1
    if [ $count -gt $seconds ]; then
        >&2 echo "No paths after $count seconds"
        exit 1
    fi
done
