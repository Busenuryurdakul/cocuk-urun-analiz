# Finish Miyuna production deploy: Atlas (optional) + Render env.
#
# If Atlas CLI login fails, use -FromDashboard and paste URI from cloud.mongodb.com
#
#   .\scripts\deploy\finish_atlas_and_render.ps1 -FromDashboard
#
# Or login first, then run without flags:
#   atlas auth login
#   .\scripts\deploy\finish_atlas_and_render.ps1

param(
    [switch]$FromDashboard
)

$ErrorActionPreference = 'Stop'
$env:Path += ';C:\Program Files (x86)\MongoDB Atlas CLI\'

$ClusterName = 'Cluster0'
$DatabaseName = 'miyuna'

function Test-AtlasLoggedIn {
    $prev = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $out = & atlas auth whoami 2>&1
    $code = $LASTEXITCODE
    $ErrorActionPreference = $prev
    return ($code -eq 0)
}

function Invoke-AtlasLoginHint {
    Write-Host ''
    Write-Host 'Atlas CLI is not logged in.' -ForegroundColor Yellow
    Write-Host ''
    Write-Host 'Option 1 — Login in this terminal, then re-run this script:'
    Write-Host '  atlas auth login'
    Write-Host '  (Select UserAccount, complete browser login)'
    Write-Host ''
    Write-Host 'Option 2 — Skip CLI, use Atlas Dashboard URI:'
    Write-Host '  .\scripts\deploy\finish_atlas_and_render.ps1 -FromDashboard'
    Write-Host ''
    Write-Host 'Dashboard: https://cloud.mongodb.com/ -> Database -> Connect -> Drivers'
    throw 'Atlas login required. Use one of the options above.'
}

function Get-ProjectId {
    $raw = & atlas projects list -o json 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw "Atlas projects list failed: $raw"
    }
    $projects = $raw | ConvertFrom-Json
    if (-not $projects.results -or $projects.results.Count -eq 0) {
        throw 'No Atlas projects found. Complete atlas setup or create a project in the dashboard.'
    }
    return $projects.results[0].id
}

function Ensure-RenderNetworkAccess {
    param([string]$ProjectId)
    $raw = & atlas accessLists list --projectId $ProjectId -o json 2>&1
    if ($LASTEXITCODE -ne 0) { return }
    $entries = $raw | ConvertFrom-Json
    $open = $entries.results | Where-Object { $_.cidrBlock -eq '0.0.0.0/0' }
    if (-not $open) {
        Write-Host 'Adding 0.0.0.0/0 for Render...'
        & atlas accessLists create --projectId $ProjectId --cidr 0.0.0.0/0 --comment 'Render' | Out-Null
    }
}

function Get-MongoUriFromUser {
    Write-Host ''
    Write-Host 'Paste MONGODB_URI from Atlas Dashboard:'
    Write-Host '  Database -> Connect -> Drivers'
    Write-Host '  Replace <password> with your DB user password'
    Write-Host "  Ensure path ends with /$DatabaseName"
    Write-Host ''
    $uri = Read-Host 'MONGODB_URI'
    if ([string]::IsNullOrWhiteSpace($uri)) { throw 'MONGODB_URI required.' }
    return $uri.Trim()
}

function Invoke-RenderConfigure {
    param([string]$MongoUri)
    $env:MONGODB_URI = $MongoUri
    $env:REDIS_URL = 'redis://red-da0tqr7lk1mc738palig:6379/1'
    $env:RENDER_API_SERVICE_ID = 'srv-dae8m7dbedkc73ajnipg'
    $env:RENDER_AGENT_SERVICE_ID = 'srv-dae8k10u01pc73dhddp0'
    & "$PSScriptRoot\configure_render_production.ps1"
}

function Test-ApiHealth {
    Write-Host ''
    Write-Host 'Waiting 90s for Render cold start...'
    Start-Sleep -Seconds 90
    try {
        $health = Invoke-RestMethod -Uri 'https://miyuna-api.onrender.com/health' -TimeoutSec 90
        Write-Host "Health OK: $($health | ConvertTo-Json -Compress)" -ForegroundColor Green
    } catch {
        Write-Host "Health check pending: $($_.Exception.Message)" -ForegroundColor Yellow
        Write-Host 'Try again in 1-2 min: https://miyuna-api.onrender.com/health'
    }
}

if ($FromDashboard) {
    Write-Host 'Dashboard mode — skipping Atlas CLI checks.'
    $uri = Get-MongoUriFromUser
    Invoke-RenderConfigure -MongoUri $uri
    Test-ApiHealth
    exit 0
}

Write-Host 'Checking Atlas login...'
if (-not (Test-AtlasLoggedIn)) {
    Invoke-AtlasLoginHint
}

$projectId = Get-ProjectId
Write-Host "Project ID: $projectId"
Ensure-RenderNetworkAccess -ProjectId $projectId

$raw = & atlas clusters describe $ClusterName --projectId $projectId -o json 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "Cluster '$ClusterName' not found via CLI. Use dashboard URI instead." -ForegroundColor Yellow
    Write-Host "  .\scripts\deploy\finish_atlas_and_render.ps1 -FromDashboard"
    throw "Cluster describe failed: $raw"
}

$cluster = $raw | ConvertFrom-Json
if ($cluster.stateName -ne 'IDLE') {
    throw "Cluster '$ClusterName' state is $($cluster.stateName). Wait until IDLE in Atlas dashboard."
}

Write-Host "Cluster '$ClusterName' is ready."
$uri = Get-MongoUriFromUser
Invoke-RenderConfigure -MongoUri $uri
Test-ApiHealth
