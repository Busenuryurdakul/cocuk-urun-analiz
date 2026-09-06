# Update only MONGODB_URI on Render miyuna-api (preserves other env vars).
# Usage:
#   $env:MONGODB_URI = 'mongodb+srv://...'
#   $env:RENDER_API_SERVICE_ID = 'srv-dae8m7dbedkc73ajnipg'
#   .\scripts\deploy\update_render_mongodb_uri.ps1

$ErrorActionPreference = 'Stop'

function Get-RenderApiKey {
    if ($env:RENDER_API_KEY) { return $env:RENDER_API_KEY }
    $cfgPath = Join-Path $env:USERPROFILE '.render\cli.yaml'
    if (-not (Test-Path $cfgPath)) { throw 'Render API key missing. Run: render login' }
    foreach ($line in Get-Content $cfgPath) {
        if ($line -match '^\s*key:\s*(.+)$') { return $Matches[1].Trim() }
    }
    throw 'Could not read Render API key from cli.yaml'
}

$serviceId = if ($env:RENDER_API_SERVICE_ID) { $env:RENDER_API_SERVICE_ID } else { 'srv-dae8m7dbedkc73ajnipg' }
$mongodbUri = $env:MONGODB_URI
if ([string]::IsNullOrWhiteSpace($mongodbUri)) {
    Write-Host 'MONGODB_URI is required.'
    exit 1
}

$apiKey = Get-RenderApiKey
$headers = @{
    Authorization = "Bearer $apiKey"
    Accept        = 'application/json'
}

$getUri = "https://api.render.com/v1/services/$serviceId/env-vars"
$existing = Invoke-RestMethod -Method Get -Uri $getUri -Headers $headers

$vars = @{}
foreach ($item in $existing) {
    $ev = $item.envVar
    if ($ev.key) { $vars[$ev.key] = [string]$ev.value }
}
$vars['MONGODB_URI'] = $mongodbUri

$body = @(
    foreach ($key in ($vars.Keys | Sort-Object)) {
        @{ key = $key; value = $vars[$key] }
    }
) | ConvertTo-Json -Depth 3 -Compress

Invoke-RestMethod -Method Put -Uri $getUri -Headers ($headers + @{ 'Content-Type' = 'application/json' }) -Body $body | Out-Null

Write-Host "Updated MONGODB_URI on $serviceId (total vars: $($vars.Count))"
Write-Host 'Triggering redeploy...'
render deploys create $serviceId --confirm -o json | Out-Null
Write-Host 'Done.'
