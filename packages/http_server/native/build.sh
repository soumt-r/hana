#!/bin/sh
# Builds the native library for this platform into this folder, named the way
# hana.pkg.json expects (http_server-<os>-<arch>.<so|dylib>). Needs Go and a C compiler.
set -e
cd "$(dirname "$0")"
os=$(go env GOOS); arch=$(go env GOARCH)
case "$os" in darwin) ext=dylib ;; *) ext=so ;; esac
# -s -w -trimpath drop debug info and build paths: about half the size (11MB -> 5.7MB)
CGO_ENABLED=1 go build -buildmode=c-shared -ldflags="-s -w" -trimpath -o "http_server-$os-$arch.$ext" .
rm -f "http_server-$os-$arch.h"
echo "built http_server-$os-$arch.$ext"
