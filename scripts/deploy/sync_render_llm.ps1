# Sync Hugging Face LLM credentials on miyuna-api (safe merge — preserves existing env vars).
# Usage:
#   $env:HF_TOKEN = 'hf_...'
#   .\scripts\deploy\sync_render_llm.ps1
$ErrorActionPreference = 'Stop'

$ApiServiceId = if ($env:RENDER_API_SERVICE_ID) { $env:RENDER_API_SERVICE_ID } else { 'srv-dae8m7dbedkc73ajnipg' }
$HfRouter = 'https://router.huggingface.co/v1'

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

Write-Host 'Reading current miyuna-api env map...'
$apiEnv = Get-RenderEnvMap -ServiceId $ApiServiceId
if ($apiEnv.Count -eq 0) {
    throw 'Refusing to update: API env map is empty (would wipe service config).'
}

$token = $env:HF_TOKEN
if ([string]::IsNullOrWhiteSpace($token)) { $token = $env:LLM_PRIMARY_API_KEY }
if ([string]::IsNullOrWhiteSpace($token)) { $token = $apiEnv['LLM_PRIMARY_API_KEY'] }
if ([string]::IsNullOrWhiteSpace($token)) { $token = $apiEnv['HF_TOKEN'] }
if ([string]::IsNullOrWhiteSpace($token)) {
    throw 'HF_TOKEN (or LLM_PRIMARY_API_KEY) is required. Set $env:HF_TOKEN before running this script.'
}

$apiEnv['LLM_USE_MOCK'] = 'false'
$apiEnv['LLM_PRIMARY_BASE_URL'] = $HfRouter
$apiEnv['LLM_SECONDARY_BASE_URL'] = $HfRouter
if (-not $apiEnv.ContainsKey('LLM_PRIMARY_MODEL_NAME') -or [string]::IsNullOrWhiteSpace($apiEnv['LLM_PRIMARY_MODEL_NAME'])) {
    $apiEnv['LLM_PRIMARY_MODEL_NAME'] = 'Qwen/Qwen2.5-0.5B-Instruct'
}
if (-not $apiEnv.ContainsKey('LLM_SECONDARY_MODEL_NAME') -or [string]::IsNullOrWhiteSpace($apiEnv['LLM_SECONDARY_MODEL_NAME'])) {
    $apiEnv['LLM_SECONDARY_MODEL_NAME'] = 'meta-llama/Llama-3.2-1B-Instruct'
}
$apiEnv['LLM_PRIMARY_API_KEY'] = $token
$apiEnv['LLM_SECONDARY_API_KEY'] = $token

Write-Host "Updating miyuna-api LLM env ($($apiEnv.Count) vars)..."
Set-RenderEnvMap -ServiceId $ApiServiceId -Map $apiEnv

Write-Host 'Triggering API redeploy...'
render deploys create $ApiServiceId --confirm -o json | Out-Null

Write-Host 'LLM sync complete.'
Write-Host "  LLM_PRIMARY_BASE_URL   -> $HfRouter"
Write-Host "  LLM_SECONDARY_BASE_URL -> $HfRouter"
Write-Host "  LLM_PRIMARY_MODEL_NAME -> $($apiEnv['LLM_PRIMARY_MODEL_NAME'])"
Write-Host "  LLM_SECONDARY_MODEL_NAME -> $($apiEnv['LLM_SECONDARY_MODEL_NAME'])"
Write-Host '  API keys set from HF token (value not printed).'
