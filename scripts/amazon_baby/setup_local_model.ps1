# Download (if needed), register, and start local Amazon Baby Qwen3 inference server.
# Ollama 0.34 does not support Qwen3ForCausalLM — we use a transformers OpenAI-compatible server.

[CmdletBinding()]
param(
    [string]$RepoRoot = '',
    [string]$OllamaModelName = 'miyuna-amazon-baby-qwen3',
    [int]$Port = 8765,
    [switch]$SkipInstall,
    [switch]$SkipStart
)

$ErrorActionPreference = 'Stop'
if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
    $ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
    $RepoRoot = (Resolve-Path (Join-Path $ScriptDir '..\..')).Path
}

$modelDir = Join-Path $RepoRoot 'models\amazon-baby-qwen3-finetuned'
$weightsDir = Join-Path $modelDir 'weights'
$zipPath = Join-Path $modelDir 'amazon-baby-qwen3-finetuned.zip'
$reqFile = Join-Path $RepoRoot 'scripts\amazon_baby\requirements-inference.txt'
$serverScript = Join-Path $RepoRoot 'scripts\amazon_baby\local_inference_server.py'

if (-not (Test-Path $weightsDir)) {
    if (-not (Test-Path $zipPath)) {
        Write-Host 'Downloading from Kaggle...'
        $env:PYTHONIOENCODING = 'utf-8'
        New-Item -ItemType Directory -Force -Path $modelDir | Out-Null
        py -m kaggle kernels output busenuryurdakul/amazon-baby-qwen3-0-6b-qlora-finetune `
            -p $modelDir --file-pattern '.*\.zip$' -o -q
    }
    New-Item -ItemType Directory -Force -Path $weightsDir | Out-Null
    Expand-Archive -Path $zipPath -DestinationPath $weightsDir -Force
}

if (-not (Test-Path (Join-Path $weightsDir 'model.safetensors'))) {
    throw "Weights missing under $weightsDir"
}

if (-not $SkipInstall) {
    Write-Host 'Installing inference dependencies...'
    py -m pip install -r $reqFile -q
}

$baseUrl = "http://127.0.0.1:$Port/v1"
$env:LLM_USE_MOCK = 'false'
$env:LLM_AMAZON_BABY_MODEL_NAME = $OllamaModelName
$env:LLM_AMAZON_BABY_BASE_URL = $baseUrl
$env:LLM_AMAZON_BABY_API_KEY = ''
$env:AMAZON_BABY_WEIGHTS_PATH = $weightsDir
$env:AMAZON_BABY_INFERENCE_PORT = "$Port"

if (-not $SkipStart) {
    $healthUrl = "http://127.0.0.1:$Port/health"
    $alreadyRunning = $false
    try {
        $resp = Invoke-WebRequest -Uri $healthUrl -UseBasicParsing -TimeoutSec 3
        if ($resp.StatusCode -eq 200) {
            $alreadyRunning = $true
        }
    } catch {
        $alreadyRunning = $false
    }

    if (-not $alreadyRunning) {
        Write-Host "Starting local inference server on port $Port..."
        Start-Process -FilePath 'py' -ArgumentList @(
            $serverScript
        ) -WorkingDirectory $RepoRoot -WindowStyle Hidden

        $deadline = (Get-Date).AddMinutes(3)
        while ((Get-Date) -lt $deadline) {
            Start-Sleep -Seconds 2
            try {
                $resp = Invoke-WebRequest -Uri $healthUrl -UseBasicParsing -TimeoutSec 5
                if ($resp.StatusCode -eq 200) {
                    $alreadyRunning = $true
                    break
                }
            } catch {
                continue
            }
        }
        if (-not $alreadyRunning) {
            Write-Warning "Server did not respond on $healthUrl within 3 minutes. Start manually:"
            Write-Host "  py scripts/amazon_baby/local_inference_server.py"
        }
    } else {
        Write-Host "Inference server already running on port $Port"
    }
}

Write-Host ''
Write-Host 'Amazon Baby local model ready (transformers server, not Ollama).'
Write-Host "Weights: $weightsDir"
Write-Host "LLM_AMAZON_BABY_MODEL_NAME=$OllamaModelName"
Write-Host "LLM_AMAZON_BABY_BASE_URL=$baseUrl"
Write-Host ''
Write-Host 'Start API with LLM env (separate terminal):'
Write-Host '  powershell -File scripts/amazon_baby/start_api_with_llm.ps1'
Write-Host ''
Write-Host 'Test:'
Write-Host '  py scripts/amazon_baby/rating_predict.py --product "Great stroller" --review "Love it"'
