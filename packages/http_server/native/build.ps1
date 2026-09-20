# Builds the native library for this platform into this folder, named the way
# hana.pkg.json expects (http_server-windows-<arch>.dll). Needs Go and a C compiler (gcc).
Set-Location $PSScriptRoot
$env:CGO_ENABLED = "1"
$arch = go env GOARCH
# -s -w -trimpath drop debug info and build paths: about half the size (11MB -> 5.7MB)
go build -buildmode=c-shared -ldflags="-s -w" -trimpath -o "http_server-windows-$arch.dll" .
Remove-Item "http_server-windows-$arch.h" -ErrorAction SilentlyContinue
Write-Host "built http_server-windows-$arch.dll"
