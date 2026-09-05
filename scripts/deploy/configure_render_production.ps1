# Configure Miyuna Render production env vars via Render API.
# Requires: Render CLI logged in (~/.render/cli.yaml) OR RENDER_API_KEY env var.
# Usage:
#   $env:MONGODB_URI = 'mongodb+srv://...'
#   $env:RENDER_API_SERVICE_ID = 'srv-...'
#   $env:RENDER_AGENT_SERVICE_ID = 'srv-...'
#   $env:REDIS_URL = 'redis://...'
#   $env:HF_TOKEN = 'hf_...'   # optional, for Hugging Face LLM
#   .\scripts\deploy\configure_render_production.ps1

$ErrorActionPreference = 'Stop'

function Get-RenderApiKey {
    if ($env:RENDER_API_KEY) { return $env:RENDER_API_KEY }
    $cfgPath = Join-Path $env:USERPROFILE '.render\cli.yaml'
    if (-not (Test-Path $cfgPath)) {
        throw 'Render API key missing. Run: render login'
    }
    $lines = Get-Content $cfgPath
    foreach ($line in $lines) {
        if ($line -match '^\s*key:\s*(.+)$') {
            return $Matches[1].Trim()
        }
    }
    throw 'Could not read Render API key from cli.yaml'
}

function Set-RenderEnvVars {
    param(
        [string]$ServiceId,
        [hashtable]$Vars
    )
    $apiKey = Get-RenderApiKey
    $headers = @{
        Authorization = "Bearer $apiKey"
        'Content-Type' = 'application/json'
        Accept = 'application/json'
    }
    $body = @(
        foreach ($key in $Vars.Keys) {
            @{ key = $key; value = [string]$Vars[$key] }
        }
    )
    $json = $body | ConvertTo-Json -Depth 3 -Compress
    $uri = "https://api.render.com/v1/services/$ServiceId/env-vars"
    Invoke-RestMethod -Method Put -Uri $uri -Headers $headers -Body $json | Out-Null
}

function New-RandomSecret {
    param([int]$Length = 48)
    $chars = (48..57) + (65..90) + (97..122)
    -join ($chars | Get-Random -Count $Length | ForEach-Object { [char]$_ })
}

function Require-EnvVar {
    param([string]$Name)
    $value = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrWhiteSpace($value)) {
        Write-Host "$Name is required."
        exit 1
    }
    return $value
}

$apiServiceId = Require-EnvVar -Name 'RENDER_API_SERVICE_ID'
$agentServiceId = Require-EnvVar -Name 'RENDER_AGENT_SERVICE_ID'
$mongodbUri = Require-EnvVar -Name 'MONGODB_URI'
$redisUrl = Require-EnvVar -Name 'REDIS_URL'

$internalToken = if ($env:AGENT_INTERNAL_TOKEN) { $env:AGENT_INTERNAL_TOKEN } else { New-RandomSecret }
$jwtSecret = if ($env:JWT_SECRET) { $env:JWT_SECRET } else { New-RandomSecret 64 }
$credKey = if ($env:CREDENTIALS_ENCRYPTION_KEY) { $env:CREDENTIALS_ENCRYPTION_KEY } else { New-RandomSecret 32 }
$hfToken = $env:HF_TOKEN

$apiBaseUrl = if ($env:RENDER_API_URL) { $env:RENDER_API_URL } else { 'https://miyuna-api.onrender.com' }
$agentBaseUrl = if ($env:RENDER_AGENT_URL) { $env:RENDER_AGENT_URL } else { 'https://miyuna-agent.onrender.com' }
$webBaseUrl = if ($env:WEB_BASE_URL) { $env:WEB_BASE_URL } else { 'https://miyuna-web.vercel.app' }

$apiVars = @{
    API_HOST = '0.0.0.0'
    API_PORT = '8080'
    ALLOW_GRAPHQL_PLAYGROUND = 'false'
    COOKIE_SECURE = 'true'
    MAIL_QUEUE_ENABLED = 'true'
    LLM_USE_MOCK = 'false'
    TURNSTILE_ENABLED = 'false'
    WEB_BASE_URL = $webBaseUrl
    MONGODB_URI = $mongodbUri
    REDIS_URL = $redisUrl
    JWT_SECRET = $jwtSecret
    AGENT_INTERNAL_TOKEN = $internalToken
    AGENT_ORCHESTRATOR_URL = $agentBaseUrl
    GO_INTERNAL_API_URL = $apiBaseUrl
    LLM_PRIMARY_BASE_URL = 'https://router.huggingface.co/v1'
    LLM_SECONDARY_BASE_URL = 'https://router.huggingface.co/v1'
    LLM_PRIMARY_MODEL_NAME = 'Qwen/Qwen2.5-0.5B-Instruct'
    LLM_SECONDARY_MODEL_NAME = 'meta-llama/Llama-3.2-1B-Instruct'
    CREDENTIALS_ENCRYPTION_KEY = $credKey
    MAIL_SMTP_HOST = if ($env:MAIL_SMTP_HOST) { $env:MAIL_SMTP_HOST } else { 'localhost' }
    MAIL_SMTP_PORT = if ($env:MAIL_SMTP_PORT) { $env:MAIL_SMTP_PORT } else { '1025' }
    MAIL_SMTP_TLS = if ($env:MAIL_SMTP_TLS) { $env:MAIL_SMTP_TLS } else { 'false' }
    MAIL_FROM = if ($env:MAIL_FROM) { $env:MAIL_FROM } else { 'noreply@miyuna.local' }
}
if ($hfToken) {
    $apiVars['LLM_PRIMARY_API_KEY'] = $hfToken
    $apiVars['LLM_SECONDARY_API_KEY'] = $hfToken
}
if ($env:MAIL_SMTP_USER) { $apiVars['MAIL_SMTP_USER'] = $env:MAIL_SMTP_USER }
if ($env:MAIL_SMTP_PASS) { $apiVars['MAIL_SMTP_PASS'] = $env:MAIL_SMTP_PASS }

$agentVars = @{
    GO_API_URL = $apiBaseUrl
    INTERNAL_TOKEN = $internalToken
}

Write-Host 'Setting miyuna-api environment variables...'
Set-RenderEnvVars -ServiceId $apiServiceId -Vars $apiVars
Write-Host 'Setting miyuna-agent environment variables...'
Set-RenderEnvVars -ServiceId $agentServiceId -Vars $agentVars

Write-Host 'Triggering redeploy...'
render deploys create $apiServiceId --confirm -o json | Out-Null
render deploys create $agentServiceId --confirm -o json | Out-Null

Write-Host "Done. API: $apiBaseUrl  Agent: $agentBaseUrl"
Write-Host "Next: set Vercel NEXT_PUBLIC_API_URL=$apiBaseUrl/graphql and redeploy web."
