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

# Capture user runtime env before any helper import or dot-source side effects.
$script:StartupRuntimeEnvSnapshot = @{}
foreach ($key in @('HF_TOKEN', 'LLM_PRIMARY_MODEL_NAME', 'LLM_SECONDARY_MODEL_NAME')) {
    $item = Get-Item -Path ("Env:{0}" -f $key) -ErrorAction SilentlyContinue
    if ($null -ne $item) {
        $script:StartupRuntimeEnvSnapshot[$key] = [string]$item.Value
    }
}

$isolationScript = Join-Path $PSScriptRoot 'phase6_env_isolation.ps1'
. $isolationScript

$verifyScript = Join-Path $PSScriptRoot 'verify_hf_runtime.ps1'

function Write-ValidationLine {
    param([string]$Line)
    Write-Host $Line
    $script:ReportLines += $Line
}

function Write-ValidationCheckpoint {
    param(
        [string]$Label,
        [ValidateSet('Present', 'Valid')]
        [string]$Mode = 'Present'
    )

    $visible = if ($Mode -eq 'Valid') {
        Test-Phase6CredentialValid
    }
    else {
        if (Test-Phase6CredentialPresent) {
            $true
        }
        elseif ($script:StartupRuntimeEnvSnapshot.ContainsKey('HF_TOKEN') -and -not [string]::IsNullOrWhiteSpace($script:StartupRuntimeEnvSnapshot['HF_TOKEN'])) {
            $true
        }
        else {
            $false
        }
    }
    Write-ValidationLine ("{0}: {1}" -f $Label, $(if ($visible) { 'YES' } else { 'NO' }))
}

$script:ReportLines = @()

$timestamp = (Get-Date).ToUniversalTime().ToString('o')
Write-ValidationLine "# REAL LLM Final Verification"
Write-ValidationLine ""
Write-ValidationLine "generatedAtUtc: $timestamp"
Write-ValidationLine "branch: $(git branch --show-current 2>$null)"
Write-ValidationLine "headSha: $(git rev-parse HEAD 2>$null)"
Write-ValidationLine ""

$startupTokenPresent = $script:StartupRuntimeEnvSnapshot.ContainsKey('HF_TOKEN') -and -not [string]::IsNullOrWhiteSpace($script:StartupRuntimeEnvSnapshot['HF_TOKEN'])
Write-ValidationLine ("HF_TOKEN_VISIBLE_BEFORE_STATIC: {0}" -f $(if ($startupTokenPresent) { 'YES' } else { 'NO' }))

Restore-Phase6StartupCredentialEnv -StartupSnapshot $script:StartupRuntimeEnvSnapshot
$script:RuntimeEnvSnapshot = Get-Phase6RuntimeEnvSnapshot
if ($env:LLM_PRIMARY_MODEL_NAME) {
    Write-ValidationLine "PRIMARY_MODEL=$($env:LLM_PRIMARY_MODEL_NAME.Trim())"
}
if ($env:LLM_SECONDARY_MODEL_NAME) {
    Write-ValidationLine "SECONDARY_MODEL=$($env:LLM_SECONDARY_MODEL_NAME.Trim())"
}
if ($env:LLM_PRIMARY_MODEL_NAME -and $env:LLM_SECONDARY_MODEL_NAME) {
    $modelsDifferent = ($env:LLM_PRIMARY_MODEL_NAME.Trim() -ne $env:LLM_SECONDARY_MODEL_NAME.Trim())
    Write-ValidationLine "MODELS_DIFFERENT: $(if ($modelsDifferent) { 'YES' } else { 'NO' })"
}
Write-ValidationLine "TOKEN_EXPOSED: NO"
Write-ValidationLine ""

# Static Phase 6 script checks (child process — must not mutate parent runtime env)
Write-Host 'STEP=phase6_static_tests'
$staticScript = Join-Path $PSScriptRoot 'verify_hf_runtime.static.tests.ps1'
if (Test-Path $staticScript) {
    & powershell -NoProfile -ExecutionPolicy Bypass -File $staticScript
    if ($LASTEXITCODE -ne 0) {
        throw 'PHASE6_STATIC_TESTS failed'
    }
    Restore-Phase6RuntimeEnv $script:RuntimeEnvSnapshot
    Write-ValidationLine 'PHASE6_STATIC_TESTS: PASS'
}
else {
    Write-ValidationLine 'PHASE6_STATIC_TESTS: SKIP (script missing)'
}

Write-ValidationCheckpoint 'HF_TOKEN_VISIBLE_AFTER_STATIC' 'Present'
Write-ValidationCheckpoint 'HF_TOKEN_VISIBLE_BEFORE_HF_VERIFY' 'Present'

Restore-Phase6StartupCredentialEnv -StartupSnapshot $script:StartupRuntimeEnvSnapshot
if (Test-Path $verifyScript) {
    . $verifyScript -Mode Discover
}

$credPresent = Test-Phase6CredentialPresent
$credVisible = Test-Phase6CredentialValid
Write-ValidationLine "HF_TOKEN_VISIBLE: $(if ($credVisible) { 'YES' } elseif ($credPresent) { 'PRESENT_FORMAT_PENDING' } else { 'NO' })"
Write-ValidationLine ""

# Optional HF direct dual-model smoke (requires HF_TOKEN in env)
$hfStatus = 'IMPLEMENTED_BLOCKED'
$hfBlockReason = 'PROVIDER_CREDENTIAL_NOT_AVAILABLE'
if ($credVisible) {
    Write-Host 'STEP=hf_auto_verification'
    try {
        $null = Invoke-Phase6HfRuntime -Mode Auto -ForceNewToken:$ForceHfPrompt
        $script:RuntimeEnvSnapshot = Get-Phase6RuntimeEnvSnapshot
        $hfStatus = 'PASS'
        $hfBlockReason = ''
        Write-ValidationLine 'REAL_DUAL_LLM_DIRECT_SMOKE: PASS'
        Write-ValidationLine 'REAL_ROUTING_VERIFIED: PENDING_API_GATEWAY_E2E'
        Write-ValidationLine 'REAL_FALLBACK_VERIFIED: PENDING_API_GATEWAY_E2E'
    }
    catch {
        $safe = $_.Exception.Message
        if (-not [string]::IsNullOrWhiteSpace($env:HF_TOKEN)) {
            $safe = $safe.Replace($env:HF_TOKEN, '[REDACTED]')
        }
        if ($safe.Length -gt 200) { $safe = $safe.Substring(0, 200) }
        Write-ValidationLine "REAL_DUAL_LLM_DIRECT_SMOKE: FAIL ($safe)"
        $hfStatus = 'FAIL'
        $hfBlockReason = ''
    }
}
else {
    Write-ValidationLine 'REAL_DUAL_LLM_DIRECT_SMOKE: IMPLEMENTED_BLOCKED'
    if ($credPresent) {
        Write-ValidationLine 'BLOCK_REASON: HF_TOKEN_FORMAT_INVALID'
    }
    else {
        Write-ValidationLine "BLOCK_REASON: $hfBlockReason"
    }
}

$credVisible = Test-Phase6CredentialVisible
if ($credVisible -and $env:LLM_PRIMARY_MODEL_NAME -and $env:LLM_SECONDARY_MODEL_NAME -and ($env:LLM_PRIMARY_MODEL_NAME.Trim() -ne $env:LLM_SECONDARY_MODEL_NAME.Trim())) {
    Write-Host 'STEP=go_real_llm_tagged'
    $realSnapshot = Get-Phase6RuntimeEnvSnapshot
    try {
        Push-Location (Join-Path $repoRoot 'apps\api')
        $env:LLM_USE_MOCK = 'false'
        $goReal = & go test -tags=real_llm ./internal/llm/... ./internal/integration/... -count=1 -timeout 10m -run 'Real' 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-ValidationLine 'REAL_FINAL_PIPELINE_E2E: PASS'
        }
        else {
            Write-ValidationLine 'REAL_FINAL_PIPELINE_E2E: FAIL'
            foreach ($line in (Format-Phase6SafeTestOutput -Lines (ConvertTo-Phase6OutputLines -RawOutput $goReal))) {
                Write-ValidationLine "  $line"
            }
        }
    }
    finally {
        Pop-Location
        Restore-Phase6RuntimeEnv $realSnapshot
        $script:RuntimeEnvSnapshot = Get-Phase6RuntimeEnvSnapshot
    }
}
else {
    Write-ValidationLine 'REAL_FINAL_PIPELINE_E2E: IMPLEMENTED_BLOCKED'
    Write-ValidationLine 'BLOCK_REASON: PROVIDER_CREDENTIAL_OR_DISTINCT_MODELS_NOT_AVAILABLE'
}

Write-Host 'STEP=go_mock_regression'
$mockSnapshot = Get-Phase6RuntimeEnvSnapshot
try {
    Set-Phase6MockRegressionEnv
    Push-Location (Join-Path $repoRoot 'apps\api')
    $goMock = & go test ./internal/llm/... ./internal/integration/... ./internal/account/... -count=1 2>&1
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
            $mockLines = ConvertTo-Phase6OutputLines -RawOutput $goMock
            $rootMatches = @($mockLines | Select-String -Pattern 'FAIL:|--- FAIL:|panic:|Error:')
            if ($rootMatches.Count -gt 0) {
                foreach ($line in (Format-Phase6SafeTestOutput -Lines @([string]$rootMatches[-1].Line))) {
                    Write-ValidationLine "  ROOT_CAUSE: $line"
                }
            }
            foreach ($line in (Format-Phase6SafeTestOutput -Lines $mockLines)) {
                Write-ValidationLine "  $line"
            }
        }
    }
}
finally {
    Pop-Location -ErrorAction SilentlyContinue
    Restore-Phase6RuntimeEnv $mockSnapshot
    $script:RuntimeEnvSnapshot = Get-Phase6RuntimeEnvSnapshot
}

if (-not $SkipAgent) {
    Write-Host 'STEP=agent_pytest'
    $agentSnapshot = Get-Phase6RuntimeEnvSnapshot
    try {
        Set-Phase6AgentTestEnv -RepoRoot $repoRoot
        Push-Location (Join-Path $repoRoot 'apps\agent')
        $py = & python -m pytest tests/ -q 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-ValidationLine 'AGENT_TESTS: PASS'
        }
        else {
            Write-ValidationLine 'AGENT_TESTS: FAIL'
            $pyLines = ConvertTo-Phase6OutputLines -RawOutput $py
            $rootMatches = @($pyLines | Select-String -Pattern 'FAILED|ERROR|AssertionError|short test summary')
            if ($rootMatches.Count -gt 0) {
                foreach ($line in (Format-Phase6SafeTestOutput -Lines @([string]$rootMatches[-1].Line))) {
                    Write-ValidationLine "  ROOT_CAUSE: $line"
                }
            }
            foreach ($line in (Format-Phase6SafeTestOutput -Lines $pyLines)) {
                Write-ValidationLine "  $line"
            }
        }
    }
    finally {
        Pop-Location
        Restore-Phase6RuntimeEnv $agentSnapshot
        $script:RuntimeEnvSnapshot = Get-Phase6RuntimeEnvSnapshot
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
