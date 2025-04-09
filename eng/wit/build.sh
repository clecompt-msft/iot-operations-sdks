#!/bin/sh
cd $(dirname "$0")

if [ ! -d "deps" ]; then
    git clone --depth 1 --branch v0.2.5 https://github.com/WebAssembly/WASI

    mkdir deps
    for dir in WASI/wasip2/*/; do
        mv "$dir" "deps/wasi-$(basename "$dir")"
    done    

    rm -rf WASI
fi
