$ErrorActionPreference = 'Stop'

param(
  [string]$ProxyPort = '49953',
  [string]$GoExe = 'C:\Program Files\go\bin\go.exe',
  [string[]]$Packages = @('./internal/config', './internal/server', './internal/service', './cmd/server')
)

$repoRoot = Split-Path -Parent $PSScriptRoot
$workspaceRoot = [System.IO.Path]::GetFullPath((Join-Path $repoRoot '..'))

$env:GOMODCACHE = Join-Path $workspaceRoot '.gomodcache'
$env:GOPATH = Join-Path $workspaceRoot '.gopath'
$env:GOCACHE = Join-Path $workspaceRoot '.gocache'
$env:GOTOOLCHAIN = 'local'
$env:GOPROXY = 'https://goproxy.io,direct'
$env:GOSUMDB = 'off'
$env:HTTP_PROXY = "http://127.0.0.1:$ProxyPort"
$env:HTTPS_PROXY = "http://127.0.0.1:$ProxyPort"
$env:NO_PROXY = '127.0.0.1,localhost'

New-Item -ItemType Directory -Force -Path $env:GOMODCACHE, $env:GOPATH, $env:GOCACHE | Out-Null

Write-Host "Using Go executable: $GoExe"
Write-Host "Using proxy: $($env:HTTP_PROXY)"
Write-Host "Using GOMODCACHE: $($env:GOMODCACHE)"
Write-Host "Using GOPATH: $($env:GOPATH)"
Write-Host "Using GOCACHE: $($env:GOCACHE)"
Write-Host "Using GOTOOLCHAIN: $($env:GOTOOLCHAIN)"

Push-Location $repoRoot
try {
  & $GoExe test @Packages
} finally {
  Pop-Location
}
