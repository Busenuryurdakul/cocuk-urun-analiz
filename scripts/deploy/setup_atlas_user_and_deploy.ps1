# One-shot: create Atlas miyuna_app user (Cluster0) + push MONGODB_URI to Render + redeploy.
#
# Prerequisite (once per machine):
#   & "C:\Program Files (x86)\MongoDB Atlas CLI\atlas.exe" auth login
#
# Usage:
#   .\scripts\deploy\setup_atlas_user_and_deploy.ps1
#   .\scripts\deploy\setup_atlas_user_and_deploy.ps1 -SkipDbUser   # URI only, user already exists
#
# Optional env:
#   MONGODB_ATLAS_PROJECT_ID=6a9c906f2c0d1cfd602f8794
#   MIYUNA_ATLAS_DB_PASSWORD=...   # if user exists, skip password generation

param(
    [string]$ProjectId = '6a9c906f2c0d1cfd602f8794',
    [string]$ClusterName = 'Cluster0',
    [string]$DatabaseName = 'miyuna',
    [string]$DbUser = 'miyuna_app',
    [switch]$SkipDbUser,
    [ValidateSet('huggingface', 'render')]
    [string]$LlmBackend = 'huggingface'
)

$ErrorActionPreference = 'Stop'
$AtlasExe = 'C:\Program Files (x86)\MongoDB Atlas CLI\atlas.exe'
if (-not (Test-Path $AtlasExe)) {
    throw "Atlas CLI not found at $AtlasExe. Run: winget install -e --id MongoDB.MongoDBAtlasCLI"
}

function New-RandomPassword {
    param([int]$Length = 24)
    $bytes = New-Object byte[] $Length
    [System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
    [Convert]::ToBase64String($bytes).TrimEnd('=').Replace('+', 'x').Replace('/', 'y').Substring(0, $Length)
}

function Assert-AtlasLogin {
    $prev = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    & $AtlasExe auth whoami 2>&1 | Out-Null
    $ok = ($LASTEXITCODE -eq 0)
    $ErrorActionPreference = $prev
    if (-not $ok) {
        Write-Host 'Atlas CLI login required. Run in this terminal:' -ForegroundColor Yellow
        Write-Host "  & '$AtlasExe' auth login"
        Write-Host 'Then re-run this script.'
        exit 1
    }
}

function Ensure-DbUser {
    param([ref]$PasswordOut)
    $raw = & $AtlasExe dbusers list --projectId $ProjectId -o json
    if ($LASTEXITCODE -ne 0) { throw "dbusers list failed: $raw" }
    $users = ($raw | ConvertFrom-Json).results
    $existing = $users | Where-Object { $_.username -eq $DbUser } | Select-Object -First 1
    if ($existing) {
        Write-Host "DB user '$DbUser' already exists."
        if ($env:MIYUNA_ATLAS_DB_PASSWORD) {
            $PasswordOut.Value = $env:MIYUNA_ATLAS_DB_PASSWORD
            return
        }
        throw "User exists but MIYUNA_ATLAS_DB_PASSWORD is not set. Reset password in Atlas UI or export it."
    }

    $pwd = if ($env:MIYUNA_ATLAS_DB_PASSWORD) { $env:MIYUNA_ATLAS_DB_PASSWORD } else { New-RandomPassword }
    $PasswordOut.Value = $pwd
    Write-Host "Creating DB user '$DbUser' (readWrite@$DatabaseName)..."
    & $AtlasExe dbusers create `
        --projectId $ProjectId `
        --username $DbUser `
        --password $pwd `
        --role "readWrite@$DatabaseName" | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'dbusers create failed' }
}

function Build-MongoUri {
    param([string]$Password)
    $raw = & $AtlasExe clusters describe $ClusterName --projectId $ProjectId -o json
    if ($LASTEXITCODE -ne 0) { throw "clusters describe failed: $raw" }
    $cluster = $raw | ConvertFrom-Json
    if ($cluster.stateName -ne 'IDLE') {
        throw "Cluster '$ClusterName' state is $($cluster.stateName). Wait until IDLE."
    }
    $host = $cluster.connectionStrings.standardSrv
    if (-not $host) { throw 'Missing SRV connection string' }
    $base = $host -replace '^mongodb\+srv://', ''
    $encodedUser = [uri]::EscapeDataString($DbUser)
    $encodedPass = [uri]::EscapeDataString($Password)
    return "mongodb+srv://${encodedUser}:${encodedPass}@${base}/${DatabaseName}?retryWrites=true&w=majority"
}

Write-Host '=== Atlas user + Render deploy ===' -ForegroundColor Cyan
Assert-AtlasLogin

$dbPassword = $null
if (-not $SkipDbUser) {
    Ensure-DbUser -PasswordOut ([ref]$dbPassword)
} else {
    if (-not $env:MIYUNA_ATLAS_DB_PASSWORD) {
        throw '-SkipDbUser requires MIYUNA_ATLAS_DB_PASSWORD'
    }
    $dbPassword = $env:MIYUNA_ATLAS_DB_PASSWORD
}

$uri = Build-MongoUri -Password $dbPassword
Write-Host 'MONGODB_URI built (password not printed).'

$env:MONGODB_URI = $uri
$env:REDIS_URL = 'redis://red-da0tqr7lk1mc738palig:6379/1'
$env:RENDER_API_SERVICE_ID = 'srv-dae8m7dbedkc73ajnipg'
$env:RENDER_AGENT_SERVICE_ID = 'srv-dae8k10u01pc73dhddp0'

& "$PSScriptRoot\configure_render_production.ps1" -LlmBackend $LlmBackend

Write-Host ''
Write-Host 'Waiting 90s for Render cold start...'
Start-Sleep -Seconds 90
try {
    $health = Invoke-RestMethod -Uri 'https://miyuna-api.onrender.com/health' -TimeoutSec 90
    Write-Host "Health OK: $($health | ConvertTo-Json -Compress)" -ForegroundColor Green
} catch {
    Write-Host "Health pending: $($_.Exception.Message)" -ForegroundColor Yellow
    Write-Host 'Logs: render logs -r srv-dae8m7dbedkc73ajnipg --limit 30'
}

Write-Host ''
Write-Host 'Save this password in your password manager (shown once):' -ForegroundColor Yellow
Write-Host $dbPassword
