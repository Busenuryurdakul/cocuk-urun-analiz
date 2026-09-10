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
    'LLM_BASE_URL',
    'PYTHONPATH'
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
    param(
        [Parameter(Mandatory = $true)]
        [string]$RepoRoot
    )

    $agentSrc = Join-Path $RepoRoot 'apps\agent\src'
    if (-not (Test-Path -LiteralPath $agentSrc)) {
        throw "Agent source path not found: $agentSrc"
    }

    $env:PYTHONPATH = $agentSrc
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

function Test-Phase6CredentialPresent {
    return -not [string]::IsNullOrWhiteSpace($env:HF_TOKEN)
}

function Test-Phase6CredentialValid {
    if (-not (Test-Phase6CredentialPresent)) {
        return $false
    }
    if (Get-Command Test-HfAccessTokenCandidate -ErrorAction SilentlyContinue) {
        return (Test-HfAccessTokenCandidate -Raw $env:HF_TOKEN)
    }
    return $true
}

function Test-Phase6CredentialVisible {
    return (Test-Phase6CredentialValid)
}

function Restore-Phase6StartupCredentialEnv {
    param(
        [Parameter(Mandatory = $true)]
        [hashtable]$StartupSnapshot
    )

    if ($StartupSnapshot.ContainsKey('HF_TOKEN')) {
        Set-Item -Path 'Env:HF_TOKEN' -Value $StartupSnapshot['HF_TOKEN']
    }
    if ($StartupSnapshot.ContainsKey('LLM_PRIMARY_MODEL_NAME')) {
        Set-Item -Path 'Env:LLM_PRIMARY_MODEL_NAME' -Value $StartupSnapshot['LLM_PRIMARY_MODEL_NAME']
    }
    if ($StartupSnapshot.ContainsKey('LLM_SECONDARY_MODEL_NAME')) {
        Set-Item -Path 'Env:LLM_SECONDARY_MODEL_NAME' -Value $StartupSnapshot['LLM_SECONDARY_MODEL_NAME']
    }
}

function Write-Phase6CredentialCheckpoint {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Label,

        [ValidateSet('Present', 'Valid')]
        [string]$Mode = 'Present'
    )

    $visible = if ($Mode -eq 'Valid') {
        Test-Phase6CredentialValid
    }
    else {
        Test-Phase6CredentialPresent
    }
    Write-Host ("{0}: {1}" -f $Label, $(if ($visible) { 'YES' } else { 'NO' }))
}

function ConvertTo-Phase6OutputLines {
    param(
        [AllowNull()]
        $RawOutput
    )

    if ($null -eq $RawOutput) {
        return @()
    }

    if ($RawOutput -is [System.Array]) {
        return @(
            $RawOutput |
                ForEach-Object {
                    if ($null -eq $_) { return }
                    [string]$_
                } |
                Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
        )
    }

    $text = [string]$RawOutput
    if ([string]::IsNullOrWhiteSpace($text)) {
        return @()
    }

    return @(
        $text -split "`r?`n" |
            Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
    )
}

function Format-Phase6SafeTestOutput {
    param(
        [AllowNull()]
        [AllowEmptyCollection()]
        [string[]]$Lines = @(),

        [int]$MaxLines = 12
    )

    $normalized = ConvertTo-Phase6OutputLines -RawOutput $Lines
    if ($normalized.Count -eq 0) {
        return @()
    }

    $safeLines = @()
    foreach ($line in ($normalized | Select-Object -Last $MaxLines)) {
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

function Write-Phase6SafeOutputLines {
    param(
        [AllowNull()]
        $RawOutput,

        [string]$Prefix = '  ',
        [string]$RootCausePrefix = '  ROOT_CAUSE: '
    )

    $lines = ConvertTo-Phase6OutputLines -RawOutput $RawOutput
    if ($lines.Count -eq 0) {
        return
    }

    $rootCause = ($lines | Select-String -Pattern 'FAIL:|ERROR:|AssertionError|short test summary|panic:' | Select-Object -Last 1)
    if ($null -ne $rootCause -and -not [string]::IsNullOrWhiteSpace([string]$rootCause.Line)) {
        foreach ($line in (Format-Phase6SafeTestOutput -Lines @([string]$rootCause.Line))) {
            Write-Host ($RootCausePrefix + $line)
        }
    }

    foreach ($line in (Format-Phase6SafeTestOutput -Lines $lines)) {
        Write-Host ($Prefix + $line)
    }
}
