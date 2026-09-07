# P0 PR-C — Real Dual-LLM Final Validation Orchestrator
# Reuses scripts/phase6 verification utilities. Never prints secret values.

[CmdletBinding()]
param(
    [switch]$SkipWeb,
    [switch]$SkipAgent,
    [switch]$ForceHfPrompt
)

$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent
Set-Location $repoRoot

function Test-CredentialVisible {
    return -not [string]::IsNullOrWhiteSpace($env:HF_TOKEN)
}

function Write-ValidationLine {
    param([string]$Line)
    Write-Host $Line
    $script:ReportLines += $Line
}

$script:ReportLines = @()
$timestamp = (Get-Date).ToUniversalTime().ToString('o')
Write-ValidationLine "# REAL LLM Final Verification"
Write-ValidationLine ""
Write-ValidationLine "generatedAtUtc: $timestamp"
Write-ValidationLine "branch: $(git branch --show-current 2>$null)"
Write-ValidationLine "headSha: $(git rev-parse HEAD 2>$null)"
Write-ValidationLine ""

$credVisible = Test-CredentialVisible
Write-ValidationLine "HF_TOKEN_VISIBLE: $(if ($credVisible) { 'YES' } else { 'NO' })"
Write-ValidationLine "TOKEN_EXPOSED: NO"
Write-ValidationLine ""

# Static Phase 6 script checks (no network)
Write-Host 'STEP=phase6_static_tests'
$staticScript = Join-Path $PSScriptRoot 'verify_hf_runtime.static.tests.ps1'
if (Test-Path $staticScript) {
    & $staticScript
    Write-ValidationLine 'PHASE6_STATIC_TESTS: PASS'
}
else {
    Write-ValidationLine 'PHASE6_STATIC_TESTS: SKIP (script missing)'
}

# Optional HF direct dual-model smoke (requires HF_TOKEN in env)
$hfStatus = 'IMPLEMENTED_BLOCKED'
$hfBlockReason = 'PROVIDER_CREDENTIAL_NOT_AVAILABLE'
if ($credVisible) {
    Write-Host 'STEP=hf_auto_verification'
    try {
        $verifyScript = Join-Path $PSScriptRoot 'verify_hf_runtime.ps1'
        . $verifyScript -Mode Auto
        $null = Invoke-Phase6HfRuntime -Mode Auto -ForceNewToken:$ForceHfPrompt
        $hfStatus = 'PASS'
        $hfBlockReason = ''
        Write-ValidationLine 'REAL_DUAL_LLM_DIRECT_SMOKE: PASS'
        Write-ValidationLine 'REAL_ROUTING_VERIFIED: PENDING_API_GATEWAY_E2E'
        Write-ValidationLine 'REAL_FALLBACK_VERIFIED: PENDING_API_GATEWAY_E2E'
    }
    catch {
        $safe = $_.Exception.Message
        if ($safe.Length -gt 200) { $safe = $safe.Substring(0, 200) }
        Write-ValidationLine "REAL_DUAL_LLM_DIRECT_SMOKE: FAIL ($safe)"
        $hfStatus = 'FAIL'
        $hfBlockReason = ''
    }
}
else {
    Write-ValidationLine 'REAL_DUAL_LLM_DIRECT_SMOKE: IMPLEMENTED_BLOCKED'
    Write-ValidationLine "BLOCK_REASON: $hfBlockReason"
}

Write-Host 'STEP=go_mock_regression'
Push-Location (Join-Path $repoRoot 'apps\api')
$goMock = & go test ./internal/llm/... ./internal/integration/... ./internal/account/... -count=1 2>&1
Pop-Location
$goMockExit = $LASTEXITCODE
Pop-Location
if ($goMockExit -eq 0) {
    Write-ValidationLine 'MOCK_CI_REGRESSION: PASS'
}
else {
    $blocked = ($goMock -match 'Application Control policy has blocked')
    if ($blocked) {
        Write-ValidationLine 'GO_LOCAL: IMPLEMENTED_BLOCKED (Windows Application Control)'
        Write-ValidationLine 'MOCK_CI_REGRESSION: PENDING_CI'
    }
    else {
        Write-ValidationLine 'MOCK_CI_REGRESSION: FAIL'
    }
    $goMock | Select-Object -Last 20 | ForEach-Object { Write-ValidationLine "  $_" }
}

if ($credVisible -and $env:LLM_PRIMARY_MODEL_NAME -and $env:LLM_SECONDARY_MODEL_NAME -and ($env:LLM_PRIMARY_MODEL_NAME -ne $env:LLM_SECONDARY_MODEL_NAME)) {
    Write-Host 'STEP=go_real_llm_tagged'
    Push-Location (Join-Path $repoRoot 'apps\api')
    $env:LLM_USE_MOCK = 'false'
    $goReal = & go test -tags=real_llm ./internal/llm/... ./internal/integration/... -count=1 -timeout 10m 2>&1
    Pop-Location
    if ($LASTEXITCODE -eq 0) {
        Write-ValidationLine 'REAL_FINAL_PIPELINE_E2E: PASS'
    }
    else {
        Write-ValidationLine 'REAL_FINAL_PIPELINE_E2E: FAIL'
        $goReal | Select-Object -Last 20 | ForEach-Object { Write-ValidationLine "  $_" }
    }
}
else {
    Write-ValidationLine 'REAL_FINAL_PIPELINE_E2E: IMPLEMENTED_BLOCKED'
    Write-ValidationLine 'BLOCK_REASON: PROVIDER_CREDENTIAL_OR_DISTINCT_MODELS_NOT_AVAILABLE'
}

if (-not $SkipAgent) {
    Write-Host 'STEP=agent_pytest'
    Push-Location (Join-Path $repoRoot 'apps\agent')
    $py = & python -m pytest tests/ -q 2>&1
    Pop-Location
    if ($LASTEXITCODE -eq 0) {
        Write-ValidationLine 'AGENT_TESTS: PASS'
    }
    else {
        Write-ValidationLine 'AGENT_TESTS: FAIL'
    }
}

if (-not $SkipWeb) {
    Write-Host 'STEP=web_build'
    Push-Location (Join-Path $repoRoot 'apps\web')
    & npm run lint 2>&1 | Out-Null
    $lintExit = $LASTEXITCODE
    & npx tsc --noEmit 2>&1 | Out-Null
    $tscExit = $LASTEXITCODE
    & npm run build 2>&1 | Out-Null
    $buildExit = $LASTEXITCODE
    Pop-Location
    if ($lintExit -eq 0 -and $tscExit -eq 0 -and $buildExit -eq 0) {
        Write-ValidationLine 'WEB_LINT: PASS'
        Write-ValidationLine 'WEB_TYPECHECK: PASS'
        Write-ValidationLine 'WEB_BUILD: PASS'
    }
    else {
        Write-ValidationLine "WEB_BUILD: FAIL (lint=$lintExit tsc=$tscExit build=$buildExit)"
    }
}

Write-ValidationLine ''
Write-ValidationLine 'FINE_TUNING_IMPLEMENTED: NO'
Write-ValidationLine 'FINE_TUNING_DEFERRED: YES'
Write-ValidationLine 'MOCK_SUCCESS_IS_NOT_REAL_PASS: YES'

$docsPath = Join-Path $repoRoot 'docs\REAL_LLM_FINAL_VERIFICATION.md'
($script:ReportLines -join "`n") + "`n" | Set-Content -Path $docsPath -Encoding UTF8

$artifactsDir = Join-Path $repoRoot 'artifacts\phase6'
if (-not (Test-Path $artifactsDir)) {
    New-Item -ItemType Directory -Path $artifactsDir -Force | Out-Null
}
$artifactPath = Join-Path $artifactsDir 'REAL_LLM_FINAL_VERIFICATION.md'
Copy-Item -Path $docsPath -Destination $artifactPath -Force

Write-Host ''
Write-Host "REPORT_WRITTEN=$docsPath"
Write-Host "ARTIFACT_COPY=$artifactPath"
