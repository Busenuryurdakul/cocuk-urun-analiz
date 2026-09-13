# Start local Miyuna API with Amazon Baby inference env vars.
$ErrorActionPreference = 'Stop'
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = (Resolve-Path (Join-Path $ScriptDir '..\..')).Path
$ApiExe = Join-Path $RepoRoot 'apps\api\bin\miyuna-api-dev.exe'

$env:LLM_USE_MOCK = 'false'
$env:LLM_AMAZON_BABY_BASE_URL = 'http://127.0.0.1:8765/v1'
$env:LLM_AMAZON_BABY_API_KEY = ''
$env:LLM_AMAZON_BABY_MODEL_NAME = 'miyuna-amazon-baby-qwen3'
$env:LLM_PRIMARY_BASE_URL = 'http://127.0.0.1:11434/v1'
$env:LLM_PRIMARY_API_KEY = ''
$env:LLM_PRIMARY_MODEL_NAME = 'qwen2.5:0.5b'
$env:LLM_SECONDARY_BASE_URL = 'http://127.0.0.1:11434/v1'
$env:LLM_SECONDARY_API_KEY = ''
$env:LLM_SECONDARY_MODEL_NAME = 'llama3.2:latest'

if (-not $env:PORT) { $env:PORT = '8080' }
Set-Location (Join-Path $RepoRoot 'apps\api')
if (Test-Path $ApiExe) {
    try {
        & $ApiExe
        return
    } catch {
        Write-Warning "Binary launch blocked, falling back to go run: $($_.Exception.Message)"
    }
}
go run ./cmd/server
