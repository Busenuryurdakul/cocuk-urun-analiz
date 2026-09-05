# Applies detected Ollama models to Miyuna backend runtime env and verifies gateway path.

[CmdletBinding()]
param(
    [string]$ModelName = 'qwen2.5:0.5b',
    [string]$HeavyModelName = 'llama3.2:latest',
    [string]$BaseUrl = 'http://127.0.0.1:11434/v1',
    [switch]$SkipGatewayTest
)

$ErrorActionPreference = 'Stop'

function Get-OllamaModelNames {
    try {
        $tags = Invoke-RestMethod -Uri 'http://127.0.0.1:11434/api/tags' -Method GET -TimeoutSec 15
        return @($tags.models | ForEach-Object { [string]$_.name } | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    }
    catch {
        return @()
    }
}

$models = @(Get-OllamaModelNames)
if ($models.Count -eq 0) {
    throw 'OLLAMA_UNAVAILABLE: start Ollama and ensure at least one model is installed.'
}

if ($models -notcontains $ModelName) {
    $ModelName = $models[0]
    Write-Host ("OLLAMA_MODEL_AUTO_SELECTED={0}" -f $ModelName)
}

$env:LLM_USE_MOCK = 'false'
$env:LLM_PRIMARY_BASE_URL = $BaseUrl
$env:LLM_PRIMARY_API_KEY = ''
$env:LLM_PRIMARY_MODEL_NAME = $ModelName
$env:LLM_SECONDARY_BASE_URL = $BaseUrl
$env:LLM_SECONDARY_API_KEY = ''
$env:LLM_SECONDARY_MODEL_NAME = $HeavyModelName

Write-Host ''
Write-Host '=== Miyuna Ollama Backend Configuration ==='
Write-Host ("FAST_MODEL={0}" -f $ModelName)
Write-Host ("HEAVY_MODEL={0}" -f $HeavyModelName)
Write-Host ("LLM_PRIMARY_BASE_URL={0}" -f $env:LLM_PRIMARY_BASE_URL)
Write-Host ("LLM_PRIMARY_MODEL_NAME={0}" -f $env:LLM_PRIMARY_MODEL_NAME)
Write-Host ("LLM_SECONDARY_MODEL_NAME={0}" -f $env:LLM_SECONDARY_MODEL_NAME)
Write-Host 'LLM_USE_MOCK=false'
Write-Host 'ESCALATION=fast_to_heavy_on_long_tasks'
Write-Host ''

$localScript = Join-Path $PSScriptRoot 'verify_local_runtime.ps1'
. $localScript -Mode Auto
Invoke-Phase6LocalRuntime -Mode Auto -PrimaryBaseUrl $BaseUrl -SecondaryBaseUrl $BaseUrl

if (-not $SkipGatewayTest) {
    Write-Host ''
    Write-Host 'STEP=GatewayIntegrationTest'
    $repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
    Push-Location (Join-Path $repoRoot 'apps\api')
    try {
        go test ./internal/llm/ -run TestGatewayCompleteWithOllama -count=1 -timeout 3m
        if ($LASTEXITCODE -ne 0) {
            throw 'Gateway Ollama integration test failed.'
        }
        Write-Host 'GATEWAY_OLLAMA_INTEGRATION=PASS'
    }
    finally {
        Pop-Location
    }
}

Write-Host ''
Write-Host 'Backend env hazir. Ayni PowerShell oturumunda API/agent baslatin:'
Write-Host '  cd apps\api'
Write-Host '  go run ./cmd/server'
Write-Host 'TOKEN_EXPOSED=NO'
