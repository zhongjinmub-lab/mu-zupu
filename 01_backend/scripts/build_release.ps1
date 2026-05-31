param(
    [string]$Version = "dev",
    [string]$OutDir = "dist"
)

$ErrorActionPreference = "Stop"

$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
$ProjectRoot = Resolve-Path (Join-Path $Root "..")
$Dist = Join-Path $Root $OutDir
$PackageName = "mu-agent-saas-$Version-linux-amd64"
$PackageDir = Join-Path $Dist $PackageName

if (Test-Path $PackageDir) {
    Remove-Item -LiteralPath $PackageDir -Recurse -Force
}
New-Item -ItemType Directory -Force -Path (Join-Path $PackageDir "bin") | Out-Null

Push-Location $Root
try {
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    go build -trimpath -ldflags "-s -w" -o (Join-Path $PackageDir "bin/mu-agent-server") ./cmd/server
    go build -trimpath -ldflags "-s -w" -o (Join-Path $PackageDir "bin/mu-agent-migrate") ./cmd/migrate
    go build -trimpath -ldflags "-s -w" -o (Join-Path $PackageDir "bin/mu-agent-document-worker") ./cmd/document-worker
    go build -trimpath -ldflags "-s -w" -o (Join-Path $PackageDir "bin/mu-agent-webhook-worker") ./cmd/webhook-worker

    Copy-Item -Recurse -Force .\migrations (Join-Path $PackageDir "migrations")
    Copy-Item -Force .\deploy\production\docker-compose.yml (Join-Path $PackageDir "docker-compose.yml")
    Copy-Item -Force .\deploy\production\compose.env.example (Join-Path $PackageDir "compose.env.example")
    Copy-Item -Force .\deploy\production\mu-agent-saas.env.example (Join-Path $PackageDir "mu-agent-saas.env.example")
    Copy-Item -Recurse -Force .\deploy\production\systemd (Join-Path $PackageDir "systemd")
    Copy-Item -Recurse -Force .\deploy\production\scripts (Join-Path $PackageDir "scripts")
    Copy-Item -Recurse -Force .\deploy\production\nginx (Join-Path $PackageDir "nginx")
    Copy-Item -Force .\deploy\production\README.md (Join-Path $PackageDir "README.md")
    Copy-Item -Recurse -Force (Join-Path $ProjectRoot "02_frontend") (Join-Path $PackageDir "frontend")

    foreach ($RequiredScript in @("backup.sh", "smoke.sh", "upgrade.sh", "rollback.sh", "restore.sh", "restore-drill.sh")) {
        $ScriptPath = Join-Path $PackageDir "scripts/$RequiredScript"
        if (-not (Test-Path $ScriptPath)) {
            throw "missing release script: $RequiredScript"
        }
    }
    foreach ($RequiredFrontendFile in @("index.html", "assets/app.js", "assets/app.css")) {
        $FrontendPath = Join-Path $PackageDir "frontend/$RequiredFrontendFile"
        if (-not (Test-Path $FrontendPath)) {
            throw "missing frontend file: $RequiredFrontendFile"
        }
    }

    $Archive = Join-Path $Dist "$PackageName.tar.gz"
    if (Test-Path $Archive) {
        Remove-Item -LiteralPath $Archive -Force
    }
    tar -C $Dist -czf $Archive $PackageName

    Write-Host "release package: $Archive"
}
finally {
    Pop-Location
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
    Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue
}
