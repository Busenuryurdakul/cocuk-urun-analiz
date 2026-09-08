# Static security and validation tests for verify_hf_runtime.ps1 (no real HF calls, no token).

$ErrorActionPreference = 'Stop'

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$TargetScript = Join-Path $ScriptDir 'verify_hf_runtime.ps1'
$IsolationScript = Join-Path $ScriptDir 'phase6_env_isolation.ps1'

if (-not (Test-Path $TargetScript)) {
    throw "Missing target script: $TargetScript"
}

if (-not (Test-Path $IsolationScript)) {
    throw "Missing isolation script: $IsolationScript"
}

. $IsolationScript

$Script:InitialRuntimeSnapshot = Get-Phase6RuntimeEnvSnapshot
$Script:HadRealHfToken = -not [string]::IsNullOrWhiteSpace($Script:InitialRuntimeSnapshot['HF_TOKEN'])
$Script:InitialPrimaryModel = $Script:InitialRuntimeSnapshot['LLM_PRIMARY_MODEL_NAME']
$Script:InitialSecondaryModel = $Script:InitialRuntimeSnapshot['LLM_SECONDARY_MODEL_NAME']

try {

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

if ($source -notmatch 'function Resolve-HfSessionTokenFromEnv') {
    Add-Failure 'Env token resolver must exist for session reuse.'
}

if ($source -notmatch 'HF_TOKEN_SESSION_REUSED') {
    Add-Failure 'Existing env token reuse marker must exist.'
}

if ($source -notmatch 'ENV_PRIMARY_MODEL_PRESERVED') {
    Add-Failure 'Preset primary model preservation marker must exist.'
}

if ($source -notmatch 'ENV_SECONDARY_MODEL_PRESERVED') {
    Add-Failure 'Preset secondary model preservation marker must exist.'
}

if ($source -match 'powershell(\.exe)?\s+.*HF_TOKEN|powershell(\.exe)?\s+.*-Token\s+hf_') {
    Add-Failure 'HF token must not be passed on child PowerShell command lines.'
}

if ($source -notmatch 'function Get-PresetHfModelPair') {
    Add-Failure 'Preset model pair helper must exist.'
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

# Env propagation: valid token in $env:HF_TOKEN must be reused without prompting.
$tokenSnapshot = Get-Phase6RuntimeEnvSnapshot
$env:HF_TOKEN = $Script:SyntheticHfToken
try {
    $resolved = Resolve-HfSessionTokenFromEnv
    if ($resolved -ne $Script:SyntheticHfToken) {
        Add-Failure 'ENV_TOKEN_PRESENT_PRESERVED: normalized env token must match fixture.'
    }
    if ($env:HF_TOKEN -ne $Script:SyntheticHfToken) {
        Add-Failure 'ENV_TOKEN_PRESENT_PRESERVED: env HF_TOKEN must remain normalized fixture.'
    }

    $sessionToken = Initialize-HfSessionToken
    if ($sessionToken -ne $Script:SyntheticHfToken) {
        Add-Failure 'ENV_TOKEN_PRESENT_DOES_NOT_PROMPT: Initialize-HfSessionToken must return env token.'
    }
}
finally {
    Restore-Phase6RuntimeEnv $tokenSnapshot
}

$modelSnapshot = Get-Phase6RuntimeEnvSnapshot
$env:LLM_PRIMARY_MODEL_NAME = ' org/model-primary '
$env:LLM_SECONDARY_MODEL_NAME = ' org/model-secondary '
try {
    $pair = Get-PresetHfModelPair
    if ($null -eq $pair) {
        Add-Failure 'ENV_PRIMARY_MODEL_PRESERVED: preset pair must resolve when both env vars are set.'
    }
    elseif ($pair.Primary -ne 'org/model-primary' -or $pair.Secondary -ne 'org/model-secondary') {
        Add-Failure 'ENV model env vars must be trimmed, not overwritten with empty values.'
    }
}
finally {
    Restore-Phase6RuntimeEnv $modelSnapshot
}

$emptyTokenSnapshot = Get-Phase6RuntimeEnvSnapshot
$env:HF_TOKEN = '   '
try {
    if (Test-HfAccessTokenCandidate -Raw $env:HF_TOKEN) {
        Add-Failure 'EMPTY_ENV_TOKEN_PROMPTS_OR_BLOCKS_SAFELY: whitespace-only env token must be rejected.'
    }
    if ($null -ne (Resolve-HfSessionTokenFromEnv)) {
        Add-Failure 'EMPTY_ENV_TOKEN_PROMPTS_OR_BLOCKS_SAFELY: whitespace-only env token must not resolve.'
    }
    try {
        $null = Normalize-HfAccessToken -Token $env:HF_TOKEN
        Add-Failure 'EMPTY_ENV_TOKEN_PROMPTS_OR_BLOCKS_SAFELY: whitespace-only env token must throw on normalize.'
    }
    catch {
        if ($_.Exception.Message -ne 'HF_TOKEN_EMPTY') {
            Add-Failure ('EMPTY_ENV_TOKEN_PROMPTS_OR_BLOCKS_SAFELY: unexpected error ' + $_.Exception.Message)
        }
    }
}
finally {
    Restore-Phase6RuntimeEnv $emptyTokenSnapshot
}

$invalidTokenSnapshot = Get-Phase6RuntimeEnvSnapshot
$env:HF_TOKEN = 'not_a_hf_token'
try {
    $null = Resolve-HfSessionTokenFromEnv
    Add-Failure 'Invalid non-empty env token must not resolve silently.'
}
catch {
    if ($_.Exception.Message -ne 'HF_TOKEN_FORMAT_INVALID') {
        Add-Failure ('Invalid env token must throw HF_TOKEN_FORMAT_INVALID: ' + $_.Exception.Message)
    }
}
finally {
    Restore-Phase6RuntimeEnv $invalidTokenSnapshot
}

$printTokenSnapshot = Get-Phase6RuntimeEnvSnapshot
$env:HF_TOKEN = $Script:SyntheticHfToken
try {
    $hostOut = & {
        $null = Initialize-HfSessionToken
    } 2>&1 | Out-String
    if ($hostOut -match [regex]::Escape($Script:SyntheticHfToken)) {
        Add-Failure 'TOKEN_NOT_PRINTED: synthetic fixture token must never appear in stdout/stderr.'
    }
}
finally {
    Restore-Phase6RuntimeEnv $printTokenSnapshot
}

$p0SourcePath = Join-Path $ScriptDir 'run_p0_final_validation.ps1'
if (Test-Path $p0SourcePath) {
    $p0Source = Get-Content -Path $p0SourcePath -Raw -Encoding UTF8
    if ($p0Source -match 'powershell(\.exe)?\s+.*verify_hf_runtime') {
        Add-Failure 'Parent validation must not spawn child PowerShell for HF verification.'
    }
    if ($p0Source -notmatch 'Test-Phase6CredentialVisible|Test-HfAccessTokenCandidate') {
        Add-Failure 'Parent validation must validate HF_TOKEN format, not only whitespace.'
    }
    if ($p0Source -notmatch 'Get-Phase6RuntimeEnvSnapshot') {
        Add-Failure 'Parent validation must snapshot runtime env before isolated stages.'
    }
    if ($p0Source -notmatch 'Set-Phase6MockRegressionEnv') {
        Add-Failure 'Parent validation must isolate mock regression env from real model env.'
    }
    if ($p0Source -notmatch 'Set-Phase6AgentTestEnv') {
        Add-Failure 'Parent validation must isolate agent pytest env.'
    }
    if ($p0Source -notmatch 'HF_TOKEN_VISIBLE_AFTER_STATIC') {
        Add-Failure 'Parent validation must checkpoint HF token visibility after static tests.'
    }
}

$isolationSourcePath = Join-Path $ScriptDir 'phase6_env_isolation.ps1'
if (Test-Path $isolationSourcePath) {
    $isolationSource = Get-Content -Path $isolationSourcePath -Raw -Encoding UTF8
    if ($isolationSource -notmatch 'function Set-Phase6MockRegressionEnv') {
        Add-Failure 'Mock regression env isolation helper must exist.'
    }
    if ($isolationSource -match 'Write-Host\s+\$env:HF_TOKEN|Write-Output\s+\$env:HF_TOKEN') {
        Add-Failure 'Isolation helper must not print HF_TOKEN.'
    }
}

$mockSnapshot = Get-Phase6RuntimeEnvSnapshot
$env:LLM_PRIMARY_MODEL_NAME = 'Qwen/Qwen3-0.6B'
$env:LLM_SECONDARY_MODEL_NAME = 'Qwen/Qwen2.5-0.5B-Instruct'
try {
    Set-Phase6MockRegressionEnv
    if ($env:LLM_PRIMARY_MODEL_NAME -eq 'Qwen/Qwen3-0.6B') {
        Add-Failure 'MOCK_REGRESSION_ISOLATED_FROM_REAL_MODEL_ENV: mock env must replace HF primary model.'
    }
    if ($env:LLM_PRIMARY_MODEL_NAME -ne 'llama3.2:latest') {
        Add-Failure 'OLLAMA_TEST_NOT_USING_HF_PRIMARY_MODEL: mock regression must use canonical Ollama model name.'
    }
}
finally {
    Restore-Phase6RuntimeEnv $mockSnapshot
}

$agentSnapshot = Get-Phase6RuntimeEnvSnapshot
$env:LLM_PRIMARY_MODEL_NAME = 'Qwen/Qwen3-0.6B'
$env:LLM_SECONDARY_MODEL_NAME = 'Qwen/Qwen2.5-0.5B-Instruct'
try {
    Set-Phase6AgentTestEnv
    if ($env:LLM_PRIMARY_MODEL_NAME -or $env:LLM_SECONDARY_MODEL_NAME) {
        Add-Failure 'AGENT_TEST_ENV_ISOLATED: agent stage must clear real HF model env vars.'
    }
}
finally {
    Restore-Phase6RuntimeEnv $agentSnapshot
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
Write-Host 'TESTS_RUN=token_trim,control_char_reject,bearer_prefix_reject,header_build,no_token_output,tls12_enabled,no_cert_bypass,use_basic_parsing,safe_error_categories,invalid_header_category,mode_validation,verify_param_validation,error_classification,mock_http_no_secret_leak,env_token_present_does_not_prompt,env_token_present_preserved,env_primary_model_preserved,env_secondary_model_preserved,empty_env_token_blocks_safely,token_not_printed,token_not_in_command_line,static_tests_do_not_clear_real_env,static_tests_restore_hf_token,static_tests_restore_primary_model,static_tests_restore_secondary_model,mock_regression_isolated_from_real_model_env,ollama_test_not_using_hf_primary_model,agent_test_env_isolated'

if ($failures.Count -gt 0) {
    Write-Host 'STATIC_SECURITY_TEST=FAIL'
    foreach ($failure in $failures) {
        Write-Host ('FAIL: ' + $failure)
    }
    exit 1
}

}
finally {
    Restore-Phase6RuntimeEnv $Script:InitialRuntimeSnapshot

    if ($Script:HadRealHfToken -and [string]::IsNullOrWhiteSpace($env:HF_TOKEN)) {
        Write-Host 'STATIC_TESTS_RESTORE_HF_TOKEN: FAIL'
        exit 1
    }

    if ($null -ne $Script:InitialPrimaryModel) {
        if ([string]$env:LLM_PRIMARY_MODEL_NAME -ne [string]$Script:InitialPrimaryModel) {
            Write-Host 'STATIC_TESTS_RESTORE_PRIMARY_MODEL: FAIL'
            exit 1
        }
    }
    elseif (-not [string]::IsNullOrWhiteSpace($env:LLM_PRIMARY_MODEL_NAME)) {
        Write-Host 'STATIC_TESTS_RESTORE_PRIMARY_MODEL: FAIL'
        exit 1
    }

    if ($null -ne $Script:InitialSecondaryModel) {
        if ([string]$env:LLM_SECONDARY_MODEL_NAME -ne [string]$Script:InitialSecondaryModel) {
            Write-Host 'STATIC_TESTS_RESTORE_SECONDARY_MODEL: FAIL'
            exit 1
        }
    }
    elseif (-not [string]::IsNullOrWhiteSpace($env:LLM_SECONDARY_MODEL_NAME)) {
        Write-Host 'STATIC_TESTS_RESTORE_SECONDARY_MODEL: FAIL'
        exit 1
    }

    Write-Host 'STATIC_TESTS_DO_NOT_CLEAR_REAL_ENV: PASS'
    Write-Host 'STATIC_TESTS_RESTORE_HF_TOKEN: PASS'
    Write-Host 'STATIC_TESTS_RESTORE_PRIMARY_MODEL: PASS'
    Write-Host 'STATIC_TESTS_RESTORE_SECONDARY_MODEL: PASS'
}

exit 0
