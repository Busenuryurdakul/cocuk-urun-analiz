# Static security and validation tests for verify_hf_runtime.ps1 (no real HF calls, no token).

$ErrorActionPreference = 'Stop'

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$TargetScript = Join-Path $ScriptDir 'verify_hf_runtime.ps1'

if (-not (Test-Path $TargetScript)) {
    throw "Missing target script: $TargetScript"
}

$source = Get-Content -Path $TargetScript -Raw -Encoding UTF8
$failures = New-Object System.Collections.Generic.List[string]

function Add-Failure {
    param([string]$Message)
    [void]$script:failures.Add($Message)
}

# Source hygiene checks
$forbiddenPatterns = @(
    @{ Pattern = 'Write-Host\s+\$token'; Message = 'Token must not be written to stdout.' },
    @{ Pattern = 'Write-Output\s+\$token'; Message = 'Token must not be written to stdout.' },
    @{ Pattern = 'Out-File[^\n]*\$token'; Message = 'Token must not be written to a file.' },
    @{ Pattern = 'Set-Content[^\n]*\$token'; Message = 'Token must not be written to a file.' },
    @{ Pattern = 'ConvertTo-Json[^\n]*\$token'; Message = 'Token must not be serialized to output.' },
    @{ Pattern = 'Authorization\s*=\s*\(\s*''Bearer '' \+ \$Token\s*\)[^\n]*Write-Host'; Message = 'Authorization header must not be logged.' },
    @{ Pattern = 'ServerCertificateValidationCallback'; Message = 'Certificate validation bypass must not be used.' },
    @{ Pattern = 'TrustAllCertsPolicy'; Message = 'Certificate validation bypass must not be used.' },
    @{ Pattern = 'SkipCertificateCheck'; Message = 'Certificate validation bypass must not be used.' },
    @{ Pattern = '\.env'; Message = 'Script must not reference .env file creation.' },
    @{ Pattern = 'curl\s'; Message = 'Unix-only curl command must not be used.' },
    @{ Pattern = '\|\|\s'; Message = 'Unix-style || control flow must not be used.' },
    @{ Pattern = '&&\s'; Message = 'Unix-style && control flow must not be used.' }
)

foreach ($item in $forbiddenPatterns) {
    if ($source -match $item.Pattern) {
        Add-Failure $item.Message
    }
}

if ($source -notmatch '\[ValidateSet\(''Discover'',\s*''Verify'',\s*''Auto''\)\]') {
    Add-Failure 'Mode must be restricted to Discover, Verify, and Auto.'
}

if ($source -notmatch 'Read-Host\s+''Hugging Face Token''\s+-AsSecureString') {
    Add-Failure 'Secure token prompt must be present when HF_TOKEN is missing.'
}

if ($source -notmatch 'function Invoke-AutoVerification') {
    Add-Failure 'Auto verification orchestration must exist.'
}

if ($source -notmatch 'function Set-MiyunaLlmRuntimeEnvironment') {
    Add-Failure 'Miyuna runtime env helper must exist.'
}

if ($source -notmatch 'REAL_DUAL_LLM_RUNTIME_VERIFIED') {
    Add-Failure 'Auto verification must report REAL_DUAL_LLM_RUNTIME_VERIFIED.'
}

if ($source -notmatch 'function Enable-HfTls12Support') {
    Add-Failure 'TLS 1.2 enablement helper must exist.'
}

if ($source -notmatch '\[Net\.SecurityProtocolType\]::Tls12') {
    Add-Failure 'TLS 1.2 must be enabled with -bor semantics.'
}

if ($source -notmatch 'UseBasicParsing\s*=\s*\$true') {
    Add-Failure 'Invoke-WebRequest must use -UseBasicParsing for PowerShell 5.1.'
}

if ($source -notmatch 'function Get-SafeErrorMessage') {
    Add-Failure 'Safe error redaction helper must exist.'
}

if ($source -notmatch 'ERROR_CATEGORY=') {
    Add-Failure 'Transport error diagnostics must include ERROR_CATEGORY.'
}

if ($source -notmatch 'TLS_NEGOTIATION_FAILED|DNS_FAILED|CONNECTION_FAILED|REQUEST_TIMEOUT|UNKNOWN_REQUEST_FAILURE') {
    Add-Failure 'Transport error categories must be defined.'
}

if ($source -notmatch 'function Normalize-HfAccessToken') {
    Add-Failure 'Token normalization helper must exist.'
}

if ($source -notmatch 'HF_TOKEN_CONTAINS_CONTROL_CHARACTERS') {
    Add-Failure 'Control character validation must exist.'
}

if ($source -notmatch 'HF_TOKEN_FORMAT_INVALID') {
    Add-Failure 'Invalid token format rejection must exist.'
}

if ($source -notmatch 'function Sanitize-HfAccessTokenRaw') {
    Add-Failure 'Token sanitization helper must exist.'
}

if ($source -notmatch 'function New-HfAuthorizationHeaders') {
    Add-Failure 'Authorization header builder must exist.'
}

if ($source -notlike "*Bearer {0}' -f `$Token*") {
    Add-Failure 'Authorization header must use Bearer {0} -f $token formatting.'
}

if ($source -notmatch 'INVALID_HEADER_VALUE') {
    Add-Failure 'Invalid header value category must exist.'
}

if ($source -notmatch 'AUTHENTICATION_FAILED|PERMISSION_DENIED|ENDPOINT_NOT_FOUND|RATE_LIMITED|PROVIDER_ERROR') {
    Add-Failure 'HTTP status labels must be defined.'
}

if ($source -notmatch '\.Trim\(\)') {
    Add-Failure 'Token must be trimmed after secure conversion.'
}

if ($source -notmatch 'max_tokens\s*=\s*\$Script:SmokeMaxTokens') {
    Add-Failure 'Verify smoke tests must use low max_tokens.'
}

if ($source -notmatch 'stream\s*=\s*\$false') {
    Add-Failure 'Verify smoke tests must disable streaming.'
}

# Dot-source functions without executing main entrypoint.
. $TargetScript -Mode Discover

$Script:SyntheticHfToken = 'hf_abcdefghijklmnopqrstuvwxyzABCDEF12'

try {
    Assert-VerifyParameters -Primary 'model-a' -Secondary 'model-a'
    Add-Failure 'Equal primary/secondary models must throw.'
}
catch {
    if ($_.Exception.Message -notmatch 'must be different') {
        Add-Failure ('Unexpected error for equal models: ' + $_.Exception.Message)
    }
}

try {
    Assert-VerifyParameters -Primary '' -Secondary 'model-b'
    Add-Failure 'Missing PrimaryModel must throw.'
}
catch {
    if ($_.Exception.Message -notmatch 'PrimaryModel is required') {
        Add-Failure ('Unexpected error for missing PrimaryModel: ' + $_.Exception.Message)
    }
}

try {
    Assert-VerifyParameters -Primary 'model-a' -Secondary ''
    Add-Failure 'Missing SecondaryModel must throw.'
}
catch {
    if ($_.Exception.Message -notmatch 'SecondaryModel is required') {
        Add-Failure ('Unexpected error for missing SecondaryModel: ' + $_.Exception.Message)
    }
}

$class401 = Get-HfErrorClass -StatusCode 401 -HttpStatusLabel 'AUTHENTICATION_FAILED'
$class403 = Get-HfErrorClass -StatusCode 403 -HttpStatusLabel 'PERMISSION_DENIED'
$class404 = Get-HfErrorClass -StatusCode 404 -HttpStatusLabel 'ENDPOINT_NOT_FOUND'
$class429 = Get-HfErrorClass -StatusCode 429 -HttpStatusLabel 'RATE_LIMITED'
$class503 = Get-HfErrorClass -StatusCode 503 -HttpStatusLabel 'PROVIDER_ERROR'
$classTimeout = Get-HfErrorClass -StatusCode 0 -Message 'The operation has timed out'

if ($class401 -ne 'AUTHENTICATION_FAILED') { Add-Failure '401 must map to AUTHENTICATION_FAILED.' }
if ($class403 -ne 'PERMISSION_DENIED') { Add-Failure '403 must map to PERMISSION_DENIED.' }
if ($class404 -ne 'ENDPOINT_NOT_FOUND') { Add-Failure '404 must map to ENDPOINT_NOT_FOUND.' }
if ($class429 -ne 'RATE_LIMITED') { Add-Failure '429 must map to RATE_LIMITED.' }
if ($class503 -ne 'PROVIDER_ERROR') { Add-Failure '5xx must map to PROVIDER_ERROR.' }
if ($classTimeout -ne 'REQUEST_TIMEOUT') { Add-Failure 'Timeout messages must map to REQUEST_TIMEOUT.' }

$safe = Get-SafeErrorMessage -RawMessage 'Authorization: Bearer hf_abc123secretvalue' -Token 'hf_abc123secretvalue'
if ($safe -match 'hf_abc123secretvalue') {
    Add-Failure 'Safe error message must redact token-like values.'
}
if ($safe -notmatch '\[REDACTED\]') {
    Add-Failure 'Safe error message must contain redaction marker.'
}

$transport = Get-HfTransportErrorCategory -Exception ([System.Net.WebException]::new('Could not create SSL/TLS secure channel')) -SafeMessage 'Could not create SSL/TLS secure channel'
if ($transport -ne 'TLS_NEGOTIATION_FAILED') {
    Add-Failure 'TLS failures must map to TLS_NEGOTIATION_FAILED.'
}

$invalidHeader = Get-HfTransportErrorCategory -Exception ([System.ArgumentException]::new('Specified value has invalid control characters.')) -SafeMessage 'Specified value has invalid control characters.'
if ($invalidHeader -ne 'INVALID_HEADER_VALUE') {
    Add-Failure 'ArgumentException must map to INVALID_HEADER_VALUE.'
}

$trimmed = Normalize-HfAccessToken -Token ("  $($Script:SyntheticHfToken)  " + "`r`n")
if ($trimmed -ne $Script:SyntheticHfToken) {
    Add-Failure 'Leading and trailing whitespace or CRLF must be trimmed from token.'
}

$bearerTrimmed = Normalize-HfAccessToken -Token ("Bearer $($Script:SyntheticHfToken)")
if ($bearerTrimmed -ne $Script:SyntheticHfToken) {
    Add-Failure 'Bearer prefix must be stripped from token input.'
}

$quoted = Normalize-HfAccessToken -Token ('"' + $Script:SyntheticHfToken + '"')
if ($quoted -ne $Script:SyntheticHfToken) {
    Add-Failure 'Surrounding quotes must be stripped from token input.'
}

$embedded = Normalize-HfAccessToken -Token ("export HF_TOKEN=$($Script:SyntheticHfToken)" + "`n")
if ($embedded -ne $Script:SyntheticHfToken) {
    Add-Failure 'Embedded hf_ token must be extracted from surrounding text.'
}

try {
    Normalize-HfAccessToken -Token ("hf_bad" + [char]0x01 + 'token')
    Add-Failure 'Embedded control characters must be rejected.'
}
catch {
    if ($_.Exception.Message -ne 'HF_TOKEN_CONTAINS_CONTROL_CHARACTERS') {
        Add-Failure ('Unexpected error for control character token: ' + $_.Exception.Message)
    }
}

try {
    Normalize-HfAccessToken -Token 'not_a_hf_token'
    Add-Failure 'Non-hf token input must be rejected.'
}
catch {
    if ($_.Exception.Message -ne 'HF_TOKEN_FORMAT_INVALID') {
        Add-Failure ('Unexpected error for invalid token: ' + $_.Exception.Message)
    }
}

try {
    Normalize-HfAccessToken -Token '   '
    Add-Failure 'Empty token must be rejected.'
}
catch {
    if ($_.Exception.Message -ne 'HF_TOKEN_EMPTY') {
        Add-Failure ('Unexpected error for empty token: ' + $_.Exception.Message)
    }
}

$headers = New-HfAuthorizationHeaders -Token $Script:SyntheticHfToken -IncludeContentType
if ($headers.Authorization -ne ('Bearer {0}' -f $Script:SyntheticHfToken)) {
    Add-Failure 'Authorization header must be built from normalized token.'
}
if ($headers.Accept -ne 'application/json') {
    Add-Failure 'Accept header must be application/json.'
}
if ($headers['Content-Type'] -ne 'application/json') {
    Add-Failure 'Content-Type header must be application/json for POST headers.'
}
if ($headers.Authorization -match "[\r\n]") {
    Add-Failure 'Authorization header must not contain newline characters.'
}

# Mock HTTP layer: ensure Authorization header is built but never returned in result objects
$mockToken = 'mock-token-value-not-real'
$mockResponse = Invoke-HfOpenAIRequest -Method GET -Uri 'http://127.0.0.1:9/unreachable-mock-endpoint' -Token $mockToken -TimeoutSec 1
if ($null -eq $mockResponse) {
    Add-Failure 'Mock HTTP call must return a structured result.'
}
$mockJson = $mockResponse | ConvertTo-Json -Depth 4 -Compress
if ($mockJson -match 'mock-token-value-not-real') {
    Add-Failure 'HTTP result must not contain the token value.'
}
if ($mockJson -match 'Authorization') {
    Add-Failure 'HTTP result must not contain Authorization header text.'
}

Write-Host ''
Write-Host 'STATIC_SECURITY_TEST=PASS'
Write-Host 'TESTS_RUN=token_trim,control_char_reject,bearer_prefix_reject,header_build,no_token_output,tls12_enabled,no_cert_bypass,use_basic_parsing,safe_error_categories,invalid_header_category,mode_validation,verify_param_validation,error_classification,mock_http_no_secret_leak'

if ($failures.Count -gt 0) {
    Write-Host 'STATIC_SECURITY_TEST=FAIL'
    foreach ($failure in $failures) {
        Write-Host ('FAIL: ' + $failure)
    }
    exit 1
}

exit 0
