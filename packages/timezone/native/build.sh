#!/bin/sh
# Builds the native library for this platform into this folder, named the way
# hana.pkg.json expects (timezone-<os>-<arch>.<so|dylib>). Needs Go and a C compiler.
set -e
cd "$(dirname "$0")"
os=$(go env GOOS); arch=$(go env GOARCH)
case "$os" in darwin) ext=dylib ;; *) ext=so ;; esac
# -s -w -trimpath drop debug info and build paths: a smaller library
CGO_ENABLED=1 go build -buildmode=c-shared -ldflags="-s -w" -trimpath -o "timezone-$os-$arch.$ext" .
rm -f "timezone-$os-$arch.h"
echo "built timezone-$os-$arch.$ext"
