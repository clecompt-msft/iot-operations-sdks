#!/bin/sh
WIT=$(git rev-parse --show-toplevel)/eng/wit
"$WIT/build.sh"

rm -rf internal
wit-bindgen-go generate -w source -o internal "$WIT"
go mod tidy
