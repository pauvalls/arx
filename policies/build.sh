#!/bin/sh
# Build all reference WASM policies.
# Requires: tinygo (https://tinygo.org/)
#
# Usage: ./build.sh
# Output: .wasm files in policies/

set -e

DIR="$(cd "$(dirname "$0")" && pwd)"

echo "Building layer-balance.wasm..."
tinygo build -o "$DIR/layer-balance.wasm" -target=wasi -no-debug "$DIR/layer-balance/"

echo "Building dependency-symmetry.wasm..."
tinygo build -o "$DIR/dependency-symmetry.wasm" -target=wasi -no-debug "$DIR/dependency-symmetry/"

echo "Building no-leak.wasm..."
tinygo build -o "$DIR/no-leak.wasm" -target=wasi -no-debug "$DIR/no-leak/"

echo "All policies built successfully."
ls -la "$DIR"/*.wasm
