# Builds the native library for this platform into this folder, named the way
# hana.pkg.json expects (timezone-windows-<arch>.dll). Needs Go and a C compiler (gcc).
Set-Location $PSScriptRoot
$env:CGO_ENABLED = "1"
$arch = go env GOARCH
# -s -w -trimpath drop debug info and build paths: about half the size
go build -buildmode=c-shared -ldflags="-s -w" -trimpath -o "timezone-windows-$arch.dll" .
Remove-Item "timezone-windows-$arch.h" -ErrorAction SilentlyContinue
Write-Host "built timezone-windows-$arch.dll"
