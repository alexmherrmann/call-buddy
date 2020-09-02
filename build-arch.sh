#!/bin/sh
# Builds the given $1 target based on ${@:2} for architectures in arch.txt.
basedir=`dirname "$0"`
target="$1"
shift
cat "$basedir/arch.txt" \
    | grep -v '#' \
    | awk 'NF = 2 { printf "env GOOS=%s GOARCH=%s go build -o build/%s-%s/'"$target"' '"$@"'\n", $1, $2, $1, $2 }' \
    | sort | uniq \
    | sh

cat "$basedir/arch.txt" \
    | grep -v '^#' \
    | awk 'NF > 2 { printf "ln -sf build/%s-%s build/%s-%s\n", $1, $2, $1, $3 }' \
    | sh
