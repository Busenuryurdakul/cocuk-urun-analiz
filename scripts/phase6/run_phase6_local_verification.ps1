# Phase 6 — Yerel LLM dogrulama (token gerekmez)
# Ollama / LM Studio / llama.cpp OpenAI-compatible endpoint

[CmdletBinding()]
param(
    [string]$PrimaryBaseUrl = 'http://127.0.0.1:11434/v1',
    [string]$SecondaryBaseUrl = 'http://127.0.0.1:11434/v1',
    [string]$ManifestPath
)

$ErrorActionPreference = 'Stop'

$localScript = Join-Path $PSScriptRoot 'verify_local_runtime.ps1'
if (-not (Test-Path -LiteralPath $localScript)) {
    throw 'verify_local_runtime.ps1 not found next to run_phase6_local_verification.ps1'
}

. $localScript -Mode Auto
Invoke-Phase6LocalRuntime -Mode Auto -PrimaryBaseUrl $PrimaryBaseUrl -SecondaryBaseUrl $SecondaryBaseUrl -ManifestPath $ManifestPath
