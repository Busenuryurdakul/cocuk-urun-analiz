# Sync orchestrator URL + shared internal token across miyuna-api and miyuna-agent.
# Preserves all existing Render env vars (safe merge, not replace-with-subset).
$ErrorActionPreference = 'Stop'

$ApiServiceId = if ($env:RENDER_API_SERVICE_ID) { $env:RENDER_API_SERVICE_ID } else { 'srv-dae8m7dbedkc73ajnipg' }
$AgentServiceId = if ($env:RENDER_AGENT_SERVICE_ID) { $env:RENDER_AGENT_SERVICE_ID } else { 'srv-dae8k10u01pc73dhddp0' }
$ApiBaseUrl = if ($env:RENDER_API_URL) { $env:RENDER_API_URL } else { 'https://miyuna-api.onrender.com' }
$AgentBaseUrl = if ($env:RENDER_AGENT_URL) { $env:RENDER_AGENT_URL } else { 'https://miyuna-agent.onrender.com' }

function Get-RenderApiKey {
    if ($env:RENDER_API_KEY) { return $env:RENDER_API_KEY }
    $cfgPath = Join-Path $env:USERPROFILE '.render\cli.yaml'
    if (-not (Test-Path $cfgPath)) { throw 'Render API key missing. Run: render login' }
    foreach ($line in Get-Content $cfgPath) {
        if ($line -match '^\s*key:\s*(.+)$') { return $Matches[1].Trim() }
    }
    throw 'Could not read Render API key from cli.yaml'
}

function Get-RenderEnvMap {
    param([string]$ServiceId)
    $apiKey = Get-RenderApiKey
    $headers = @{ Authorization = "Bearer $apiKey"; Accept = 'application/json' }
    $items = Invoke-RestMethod -Uri "https://api.render.com/v1/services/$ServiceId/env-vars" -Headers $headers
    $map = @{}
    foreach ($item in $items) {
        if ($null -eq $item.envVar) { continue }
        $map[$item.envVar.key] = [string]$item.envVar.value
    }
    return $map
}

function Set-RenderEnvMap {
    param([string]$ServiceId, [hashtable]$Map)
    $apiKey = Get-RenderApiKey
    $headers = @{
        Authorization = "Bearer $apiKey"
        'Content-Type' = 'application/json'
        Accept = 'application/json'
    }
    $body = @(
        foreach ($entry in ($Map.GetEnumerator() | Sort-Object Name)) {
            @{ key = $entry.Key; value = [string]$entry.Value }
        }
    ) | ConvertTo-Json -Depth 3 -Compress
    Invoke-RestMethod -Method Put -Uri "https://api.render.com/v1/services/$ServiceId/env-vars" -Headers $headers -Body $body | Out-Null
}

function New-RandomSecret {
    param([int]$Length = 48)
    $chars = (48..57) + (65..90) + (97..122)
    -join ($chars | Get-Random -Count $Length | ForEach-Object { [char]$_ })
}

Write-Host 'Reading current Render env maps...'
$apiEnv = Get-RenderEnvMap -ServiceId $ApiServiceId
$agentEnv = Get-RenderEnvMap -ServiceId $AgentServiceId

if ($apiEnv.Count -eq 0 -or $agentEnv.Count -eq 0) {
    throw 'Refusing to update: fetched env map is empty (would wipe service config).'
}

$token = $env:AGENT_INTERNAL_TOKEN
if ([string]::IsNullOrWhiteSpace($token)) {
    $token = $apiEnv['AGENT_INTERNAL_TOKEN']
}
if ([string]::IsNullOrWhiteSpace($token)) {
    $token = $agentEnv['INTERNAL_TOKEN']
}
if ([string]::IsNullOrWhiteSpace($token)) {
    $token = New-RandomSecret
    Write-Host 'Generated new shared internal token.'
}

$apiEnv['AGENT_ORCHESTRATOR_URL'] = $AgentBaseUrl
$apiEnv['AGENT_INTERNAL_TOKEN'] = $token
$apiEnv['GO_INTERNAL_API_URL'] = $ApiBaseUrl
if (-not $apiEnv.ContainsKey('AGENT_IPC_TIMEOUT') -or [string]::IsNullOrWhiteSpace($apiEnv['AGENT_IPC_TIMEOUT'])) {
    $apiEnv['AGENT_IPC_TIMEOUT'] = '90s'
}

$agentEnv['GO_API_URL'] = $ApiBaseUrl
$agentEnv['INTERNAL_TOKEN'] = $token
if (-not $agentEnv.ContainsKey('LLM_TIMEOUT_SECONDS') -or [string]::IsNullOrWhiteSpace($agentEnv['LLM_TIMEOUT_SECONDS'])) {
    $agentEnv['LLM_TIMEOUT_SECONDS'] = '120'
}

Write-Host "Updating miyuna-api ($($apiEnv.Count) vars)..."
Set-RenderEnvMap -ServiceId $ApiServiceId -Map $apiEnv
Write-Host "Updating miyuna-agent ($($agentEnv.Count) vars)..."
Set-RenderEnvMap -ServiceId $AgentServiceId -Map $agentEnv

Write-Host 'Triggering redeploys...'
render deploys create $ApiServiceId --confirm -o json | Out-Null
render deploys create $AgentServiceId --confirm -o json | Out-Null

Write-Host "Orchestrator sync complete."
Write-Host "  API  AGENT_ORCHESTRATOR_URL -> $AgentBaseUrl"
Write-Host "  Agent GO_API_URL           -> $ApiBaseUrl"
Write-Host '  Shared token synchronized (value not printed).'
