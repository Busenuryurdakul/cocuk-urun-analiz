# Configure agent env only (no MongoDB required).
$ErrorActionPreference = 'Stop'
function Get-RenderApiKey {
    if ($env:RENDER_API_KEY) { return $env:RENDER_API_KEY }
    $cfgPath = Join-Path $env:USERPROFILE '.render\cli.yaml'
    $lines = Get-Content $cfgPath
    foreach ($line in $lines) {
        if ($line -match '^\s*key:\s*(.+)$') { return $Matches[1].Trim() }
    }
    throw 'Render API key missing'
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
    $headers = @{ Authorization = "Bearer $apiKey"; 'Content-Type' = 'application/json'; Accept = 'application/json' }
    $body = @(
        foreach ($entry in ($Map.GetEnumerator() | Sort-Object Name)) {
            @{ key = $entry.Key; value = [string]$entry.Value }
        }
    ) | ConvertTo-Json -Depth 3 -Compress
    Invoke-RestMethod -Method Put -Uri "https://api.render.com/v1/services/$ServiceId/env-vars" -Headers $headers -Body $body | Out-Null
}

$token = if ($env:AGENT_INTERNAL_TOKEN) { $env:AGENT_INTERNAL_TOKEN } else {
    -join ((48..57 + 65..90 + 97..122) | Get-Random -Count 48 | ForEach-Object { [char]$_ })
}

$agentEnv = Get-RenderEnvMap -ServiceId 'srv-dae8k10u01pc73dhddp0'
if ($agentEnv.Count -eq 0) { throw 'Refusing to update: agent env map is empty.' }
$agentEnv['GO_API_URL'] = 'https://miyuna-api.onrender.com'
$agentEnv['INTERNAL_TOKEN'] = $token
Set-RenderEnvMap -ServiceId 'srv-dae8k10u01pc73dhddp0' -Map $agentEnv

Write-Host 'Agent env merged (existing vars preserved). Sync API token via scripts/deploy/sync_render_orchestrator.ps1'
Write-Host 'AGENT_INTERNAL_TOKEN set (not printed).'
