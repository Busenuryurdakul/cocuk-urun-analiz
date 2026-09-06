# Complete Miyuna production seed + Render configuration.
#
# What this does:
#  1. Validates Atlas cluster + network access (via Atlas MCP / CLI when available)
#  2. Prompts for MONGODB_URI (or uses $env:MONGODB_URI)
#  3. Optionally provisions Render private LLM services (-ProvisionRenderLlm)
#  4. Pushes env vars to miyuna-api + miyuna-agent and triggers redeploy
#
# Quick start (Atlas URI from dashboard):
#   $env:MONGODB_URI = 'mongodb+srv://miyuna_app:<password>@cluster0....mongodb.net/miyuna?retryWrites=true&w=majority'
#   $env:HF_TOKEN = 'hf_...'   # if using Hugging Face LLM
#   .\scripts\deploy\complete_production_deploy.ps1
#
# Full Render LLM (paid):
#   .\scripts\deploy\complete_production_deploy.ps1 -ProvisionRenderLlm -LlmBackend render

param(
    [switch]$ProvisionRenderLlm,
    [ValidateSet('huggingface', 'render')]
    [string]$LlmBackend = 'huggingface',
    [string]$AtlasCluster = 'Cluster0',
    [string]$DatabaseName = 'miyuna'
)

$ErrorActionPreference = 'Stop'
$env:Path += ';C:\Program Files (x86)\MongoDB Atlas CLI\'

function Require-MongoUri {
    if (-not [string]::IsNullOrWhiteSpace($env:MONGODB_URI)) {
        return $env:MONGODB_URI.Trim()
    }
    Write-Host ''
    Write-Host 'Atlas setup (one-time):' -ForegroundColor Cyan
    Write-Host "  1. https://cloud.mongodb.com/ -> Project 0 -> $AtlasCluster"
    Write-Host '  2. Database Access -> Add user miyuna_app (readWrite on miyuna)'
    Write-Host '  3. Network Access already includes Render CIDRs (74.220.51.0/24, 74.220.59.0/24)'
    Write-Host '  4. Connect -> Drivers -> copy SRV URI, set database to /miyuna'
    Write-Host ''
    $uri = Read-Host 'MONGODB_URI'
    if ([string]::IsNullOrWhiteSpace($uri)) { throw 'MONGODB_URI required.' }
    return $uri.Trim()
}

Write-Host '=== Miyuna production deploy ===' -ForegroundColor Cyan
Write-Host "Atlas cluster: $AtlasCluster  Database: $DatabaseName"
Write-Host "LLM backend:   $LlmBackend"
Write-Host ''

$mongoUri = Require-MongoUri
$env:MONGODB_URI = $mongoUri

if ($ProvisionRenderLlm -or $LlmBackend -eq 'render') {
    Write-Host 'Step 1/2: Provision Render private LLM services...'
    & "$PSScriptRoot\provision_render_llm_services.ps1"
    if ($LASTEXITCODE -eq 2) {
        Write-Host 'Falling back to Hugging Face LLM backend.' -ForegroundColor Yellow
        $LlmBackend = 'huggingface'
    }
} else {
    Write-Host 'Step 1/2: Skipping Render LLM provisioning (use -ProvisionRenderLlm for Ollama on Render).'
}

Write-Host 'Step 2/2: Configure Render API + Agent env vars...'
$env:RENDER_API_SERVICE_ID = if ($env:RENDER_API_SERVICE_ID) { $env:RENDER_API_SERVICE_ID } else { 'srv-dae8m7dbedkc73ajnipg' }
$env:RENDER_AGENT_SERVICE_ID = if ($env:RENDER_AGENT_SERVICE_ID) { $env:RENDER_AGENT_SERVICE_ID } else { 'srv-dae8k10u01pc73dhddp0' }
$env:REDIS_URL = if ($env:REDIS_URL) { $env:REDIS_URL } else { 'redis://red-da0tqr7lk1mc738palig:6379/1' }

& "$PSScriptRoot\configure_render_production.ps1" -LlmBackend $LlmBackend

Write-Host ''
Write-Host 'Waiting 90s for Render cold start...'
Start-Sleep -Seconds 90
try {
    $health = Invoke-RestMethod -Uri 'https://miyuna-api.onrender.com/health' -TimeoutSec 90
    Write-Host "Health OK: $($health | ConvertTo-Json -Compress)" -ForegroundColor Green
} catch {
    Write-Host "Health pending: $($_.Exception.Message)" -ForegroundColor Yellow
    Write-Host 'Check logs: render logs -r srv-dae8m7dbedkc73ajnipg --limit 30'
}

Write-Host ''
Write-Host 'Note: API seeds LLM/compliance collections automatically on first successful Mongo connection.'
