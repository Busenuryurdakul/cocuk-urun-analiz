# Sets HF_TOKEN for the current PowerShell session only.
# Security: never accepts token via parameters, files, or command line flags.

[CmdletBinding()]
param(
    [switch]$ForceNewToken
)

$ErrorActionPreference = 'Stop'

$verifyScript = Join-Path $PSScriptRoot 'verify_hf_runtime.ps1'
if (-not (Test-Path -LiteralPath $verifyScript)) {
    throw 'verify_hf_runtime.ps1 not found next to set_hf_token.ps1'
}

. $verifyScript -Mode Auto

$token = $null
try {
    $token = Initialize-HfSessionToken -ForcePrompt:$ForceNewToken
    Write-Host 'HF_TOKEN_SESSION_READY'
}
finally {
    Clear-HfSecretState -TokenRef ([ref]$token)
}
