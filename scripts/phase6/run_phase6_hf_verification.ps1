# Phase 6 — Tek komut Hugging Face dogrulama
# Kullanim: token'i terminalde bir kez girin; geri kalan otomatik calisir.

[CmdletBinding()]
param(
    [switch]$ForceNewToken
)

$ErrorActionPreference = 'Stop'

$verifyScript = Join-Path $PSScriptRoot 'verify_hf_runtime.ps1'
if (-not (Test-Path -LiteralPath $verifyScript)) {
    throw 'verify_hf_runtime.ps1 not found next to run_phase6_hf_verification.ps1'
}

. $verifyScript -Mode Auto
Invoke-Phase6HfRuntime -Mode Auto -ForceNewToken:$ForceNewToken
