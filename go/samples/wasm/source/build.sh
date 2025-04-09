#!/bin/sh
WIT=$(git rev-parse --show-toplevel)/eng/wit
"$WIT/build.sh"

tinygo build \
    -target wasip2 \
    -wit-package "$WIT" \
    -wit-world source \
    -o source.wasm \
    .
