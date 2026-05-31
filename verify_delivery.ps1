param(
    [string]$NodePath = "D:\Node.js\node.exe",
    [string]$GitPath = ""
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path

function Invoke-Check {
    param(
        [string]$Name,
        [scriptblock]$Block
    )
    Write-Host "==> $Name"
    & $Block
    Write-Host "OK  $Name"
}

function Resolve-Git {
    if ($GitPath -and (Test-Path -LiteralPath $GitPath)) {
        return $GitPath
    }

    $githubDesktopGit = Join-Path $env:LOCALAPPDATA "GitHubDesktop\app-3.5.11\resources\app\git\cmd\git.exe"
    if (Test-Path -LiteralPath $githubDesktopGit) {
        return $githubDesktopGit
    }

    return "git"
}

function Resolve-Node {
    if ($NodePath -and (Test-Path -LiteralPath $NodePath)) {
        return $NodePath
    }

    return "node"
}

function Get-OneFile {
    param(
        [string]$Directory,
        [string]$Filter
    )

    $files = @(Get-ChildItem -LiteralPath $Directory -Filter $Filter -File)
    if ($files.Count -ne 1) {
        throw "Expected exactly one file for filter '$Filter', found $($files.Count)."
    }

    return $files[0].FullName
}

$Git = Resolve-Git
$Node = Resolve-Node

Invoke-Check "frontend_js_syntax" {
    & $Node --check (Join-Path $Root "02_frontend\assets\app.js")
}

Invoke-Check "backend_go_tests" {
    Push-Location (Join-Path $Root "01_backend")
    try {
        go test ./...
    }
    finally {
        Pop-Location
    }
}

Invoke-Check "git_diff_whitespace" {
    & $Git -C $Root diff --check
}

Invoke-Check "delivery_checklists_done" {
    $checkFiles = @(
        (Get-OneFile -Directory $Root -Filter "04_*.md"),
        (Get-OneFile -Directory (Join-Path $Root "02_docs_v1_delivery") -Filter "06_*.md")
    )

    foreach ($path in $checkFiles) {
        $content = Get-Content -Encoding UTF8 -LiteralPath $path -Raw
        if ($content.Contains("[ ]")) {
            throw "Checklist still has unchecked items: $path"
        }
    }
}

Invoke-Check "manifest_status" {
    $manifestPath = Join-Path $Root "manifest.json"
    if (-not (Test-Path -LiteralPath $manifestPath)) {
        throw "Missing manifest.json"
    }

    $manifest = Get-Content -Encoding UTF8 -LiteralPath $manifestPath | ConvertFrom-Json
    if ($manifest.status -ne "final_delivery_verified") {
        throw "Unexpected manifest status: $($manifest.status)"
    }
    if (-not $manifest.file_count -or $manifest.file_count -lt 1) {
        throw "Invalid manifest file_count"
    }
    if (-not $manifest.files -or $manifest.files.Count -lt 1) {
        throw "Manifest files list is empty"
    }
}

Invoke-Check "delivery_entry_documents" {
    $readme = Get-Content -Encoding UTF8 -LiteralPath (Join-Path $Root "README.md") -Raw
    if (($readme -notlike "*final_delivery_verified*") -and ($readme -notlike "*最终交付包*")) {
        throw "README does not show final delivery status"
    }

    $planPath = Get-OneFile -Directory $Root -Filter "03_*SAAS_*.md"
    $plan = Get-Content -Encoding UTF8 -LiteralPath $planPath -Raw
    if ($plan.Contains("[ ]")) {
        throw "Development plan still has unchecked checklist items"
    }
}

Write-Host "delivery verification ok"
