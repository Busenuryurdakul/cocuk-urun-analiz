# Phase 6 — process-scoped runtime env snapshot/restore helpers.
# Never logs secret values.

$Script:Phase6IsolationKeys = @(
    'HF_TOKEN',
    'LLM_PRIMARY_MODEL_NAME',
    'LLM_SECONDARY_MODEL_NAME',
    'LLM_PRIMARY_BASE_URL',
    'LLM_SECONDARY_BASE_URL',
    'LLM_PRIMARY_API_KEY',
    'LLM_SECONDARY_API_KEY',
    'LLM_USE_MOCK',
    'LLM_PROVIDER',
    'LLM_BASE_URL'
)

function Get-Phase6RuntimeEnvSnapshot {
    $snapshot = @{}
    foreach ($key in $Script:Phase6IsolationKeys) {
        $item = Get-Item -Path ("Env:{0}" -f $key) -ErrorAction SilentlyContinue
        if ($null -ne $item) {
            $snapshot[$key] = [string]$item.Value
        }
    }
    return $snapshot
}

function Restore-Phase6RuntimeEnv {
    param(
        [Parameter(Mandatory = $true)]
        [hashtable]$Snapshot
    )

    foreach ($key in $Script:Phase6IsolationKeys) {
        if ($Snapshot.ContainsKey($key)) {
            Set-Item -Path ("Env:{0}" -f $key) -Value $Snapshot[$key]
        }
        else {
            Remove-Item -Path ("Env:{0}" -f $key) -ErrorAction SilentlyContinue
        }
    }
}

function Clear-Phase6RuntimeEnvKeys {
    foreach ($key in $Script:Phase6IsolationKeys) {
        Remove-Item -Path ("Env:{0}" -f $key) -ErrorAction SilentlyContinue
    }
}

function Set-Phase6MockRegressionEnv {
    $env:LLM_USE_MOCK = 'true'
    $env:LLM_PRIMARY_BASE_URL = 'http://127.0.0.1:11434/v1'
    $env:LLM_SECONDARY_BASE_URL = 'http://127.0.0.1:11434/v1'
    $env:LLM_PRIMARY_MODEL_NAME = 'llama3.2:latest'
    $env:LLM_SECONDARY_MODEL_NAME = 'llama3.2:latest'
    Remove-Item Env:LLM_PRIMARY_API_KEY -ErrorAction SilentlyContinue
    Remove-Item Env:LLM_SECONDARY_API_KEY -ErrorAction SilentlyContinue
    Remove-Item Env:LLM_PROVIDER -ErrorAction SilentlyContinue
    Remove-Item Env:LLM_BASE_URL -ErrorAction SilentlyContinue
}

function Set-Phase6AgentTestEnv {
    $env:LLM_USE_MOCK = 'true'
    Remove-Item Env:LLM_PRIMARY_MODEL_NAME -ErrorAction SilentlyContinue
    Remove-Item Env:LLM_SECONDARY_MODEL_NAME -ErrorAction SilentlyContinue
    Remove-Item Env:LLM_PRIMARY_BASE_URL -ErrorAction SilentlyContinue
    Remove-Item Env:LLM_SECONDARY_BASE_URL -ErrorAction SilentlyContinue
    Remove-Item Env:LLM_PRIMARY_API_KEY -ErrorAction SilentlyContinue
    Remove-Item Env:LLM_SECONDARY_API_KEY -ErrorAction SilentlyContinue
    Remove-Item Env:LLM_PROVIDER -ErrorAction SilentlyContinue
    Remove-Item Env:LLM_BASE_URL -ErrorAction SilentlyContinue
}

function Test-Phase6CredentialVisible {
    if ([string]::IsNullOrWhiteSpace($env:HF_TOKEN)) {
        return $false
    }
    if (Get-Command Test-HfAccessTokenCandidate -ErrorAction SilentlyContinue) {
        return (Test-HfAccessTokenCandidate -Raw $env:HF_TOKEN)
    }
    return $true
}

function Write-Phase6CredentialCheckpoint {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Label
    )

    $visible = Test-Phase6CredentialVisible
    Write-Host ("{0}: {1}" -f $Label, $(if ($visible) { 'YES' } else { 'NO' }))
}

function Format-Phase6SafeTestOutput {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Lines,

        [int]$MaxLines = 12
    )

    $safeLines = @()
    foreach ($line in ($Lines | Select-Object -Last $MaxLines)) {
        $safe = [string]$line
        $safe = [regex]::Replace($safe, '(?i)\bBearer\s+\S+', 'Bearer [REDACTED]')
        $safe = [regex]::Replace($safe, '(?i)\bhf_[A-Za-z0-9]{8,}\b', 'hf_[REDACTED]')
        if (-not [string]::IsNullOrWhiteSpace($env:HF_TOKEN)) {
            $safe = $safe.Replace($env:HF_TOKEN, '[REDACTED]')
        }
        $safeLines += $safe
    }
    return $safeLines
}
