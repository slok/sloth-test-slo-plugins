#!/usr/bin/env sh

set -o errexit
set -o nounset

for d in ./plugins/*/; do
    (cd $d && go test -count=1 -race -bench=. -benchmem ./... -v)
done
