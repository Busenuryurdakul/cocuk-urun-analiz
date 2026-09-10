# Phase 6 — Hugging Face Inference Providers runtime verification (interactive session only).
# Security: never accepts token via parameters, files, or command line flags.
# Compatible with Windows PowerShell 5.1+

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('Discover', 'Verify', 'Auto')]
    [string]$Mode,

    [string]$PrimaryModel,
    [string]$SecondaryModel,

    [switch]$ForceNewToken
)

$ErrorActionPreference = 'Stop'

$Script:HfModelsEndpoint = 'https://router.huggingface.co/v1/models'
$Script:HfChatEndpoint = 'https://router.huggingface.co/v1/chat/completions'
$Script:MaxDiscoverCandidates = 20
$Script:SmokeMaxTokens = 16

function Enable-HfTls12Support {
    if ([Net.ServicePointManager]::SecurityProtocol -band [Net.SecurityProtocolType]::Tls12) {
        return
    }

    [Net.ServicePointManager]::SecurityProtocol =
        [Net.ServicePointManager]::SecurityProtocol -bor
        [Net.SecurityProtocolType]::Tls12
}

function Get-SafeErrorMessage {
    param(
        [string]$RawMessage,
        [string]$Token = ''
    )

    if ([string]::IsNullOrWhiteSpace($RawMessage)) {
        return ''
    }

    $safe = [string]$RawMessage
    $safe = [regex]::Replace($safe, '(?i)\bBearer\s+\S+', 'Bearer [REDACTED]')
    $safe = [regex]::Replace($safe, '(?i)\bAuthorization\s*[:=]\s*\S+', 'Authorization: [REDACTED]')
    $safe = [regex]::Replace($safe, '(?i)\bhf_[A-Za-z0-9]{8,}\b', 'hf_[REDACTED]')
    $safe = [regex]::Replace($safe, '(?i)\$env:HF_TOKEN', '$env:HF_TOKEN=[REDACTED]')

    if (-not [string]::IsNullOrWhiteSpace($Token)) {
        $safe = $safe.Replace($Token, '[REDACTED]')
    }
    if (-not [string]::IsNullOrWhiteSpace($env:HF_TOKEN)) {
        $safe = $safe.Replace($env:HF_TOKEN, '[REDACTED]')
    }

    if ($safe.Length -gt 300) {
        $safe = $safe.Substring(0, 300)
    }
    return $safe
}

function Get-HfHttpStatusLabel {
    param([int]$StatusCode)

    switch ($StatusCode) {
        401 { return 'AUTHENTICATION_FAILED' }
        403 { return 'PERMISSION_DENIED' }
        404 { return 'ENDPOINT_NOT_FOUND' }
        429 { return 'RATE_LIMITED' }
        default {
            if ($StatusCode -ge 500) {
                return 'PROVIDER_ERROR'
            }
            if ($StatusCode -gt 0) {
                return 'HTTP_ERROR'
            }
            return $null
        }
    }
}

function Get-HfTransportErrorCategory {
    param(
        [System.Exception]$Exception,
        [string]$SafeMessage
    )

    $webStatus = $null
    if ($Exception -is [System.Net.WebException]) {
        $webStatus = $Exception.Status
    }

    if ($SafeMessage -match '(?i)tls|ssl|secure channel|could not create ssl') {
        return 'TLS_NEGOTIATION_FAILED'
    }
    if ($webStatus -eq [System.Net.WebExceptionStatus]::NameResolutionFailure) {
        return 'DNS_FAILED'
    }
    if ($webStatus -eq [System.Net.WebExceptionStatus]::ConnectFailure) {
        return 'CONNECTION_FAILED'
    }
    if ($webStatus -eq [System.Net.WebExceptionStatus]::ProxyNameResolutionFailure -or
        $webStatus -eq [System.Net.WebExceptionStatus]::SendFailure) {
        return 'PROXY_FAILED'
    }
    if ($webStatus -eq [System.Net.WebExceptionStatus]::Timeout -or
        $SafeMessage -match '(?i)timed out|timeout|operation has timed out') {
        return 'REQUEST_TIMEOUT'
    }
    if ($SafeMessage -match '(?i)invoke-webrequest|parser|unexpected token|cannot bind parameter') {
        return 'POWERSHELL_HTTP_PARSING_FAILED'
    }
    if ($Exception -is [System.ArgumentException]) {
        return 'INVALID_HEADER_VALUE'
    }
    if ($SafeMessage -match '(?i)control character|invalid.*header|geçersiz kontrol') {
        return 'INVALID_HEADER_VALUE'
    }
    return 'UNKNOWN_REQUEST_FAILURE'
}

function Write-HfRequestFailureDiagnostics {
    param(
        [Parameter(Mandatory = $true)]
        $Response
    )

    Write-Host ''
    Write-Host 'HF_REQUEST_FAILURE_DIAGNOSTICS'
    if ($Response.StatusCode -gt 0) {
        Write-Host ('HTTP_STATUS={0}' -f $Response.StatusCode)
        if (-not [string]::IsNullOrWhiteSpace([string]$Response.HttpStatusLabel)) {
            Write-Host ('HTTP_STATUS_LABEL={0}' -f $Response.HttpStatusLabel)
        }
    }
    else {
        if (-not [string]::IsNullOrWhiteSpace([string]$Response.ErrorCategory)) {
            Write-Host ('ERROR_CATEGORY={0}' -f $Response.ErrorCategory)
        }
        if (-not [string]::IsNullOrWhiteSpace([string]$Response.ExceptionType)) {
            Write-Host ('EXCEPTION_TYPE={0}' -f $Response.ExceptionType)
        }
        if (-not [string]::IsNullOrWhiteSpace([string]$Response.InnerExceptionType)) {
            Write-Host ('INNER_EXCEPTION_TYPE={0}' -f $Response.InnerExceptionType)
        }
        if ($null -ne $Response.WebExceptionStatus) {
            Write-Host ('WEB_EXCEPTION_STATUS={0}' -f $Response.WebExceptionStatus)
        }
    }
    if (-not [string]::IsNullOrWhiteSpace([string]$Response.SafeErrorMessage)) {
        Write-Host ('SAFE_ERROR_MESSAGE={0}' -f $Response.SafeErrorMessage)
    }
    Write-Host 'TOKEN_EXPOSED=NO'
}

function Get-RepoRoot {
    return Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
}

function Sanitize-HfAccessTokenRaw {
    param(
        [Parameter(Mandatory = $true)]
        [AllowNull()]
        [string]$Raw
    )

    if ($null -eq $Raw) {
        return ''
    }

    $candidate = [string]$Raw
    $candidate = $candidate -replace '\x00', ''
    $candidate = $candidate.Trim()
    $candidate = ($candidate -replace '^[\x00-\x1F\x7F]+|[\x00-\x1F\x7F]+$', '').Trim()

    if ($candidate -match '^["''](.+)["'']$') {
        $candidate = $Matches[1].Trim()
    }

    if ($candidate -match '(?i)^Bearer\s*(.+)$') {
        $candidate = $Matches[1].Trim()
    }

    $candidate = ($candidate -replace '\s+', '')

    if ($candidate -match '(?i)^(hf_[A-Za-z0-9]+)$') {
        return $Matches[1]
    }

    if ($candidate -match '(?i)(hf_[A-Za-z0-9]{10,})') {
        return $Matches[1]
    }

    return $candidate
}

function Normalize-HfAccessToken {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Token
    )

    $token = Sanitize-HfAccessTokenRaw -Raw $Token

    if ([string]::IsNullOrWhiteSpace($token)) {
        throw 'HF_TOKEN_EMPTY'
    }

    if ($token -match '[\x00-\x1F\x7F]') {
        throw 'HF_TOKEN_CONTAINS_CONTROL_CHARACTERS'
    }

    if ($token -notmatch '^hf_[A-Za-z0-9]+$') {
        throw 'HF_TOKEN_FORMAT_INVALID'
    }

    return $token
}

function Test-HfAccessTokenCandidate {
    param(
        [Parameter(Mandatory = $true)]
        [AllowNull()]
        [string]$Raw
    )

    if ([string]::IsNullOrWhiteSpace($Raw)) {
        return $false
    }

    try {
        $null = Normalize-HfAccessToken -Token $Raw
        return $true
    }
    catch {
        return $false
    }
}

function ConvertFrom-HfSecureString {
    param(
        [Parameter(Mandatory = $true)]
        [System.Security.SecureString]$Secure
    )

    $bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($Secure)
    try {
        $plain = [Runtime.InteropServices.Marshal]::PtrToStringAuto($bstr)
        if ($null -ne $plain) {
            $plain = $plain -replace '\x00', ''
        }
        return [string]$plain
    }
    finally {
        [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)
        $Secure.Dispose()
    }
}

function Get-HfAccessTokenFromClipboard {
    if (-not (Get-Command Get-Clipboard -ErrorAction SilentlyContinue)) {
        return $null
    }

    try {
        $clipRaw = Get-Clipboard -Raw -ErrorAction Stop
    }
    catch {
        return $null
    }

    if ([string]::IsNullOrWhiteSpace($clipRaw)) {
        return $null
    }

    if (-not (Test-HfAccessTokenCandidate -Raw $clipRaw)) {
        return $null
    }

    return Normalize-HfAccessToken -Token $clipRaw
}

function Request-HfAccessTokenInteractive {
    Write-Host ''
    Write-Host '============================================================'
    Write-Host ' Hugging Face Token (tek seferlik, gizli giris)'
    Write-Host ' Fine-grained token: yalnizca ham hf_... (Bearer/on ek yok)'
    Write-Host '============================================================'
    Write-Host ''

    $secure = Read-Host 'Hugging Face Token' -AsSecureString
    if ($null -eq $secure) {
        throw 'HF_TOKEN_ENTRY_CANCELLED'
    }

    $plain = ConvertFrom-HfSecureString -Secure $secure
    $token = Normalize-HfAccessToken -Token $plain
    $env:HF_TOKEN = $token
    Write-Host 'HF_TOKEN_ACCEPTED (deger loglanmadi)'
    return $token
}

function Resolve-HfSessionTokenFromEnv {
    if ([string]::IsNullOrWhiteSpace($env:HF_TOKEN)) {
        return $null
    }

    if (-not (Test-HfAccessTokenCandidate -Raw $env:HF_TOKEN)) {
        throw 'HF_TOKEN_FORMAT_INVALID'
    }

    $normalized = Normalize-HfAccessToken -Token $env:HF_TOKEN
    $env:HF_TOKEN = $normalized
    return $normalized
}

function Initialize-HfSessionToken {
    param(
        [switch]$ForcePrompt
    )

    if (-not $ForcePrompt) {
        $fromEnv = Resolve-HfSessionTokenFromEnv
        if ($null -ne $fromEnv) {
            Write-Host 'HF_TOKEN_SESSION_REUSED'
            return $fromEnv
        }

        $clipboardToken = Get-HfAccessTokenFromClipboard
        if (-not [string]::IsNullOrWhiteSpace($clipboardToken)) {
            Write-Host 'HF_TOKEN_CLIPBOARD_REUSED (value not logged)'
            $env:HF_TOKEN = $clipboardToken
            return $clipboardToken
        }
    }

    return Request-HfAccessTokenInteractive
}

function Get-HfAccessToken {
    param(
        [ValidateSet('EnvFirst', 'PromptOnly')]
        [string]$Source = 'EnvFirst'
    )

    if ($Source -eq 'PromptOnly') {
        return Initialize-HfSessionToken -ForcePrompt
    }

    try {
        $fromEnv = Resolve-HfSessionTokenFromEnv
        if ($null -ne $fromEnv) {
            Write-Host 'HF_TOKEN_SESSION_REUSED'
            return $fromEnv
        }
    }
    catch {
        throw
    }

    Write-Host 'HF_TOKEN not found in the current process environment.'

    $clipboardToken = Get-HfAccessTokenFromClipboard
    if (-not [string]::IsNullOrWhiteSpace($clipboardToken)) {
        Write-Host 'HF_TOKEN_CLIPBOARD_REUSED (value not logged)'
        $env:HF_TOKEN = $clipboardToken
        return $clipboardToken
    }

    return Request-HfAccessTokenInteractive
}

function Set-HfTokenForSession {
    $token = Initialize-HfSessionToken
    Write-Host 'HF_TOKEN_SESSION_READY'
    return $token
}

function Clear-HfSecretState {
    param(
        [ref]$TokenRef
    )

    if ($null -ne $TokenRef -and $null -ne $TokenRef.Value) {
        $TokenRef.Value = $null
    }
}

function Get-HfErrorClass {
    param(
        [int]$StatusCode = 0,
        [string]$Message = '',
        [string]$HttpStatusLabel = ''
    )

    if (-not [string]::IsNullOrWhiteSpace($HttpStatusLabel)) {
        switch ($HttpStatusLabel) {
            'AUTHENTICATION_FAILED' { return 'AUTHENTICATION_FAILED' }
            'PERMISSION_DENIED' { return 'PERMISSION_DENIED' }
            'ENDPOINT_NOT_FOUND' { return 'ENDPOINT_NOT_FOUND' }
            'RATE_LIMITED' { return 'RATE_LIMITED' }
            'PROVIDER_ERROR' { return 'PROVIDER_ERROR' }
        }
    }

    if ($StatusCode -eq 401) { return 'AUTHENTICATION_FAILED' }
    if ($StatusCode -eq 403) { return 'PERMISSION_DENIED' }
    if ($StatusCode -eq 404) { return 'ENDPOINT_NOT_FOUND' }
    if ($StatusCode -eq 429) { return 'RATE_LIMITED' }
    if ($StatusCode -ge 500) { return 'PROVIDER_ERROR' }
    if ($Message -match '(?i)timeout|timed out|operation has timed out') {
        return 'REQUEST_TIMEOUT'
    }
    if ($Message -match '(?i)loading|model is currently loading|cold start') {
        return 'MODEL_LOADING'
    }
    if ($Message -match '(?i)model_not_found|"type"\s*:\s*"invalid_request_error".*model|model[^"\n]{0,80}(not found|not supported|does not exist|unavailable|unknown model|not served|not available)') {
        return 'MODEL_NOT_SERVED'
    }
    if ($StatusCode -eq 404 -and $Message -match '(?i)model') {
        return 'MODEL_NOT_SERVED'
    }
    if ($StatusCode -eq 400 -and $Message -match '(?i)model') {
        return 'MODEL_NOT_AVAILABLE'
    }
    return 'REQUEST_FAILED'
}

function Invoke-HfWebRequestCompat {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Uri,

        [Parameter(Mandatory = $true)]
        [ValidateSet('GET', 'POST')]
        [string]$Method,

        [Parameter(Mandatory = $true)]
        [hashtable]$Headers,

        [string]$Body = $null,
        [int]$TimeoutSec = 120
    )

    Enable-HfTls12Support

    $requestParams = @{
        Uri             = $Uri
        Method          = $Method
        Headers         = $Headers
        UseBasicParsing = $true
    }
    if ($Method -eq 'POST') {
        $requestParams['Body'] = $Body
    }

    $invokeCommand = Get-Command Invoke-WebRequest -ErrorAction Stop
    if ($invokeCommand.Parameters.ContainsKey('TimeoutSec')) {
        $requestParams['TimeoutSec'] = $TimeoutSec
    }

    return Invoke-WebRequest @requestParams
}

function New-HfAuthorizationHeaders {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Token,

        [switch]$IncludeContentType
    )

    $headers = @{
        Authorization = ('Bearer {0}' -f $Token)
        Accept        = 'application/json'
    }

    if ($IncludeContentType) {
        $contentType = 'application/json'
        $headers['Content-Type'] = $contentType
    }

    return $headers
}

function Invoke-HfOpenAIRequest {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateSet('GET', 'POST')]
        [string]$Method,

        [Parameter(Mandatory = $true)]
        [string]$Uri,

        [Parameter(Mandatory = $true)]
        [string]$Token,

        [string]$Body = $null,
        [int]$TimeoutSec = 120
    )

    $headers = New-HfAuthorizationHeaders -Token $Token -IncludeContentType:($Method -eq 'POST')

    try {
        $response = Invoke-HfWebRequestCompat -Uri $Uri -Method $Method -Headers $headers -Body $Body -TimeoutSec $TimeoutSec

        return [pscustomobject]@{
            Ok               = $true
            StatusCode       = [int]$response.StatusCode
            Content          = [string]$response.Content
            ErrorClass       = $null
            HttpStatusLabel  = $null
            ErrorCategory    = $null
            ExceptionType    = $null
            InnerExceptionType = $null
            WebExceptionStatus = $null
            SafeErrorMessage = $null
            Message          = $null
        }
    }
    catch {
        $statusCode = 0
        $content = ''
        $exception = $_.Exception
        $innerException = $exception.InnerException
        $message = [string]$exception.Message
        $webExceptionStatus = $null

        if ($exception -is [System.Net.WebException]) {
            $webExceptionStatus = $exception.Status
            $httpResponse = $exception.Response
            if ($null -ne $httpResponse) {
                try {
                    $statusCode = [int]$httpResponse.StatusCode
                }
                catch {
                    $statusCode = 0
                }

                try {
                    $stream = $httpResponse.GetResponseStream()
                    if ($null -ne $stream) {
                        $reader = New-Object System.IO.StreamReader($stream)
                        $content = [string]$reader.ReadToEnd()
                        $reader.Close()
                        $stream.Close()
                    }
                }
                catch {
                    $content = ''
                }
            }
        }
        elseif ($exception.Response) {
            try {
                $statusCode = [int]$exception.Response.StatusCode
            }
            catch {
                $statusCode = 0
            }
        }

        $safeMessage = Get-SafeErrorMessage -RawMessage ($message + ' ' + $content) -Token $Token
        $httpStatusLabel = Get-HfHttpStatusLabel -StatusCode $statusCode
        $errorCategory = $null
        if ($statusCode -le 0) {
            $errorCategory = Get-HfTransportErrorCategory -Exception $exception -SafeMessage $safeMessage
        }

        $errorClass = Get-HfErrorClass -StatusCode $statusCode -Message ($message + ' ' + $content) -HttpStatusLabel $httpStatusLabel
        if ($statusCode -le 0 -and -not [string]::IsNullOrWhiteSpace($errorCategory)) {
            $errorClass = $errorCategory
        }

        return [pscustomobject]@{
            Ok                 = $false
            StatusCode         = $statusCode
            Content            = $content
            ErrorClass         = $errorClass
            HttpStatusLabel    = $httpStatusLabel
            ErrorCategory      = $errorCategory
            ExceptionType      = $exception.GetType().FullName
            InnerExceptionType = $(if ($null -ne $innerException) { $innerException.GetType().FullName } else { $null })
            WebExceptionStatus = $(if ($null -ne $webExceptionStatus) { [string]$webExceptionStatus } else { $null })
            SafeErrorMessage   = $safeMessage
            Message            = $message
        }
    }
}

function Get-ProviderFromModelId {
    param([string]$ModelId)

    if ([string]::IsNullOrWhiteSpace($ModelId)) {
        return $null
    }
    $parts = $ModelId.Split(':')
    if ($parts.Count -ge 2) {
        return $parts[$parts.Count - 1]
    }
    return $null
}

function ConvertTo-HfObjectArray {
    param(
        [AllowNull()]
        $Value
    )

    if ($null -eq $Value) {
        return ,@()
    }

    if ($Value -is [string]) {
        return ,@($Value)
    }

    if ($Value -is [System.Collections.Generic.List[object]]) {
        return ,@($Value.ToArray())
    }

    if ($Value -is [System.Array]) {
        return ,@($Value)
    }

    if ($Value -is [System.Management.Automation.PSCustomObject]) {
        return ,@($Value)
    }

    if ($Value -is [System.Collections.IEnumerable]) {
        return ,@($Value)
    }

    return ,@($Value)
}

function Get-HfModelsFromDiscoveryPayload {
    param(
        [AllowNull()]
        $Parsed
    )

    if ($null -eq $Parsed) {
        return @()
    }

    if ($Parsed.PSObject.Properties.Name -contains 'data') {
        $items = ConvertTo-HfObjectArray -Value $Parsed.data
        return ,@($items)
    }

    if ($Parsed.PSObject.Properties.Name -contains 'models') {
        $items = ConvertTo-HfObjectArray -Value $Parsed.models
        return ,@($items)
    }

    if ($Parsed -is [System.Array]) {
        return ,@($Parsed)
    }

    return ,@(ConvertTo-HfObjectArray -Value $Parsed)
}

function Get-HfModelIdFromEntry {
    param(
        [AllowNull()]
        $ModelEntry
    )

    if ($null -eq $ModelEntry) {
        return ''
    }

    if ($ModelEntry -is [string]) {
        return $ModelEntry.Trim()
    }

    foreach ($prop in @('id', 'model', 'name', 'modelId')) {
        if ($ModelEntry.PSObject.Properties.Name -contains $prop) {
            $value = [string]$ModelEntry.$prop
            if (-not [string]::IsNullOrWhiteSpace($value)) {
                return $value.Trim()
            }
        }
    }

    return ''
}

function ConvertTo-HfCandidateArray {
    param(
        [AllowNull()]
        $Candidates
    )

    if ($null -eq $Candidates) {
        return ,@()
    }

    if ($Candidates -is [System.Collections.Generic.List[object]]) {
        return ,@($Candidates.ToArray())
    }

    if ($Candidates -is [System.Array]) {
        return ,@($Candidates)
    }

    return ,@($Candidates)
}

function Test-ModelSupportsChat {
    param($ModelEntry)

    if ($null -eq $ModelEntry) {
        return $false
    }

    $modelId = Get-HfModelIdFromEntry -ModelEntry $ModelEntry
    if ([string]::IsNullOrWhiteSpace($modelId)) {
        return $false
    }

    $lowerId = $modelId.ToLowerInvariant()
    $blockedHints = @('embed', 'embedding', 'rerank', 'whisper', 'speech', 'tts', 'vision-encode')
    foreach ($hint in $blockedHints) {
        if ($lowerId.Contains($hint)) {
            return $false
        }
    }

    if ($ModelEntry -is [string]) {
        return $true
    }

    if ($ModelEntry.PSObject.Properties.Name -contains 'capabilities') {
        $caps = $ModelEntry.capabilities
        if ($null -ne $caps) {
            if ($caps.PSObject.Properties.Name -contains 'chat' -and $caps.chat -eq $false) {
                return $false
            }
            if ($caps.PSObject.Properties.Name -contains 'completion' -and $caps.completion -eq $false -and
                $caps.PSObject.Properties.Name -contains 'chat' -and $caps.chat -eq $false) {
                return $false
            }
        }
    }

    return $true
}

function ConvertTo-HfModelCandidate {
    param($ModelEntry)

    $modelId = Get-HfModelIdFromEntry -ModelEntry $ModelEntry
    if ([string]::IsNullOrWhiteSpace($modelId)) {
        return $null
    }

    $provider = Get-ProviderFromModelId -ModelId $modelId
    if ($ModelEntry -is [string]) {
        return [pscustomobject]@{
            modelId         = $modelId
            provider        = $provider
            chatCompatible  = $true
            contextLength   = $null
            pricing         = $null
        }
    }

    if ($ModelEntry.PSObject.Properties.Name -contains 'owned_by' -and -not [string]::IsNullOrWhiteSpace([string]$ModelEntry.owned_by)) {
        if ([string]::IsNullOrWhiteSpace($provider)) {
            $provider = [string]$ModelEntry.owned_by
        }
    }

    $contextLength = $null
    foreach ($prop in @('context_length', 'contextLength', 'max_model_len')) {
        if ($ModelEntry.PSObject.Properties.Name -contains $prop) {
            $contextLength = $ModelEntry.$prop
            break
        }
    }

    $pricing = $null
    if ($ModelEntry.PSObject.Properties.Name -contains 'pricing') {
        $pricing = $ModelEntry.pricing
    }

    return [pscustomobject]@{
        modelId         = $modelId
        provider        = $provider
        chatCompatible  = $true
        contextLength   = $contextLength
        pricing         = $pricing
    }
}

function Get-HfChatModelCandidates {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Token
    )

    $response = Invoke-HfOpenAIRequest -Method GET -Uri $Script:HfModelsEndpoint -Token $Token -TimeoutSec 90
    if (-not $response.Ok) {
        Write-HfRequestFailureDiagnostics -Response $response
        if ($response.StatusCode -gt 0) {
            throw ('Models discovery failed with HTTP {0} ({1}).' -f $response.StatusCode, $response.HttpStatusLabel)
        }
        throw ('Models discovery failed before HTTP response ({0}).' -f $response.ErrorClass)
    }

    $parsed = $response.Content | ConvertFrom-Json
    $items = Get-HfModelsFromDiscoveryPayload -Parsed $parsed

    $candidates = New-Object System.Collections.Generic.List[object]
    foreach ($entry in $items) {
        if (-not (Test-ModelSupportsChat -ModelEntry $entry)) {
            continue
        }
        $candidate = ConvertTo-HfModelCandidate -ModelEntry $entry
        if ($null -eq $candidate) {
            continue
        }
        if (-not [string]::IsNullOrWhiteSpace($candidate.modelId)) {
            [void]$candidates.Add($candidate)
        }
        if ($candidates.Count -ge $Script:MaxDiscoverCandidates) {
            break
        }
    }

    $normalized = ConvertTo-HfCandidateArray -Candidates $candidates
    return ,@($normalized)
}

function Write-DiscoverReport {
    param(
        [Parameter(Mandatory = $true)]
        $Candidates
    )

    $Candidates = ConvertTo-HfCandidateArray -Candidates $Candidates

    Write-Host ''
    Write-Host 'HF_MODEL_CANDIDATES (max 20, chat-compatible, synthetic metadata only)'
    Write-Host '----------------------------------------------------------------'
    $index = 0
    foreach ($candidate in $Candidates) {
        $index++
        Write-Host ('[{0}] modelId={1}' -f $index, $candidate.modelId)
        if (-not [string]::IsNullOrWhiteSpace([string]$candidate.provider)) {
            Write-Host ('    provider={0}' -f $candidate.provider)
        }
        Write-Host '    chatCompatible=true'
        if ($null -ne $candidate.contextLength) {
            Write-Host ('    contextLength={0}' -f $candidate.contextLength)
        }
        if ($null -ne $candidate.pricing) {
            Write-Host ('    pricing={0}' -f ($candidate.pricing | ConvertTo-Json -Compress -Depth 4))
        }
    }

    $repoRoot = Get-RepoRoot
    $artifactsDir = Join-Path $repoRoot 'artifacts\phase6'
    if (-not (Test-Path $artifactsDir)) {
        New-Item -ItemType Directory -Path $artifactsDir -Force | Out-Null
    }
    $reportPath = Join-Path $artifactsDir 'hf-model-candidates.json'
    $payload = @{
        generatedAtUtc = (Get-Date).ToUniversalTime().ToString('o')
        source         = $Script:HfModelsEndpoint
        candidateCount = $Candidates.Count
        candidates     = $Candidates
    }
    ($payload | ConvertTo-Json -Depth 6) | Set-Content -Path $reportPath -Encoding UTF8
    Write-Host ''
    Write-Host ('DISCOVER_REPORT_JSON={0}' -f $reportPath)
    Write-Host 'TOKEN_EXPOSED=NO'
}

function Assert-VerifyParameters {
    param(
        [string]$Primary,
        [string]$Secondary
    )

    if ([string]::IsNullOrWhiteSpace($Primary)) {
        throw 'PrimaryModel is required for Verify mode.'
    }
    if ([string]::IsNullOrWhiteSpace($Secondary)) {
        throw 'SecondaryModel is required for Verify mode.'
    }
    if ($Primary.Trim() -eq $Secondary.Trim()) {
        throw 'PrimaryModel and SecondaryModel must be different.'
    }
}

function Invoke-HfChatSmokeTest {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Token,

        [Parameter(Mandatory = $true)]
        [string]$ModelId
    )

    $prompt = 'Synthetic Miyuna product safety check. Reply with the single word OK.'
    $requestBody = @{
        model    = $ModelId
        messages = @(
            @{ role = 'user'; content = $prompt }
        )
        max_tokens = $Script:SmokeMaxTokens
        stream     = $false
    } | ConvertTo-Json -Depth 5 -Compress

    $started = Get-Date
    $response = Invoke-HfOpenAIRequest -Method POST -Uri $Script:HfChatEndpoint -Token $Token -Body $requestBody -TimeoutSec 120
    $latencyMs = [int](((Get-Date) - $started).TotalMilliseconds)

    $responseReceived = $false
    $promptTokens = $null
    $completionTokens = $null
    $totalTokens = $null

    if ($response.Ok) {
        try {
            $parsed = $response.Content | ConvertFrom-Json
            if ($null -ne $parsed.choices -and $parsed.choices.Count -gt 0) {
                $content = [string]$parsed.choices[0].message.content
                $responseReceived = -not [string]::IsNullOrWhiteSpace($content)
            }
            if ($null -ne $parsed.usage) {
                $promptTokens = $parsed.usage.prompt_tokens
                $completionTokens = $parsed.usage.completion_tokens
                if ($parsed.usage.PSObject.Properties.Name -contains 'total_tokens') {
                    $totalTokens = $parsed.usage.total_tokens
                }
            }
        }
        catch {
            $responseReceived = $false
        }
    }

    $health = 'FAIL'
    if ($response.Ok -and $responseReceived) {
        $health = 'PASS'
    }

    return [pscustomobject]@{
        ModelId            = $ModelId
        Health             = $health
        HttpStatus         = $response.StatusCode
        LatencyMs          = $latencyMs
        ResponseReceived   = $responseReceived
        ErrorClass         = $response.ErrorClass
        PromptTokens       = $promptTokens
        CompletionTokens   = $completionTokens
        TotalTokens        = $totalTokens
        TokenUsageAvailable = ($null -ne $promptTokens -or $null -ne $completionTokens -or $null -ne $totalTokens)
    }
}

function Write-VerifyReport {
    param(
        [Parameter(Mandatory = $true)]
        $PrimaryResult,

        [Parameter(Mandatory = $true)]
        $SecondaryResult
    )

    $dualPass = ($PrimaryResult.Health -eq 'PASS' -and $SecondaryResult.Health -eq 'PASS')

    Write-Host ''
    Write-Host ('PRIMARY_MODEL={0}' -f $PrimaryResult.ModelId)
    Write-Host ('PRIMARY_HEALTH={0}' -f $PrimaryResult.Health)
    Write-Host ('PRIMARY_HTTP_STATUS={0}' -f $PrimaryResult.HttpStatus)
    Write-Host ('PRIMARY_LATENCY_MS={0}' -f $PrimaryResult.LatencyMs)
    Write-Host ('PRIMARY_RESPONSE_RECEIVED={0}' -f ($(if ($PrimaryResult.ResponseReceived) { 'YES' } else { 'NO' })))
    if ($PrimaryResult.TokenUsageAvailable) {
        Write-Host ('PRIMARY_PROMPT_TOKENS={0}' -f $PrimaryResult.PromptTokens)
        Write-Host ('PRIMARY_COMPLETION_TOKENS={0}' -f $PrimaryResult.CompletionTokens)
        if ($null -ne $PrimaryResult.TotalTokens) {
            Write-Host ('PRIMARY_TOTAL_TOKENS={0}' -f $PrimaryResult.TotalTokens)
        }
    }
    if ($PrimaryResult.Health -ne 'PASS' -and -not [string]::IsNullOrWhiteSpace([string]$PrimaryResult.ErrorClass)) {
        Write-Host ('PRIMARY_ERROR_CLASS={0}' -f $PrimaryResult.ErrorClass)
    }

    Write-Host ('SECONDARY_MODEL={0}' -f $SecondaryResult.ModelId)
    Write-Host ('SECONDARY_HEALTH={0}' -f $SecondaryResult.Health)
    Write-Host ('SECONDARY_HTTP_STATUS={0}' -f $SecondaryResult.HttpStatus)
    Write-Host ('SECONDARY_LATENCY_MS={0}' -f $SecondaryResult.LatencyMs)
    Write-Host ('SECONDARY_RESPONSE_RECEIVED={0}' -f ($(if ($SecondaryResult.ResponseReceived) { 'YES' } else { 'NO' })))
    if ($SecondaryResult.TokenUsageAvailable) {
        Write-Host ('SECONDARY_PROMPT_TOKENS={0}' -f $SecondaryResult.PromptTokens)
        Write-Host ('SECONDARY_COMPLETION_TOKENS={0}' -f $SecondaryResult.CompletionTokens)
        if ($null -ne $SecondaryResult.TotalTokens) {
            Write-Host ('SECONDARY_TOTAL_TOKENS={0}' -f $SecondaryResult.TotalTokens)
        }
    }
    if ($SecondaryResult.Health -ne 'PASS' -and -not [string]::IsNullOrWhiteSpace([string]$SecondaryResult.ErrorClass)) {
        Write-Host ('SECONDARY_ERROR_CLASS={0}' -f $SecondaryResult.ErrorClass)
    }

    Write-Host ('REAL_DUAL_LLM_DIRECT_SMOKE={0}' -f ($(if ($dualPass) { 'YES' } else { 'NO' })))
    Write-Host 'TOKEN_EXPOSED=NO'
}

function Get-HfDistinctModelIds {
    param(
        [Parameter(Mandatory = $true)]
        $Candidates
    )

    $Candidates = ConvertTo-HfCandidateArray -Candidates $Candidates

    $ids = New-Object System.Collections.Generic.List[string]
    $seen = @{}

    foreach ($candidate in $Candidates) {
        $modelId = [string]$candidate.modelId
        if ([string]::IsNullOrWhiteSpace($modelId)) {
            continue
        }
        if ($seen.ContainsKey($modelId)) {
            continue
        }
        $seen[$modelId] = $true
        [void]$ids.Add($modelId)
    }

    return ,@($ids.ToArray())
}

function Invoke-HfDualModelVerificationWithFallback {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Token,

        [Parameter(Mandatory = $true)]
        $Candidates,

        [int]$MaxPairAttempts = 6
    )

    $Candidates = ConvertTo-HfCandidateArray -Candidates $Candidates
    $modelIds = Get-HfDistinctModelIds -Candidates $Candidates
    if ($modelIds.Count -lt 2) {
        throw 'DISCOVER_INSUFFICIENT_MODELS'
    }

    $attempts = 0
    for ($i = 0; $i -lt ($modelIds.Count - 1); $i++) {
        for ($j = $i + 1; $j -lt $modelIds.Count; $j++) {
            if ($attempts -ge $MaxPairAttempts) {
                break
            }

            $attempts++
            Write-Host ('VERIFY_PAIR_ATTEMPT={0} PRIMARY_CANDIDATE={1} SECONDARY_CANDIDATE={2}' -f $attempts, $modelIds[$i], $modelIds[$j])

            $primaryResult = Invoke-HfChatSmokeTest -Token $Token -ModelId $modelIds[$i]
            $secondaryResult = Invoke-HfChatSmokeTest -Token $Token -ModelId $modelIds[$j]

            if ($primaryResult.Health -eq 'PASS' -and $secondaryResult.Health -eq 'PASS') {
                return [pscustomobject]@{
                    PrimaryResult = $primaryResult
                    SecondaryResult = $secondaryResult
                    PairAttempts    = $attempts
                }
            }

            if ($primaryResult.Health -ne 'PASS' -and -not [string]::IsNullOrWhiteSpace([string]$primaryResult.ErrorClass)) {
                Write-Host ('PRIMARY_SMOKE_FAILED errorClass={0}' -f $primaryResult.ErrorClass)
            }
            if ($secondaryResult.Health -ne 'PASS' -and -not [string]::IsNullOrWhiteSpace([string]$secondaryResult.ErrorClass)) {
                Write-Host ('SECONDARY_SMOKE_FAILED errorClass={0}' -f $secondaryResult.ErrorClass)
            }
        }

        if ($attempts -ge $MaxPairAttempts) {
            break
        }
    }

    throw 'VERIFY_NO_WORKING_MODEL_PAIR'
}

function Set-MiyunaLlmRuntimeEnvironment {
    param(
        [Parameter(Mandatory = $true)]
        [string]$PrimaryModel,

        [Parameter(Mandatory = $true)]
        [string]$SecondaryModel,

        [Parameter(Mandatory = $true)]
        [string]$HfToken
    )

    $hfBaseUrl = 'https://router.huggingface.co/v1'

    $env:LLM_USE_MOCK = 'false'
    $env:LLM_PRIMARY_BASE_URL = $hfBaseUrl
    $env:LLM_PRIMARY_API_KEY = $HfToken
    $env:LLM_PRIMARY_MODEL_NAME = $PrimaryModel
    $env:LLM_SECONDARY_BASE_URL = $hfBaseUrl
    $env:LLM_SECONDARY_API_KEY = $HfToken
    $env:LLM_SECONDARY_MODEL_NAME = $SecondaryModel
}

function Write-MiyunaRuntimeEnvArtifact {
    param(
        [Parameter(Mandatory = $true)]
        [string]$PrimaryModel,

        [Parameter(Mandatory = $true)]
        [string]$SecondaryModel
    )

    $repoRoot = Get-RepoRoot
    $artifactsDir = Join-Path $repoRoot 'artifacts\phase6'
    if (-not (Test-Path $artifactsDir)) {
        New-Item -ItemType Directory -Path $artifactsDir -Force | Out-Null
    }

    $artifactPath = Join-Path $artifactsDir 'miyuna-llm-runtime.public.ps1'
    $lines = @(
        '# Non-secret Miyuna LLM runtime variables for the current PowerShell session.'
        '# Secrets (HF_TOKEN / LLM_*_API_KEY) are set in-memory by run_phase6_hf_verification.ps1.'
        '# Run API/agent in the SAME shell after Auto verification succeeds.'
        '$env:LLM_USE_MOCK = ''false'''
        ('$env:LLM_PRIMARY_BASE_URL = ''https://router.huggingface.co/v1''')
        ('$env:LLM_PRIMARY_MODEL_NAME = ''{0}''' -f $PrimaryModel.Replace("'", "''"))
        ('$env:LLM_SECONDARY_BASE_URL = ''https://router.huggingface.co/v1''')
        ('$env:LLM_SECONDARY_MODEL_NAME = ''{0}''' -f $SecondaryModel.Replace("'", "''"))
        'if (-not [string]::IsNullOrWhiteSpace($env:HF_TOKEN)) {'
        '    $env:LLM_PRIMARY_API_KEY = $env:HF_TOKEN'
        '    $env:LLM_SECONDARY_API_KEY = $env:HF_TOKEN'
        '}'
    )
    $lines | Set-Content -Path $artifactPath -Encoding UTF8
    return $artifactPath
}

function Write-AutoVerificationArtifacts {
    param(
        [Parameter(Mandatory = $true)]
        $Candidates,

        [Parameter(Mandatory = $true)]
        $VerifyOutcome,

        [string]$PublicEnvArtifactPath
    )

    $Candidates = ConvertTo-HfCandidateArray -Candidates $Candidates

    $repoRoot = Get-RepoRoot
    $artifactsDir = Join-Path $repoRoot 'artifacts\phase6'
    if (-not (Test-Path $artifactsDir)) {
        New-Item -ItemType Directory -Path $artifactsDir -Force | Out-Null
    }

    $summaryPath = Join-Path $artifactsDir 'verification-summary.json'
    $payload = @{
        generatedAtUtc          = (Get-Date).ToUniversalTime().ToString('o')
        mode                    = 'Auto'
        discoverCandidateCount  = $Candidates.Count
        primaryModel            = $VerifyOutcome.PrimaryResult.ModelId
        secondaryModel          = $VerifyOutcome.SecondaryResult.ModelId
        verifyPairAttempts      = $VerifyOutcome.PairAttempts
        realDualLlmDirectSmoke  = 'YES'
        miyunaEnvConfigured     = 'YES'
        realRoutingVerified     = 'PENDING_API_RESTART'
        realFallbackVerified    = 'PENDING_API_RESTART'
        publicEnvArtifact       = $PublicEnvArtifactPath
    }
    ($payload | ConvertTo-Json -Depth 5) | Set-Content -Path $summaryPath -Encoding UTF8
    return $summaryPath
}

function Write-AutoVerificationReport {
    param(
        [Parameter(Mandatory = $true)]
        $VerifyOutcome,

        [string]$SummaryPath,
        [string]$PublicEnvArtifactPath
    )

    Write-VerifyReport -PrimaryResult $VerifyOutcome.PrimaryResult -SecondaryResult $VerifyOutcome.SecondaryResult

    Write-Host ''
    Write-Host '=== Phase 6 Auto Verification ==='
    Write-Host ('REAL_DUAL_LLM_RUNTIME_VERIFIED={0}' -f 'YES')
    Write-Host ('REAL_DUAL_LLM_DIRECT_SMOKE={0}' -f 'YES')
    Write-Host ('MIYUNA_ENV_CONFIGURED={0}' -f 'YES')
    Write-Host ('REAL_ROUTING_VERIFIED={0}' -f 'PENDING_API_RESTART')
    Write-Host ('REAL_FALLBACK_VERIFIED={0}' -f 'PENDING_API_RESTART')
    Write-Host ('MOCK_ONLY_IMPLEMENTATION={0}' -f 'NO')
    Write-Host ('VERIFY_PAIR_ATTEMPTS={0}' -f $VerifyOutcome.PairAttempts)
    Write-Host ('VERIFICATION_SUMMARY_JSON={0}' -f $SummaryPath)
    Write-Host ('MIYUNA_RUNTIME_PUBLIC_PS1={0}' -f $PublicEnvArtifactPath)
    Write-Host ''
    Write-Host 'Sonraki adim: API ve agent''i AYNI PowerShell oturumunda baslatin.'
    Write-Host 'Bu oturumda HF_TOKEN ve LLM_* env degiskenleri zaten ayarli.'
    Write-Host 'TOKEN_EXPOSED=NO'
}

function Get-PresetHfModelPair {
    $primary = [string]$env:LLM_PRIMARY_MODEL_NAME
    $secondary = [string]$env:LLM_SECONDARY_MODEL_NAME

    if ([string]::IsNullOrWhiteSpace($primary) -or [string]::IsNullOrWhiteSpace($secondary)) {
        return $null
    }

    $primary = $primary.Trim()
    $secondary = $secondary.Trim()

    if ($primary -eq $secondary) {
        throw 'PRESET_MODELS_MUST_DIFFER'
    }

    return [pscustomobject]@{
        Primary   = $primary
        Secondary = $secondary
    }
}

function Invoke-AutoVerification {
    param(
        [switch]$ForceNewToken
    )

    Write-Host ''
    Write-Host 'Phase 6 Hugging Face Auto Verification'
    Write-Host 'Discover -> Verify -> Miyuna runtime env (tek token girisi)'
    Write-Host ''

    $token = Initialize-HfSessionToken -ForcePrompt:$ForceNewToken

    $candidates = @()
    $verifyOutcome = $null
    $preset = Get-PresetHfModelPair

    if ($null -ne $preset) {
        Write-Host ''
        Write-Host 'STEP=VerifyPreset'
        Write-Host ('ENV_PRIMARY_MODEL_PRESERVED={0}' -f $preset.Primary)
        Write-Host ('ENV_SECONDARY_MODEL_PRESERVED={0}' -f $preset.Secondary)

        $primaryResult = Invoke-HfChatSmokeTest -Token $token -ModelId $preset.Primary
        $secondaryResult = Invoke-HfChatSmokeTest -Token $token -ModelId $preset.Secondary

        if ($primaryResult.Health -eq 'PASS' -and $secondaryResult.Health -eq 'PASS') {
            $verifyOutcome = [pscustomobject]@{
                PrimaryResult = $primaryResult
                SecondaryResult = $secondaryResult
                PairAttempts    = 1
            }
        }
        else {
            Write-Host 'PRESET_MODELS_SMOKE_FAILED — falling back to discovery'
            if ($primaryResult.Health -ne 'PASS' -and -not [string]::IsNullOrWhiteSpace([string]$primaryResult.ErrorClass)) {
                Write-Host ('PRIMARY_SMOKE_FAILED errorClass={0}' -f $primaryResult.ErrorClass)
                if ($primaryResult.ErrorClass -in @('MODEL_NOT_SERVED', 'MODEL_NOT_AVAILABLE', 'ENDPOINT_NOT_FOUND')) {
                    Write-Host ('PRESET_PRIMARY_STATUS={0}' -f $primaryResult.ErrorClass)
                }
            }
            if ($secondaryResult.Health -ne 'PASS' -and -not [string]::IsNullOrWhiteSpace([string]$secondaryResult.ErrorClass)) {
                Write-Host ('SECONDARY_SMOKE_FAILED errorClass={0}' -f $secondaryResult.ErrorClass)
                if ($secondaryResult.ErrorClass -in @('MODEL_NOT_SERVED', 'MODEL_NOT_AVAILABLE', 'ENDPOINT_NOT_FOUND')) {
                    Write-Host ('PRESET_SECONDARY_STATUS={0}' -f $secondaryResult.ErrorClass)
                }
            }
        }
    }

    if ($null -eq $verifyOutcome) {
        Write-Host ''
        Write-Host 'STEP=Discover'
        try {
            $candidates = Get-HfChatModelCandidates -Token $token
        }
        catch {
            $safe = Get-SafeErrorMessage -RawMessage $_.Exception.Message -Token $token
            if ($null -ne $_.InvocationInfo -and -not [string]::IsNullOrWhiteSpace([string]$_.InvocationInfo.PositionMessage)) {
                Write-Host ('DISCOVERY_ERROR_AT={0}' -f (Get-SafeErrorMessage -RawMessage $_.InvocationInfo.PositionMessage -Token $token))
            }
            throw $safe
        }
        if ($candidates.Count -eq 0) {
            throw 'No chat-compatible models were returned by the Hugging Face models endpoint.'
        }
        Write-DiscoverReport -Candidates $candidates

        Write-Host ''
        Write-Host 'STEP=Verify'
        $verifyOutcome = Invoke-HfDualModelVerificationWithFallback -Token $token -Candidates $candidates
    }

    Write-Host ''
    Write-Host 'STEP=ConfigureMiyunaRuntime'
    Set-MiyunaLlmRuntimeEnvironment -PrimaryModel $verifyOutcome.PrimaryResult.ModelId -SecondaryModel $verifyOutcome.SecondaryResult.ModelId -HfToken $token
    $publicEnvArtifact = Write-MiyunaRuntimeEnvArtifact -PrimaryModel $verifyOutcome.PrimaryResult.ModelId -SecondaryModel $verifyOutcome.SecondaryResult.ModelId
    $summaryPath = Write-AutoVerificationArtifacts -Candidates $candidates -VerifyOutcome $verifyOutcome -PublicEnvArtifactPath $publicEnvArtifact
    Write-AutoVerificationReport -VerifyOutcome $verifyOutcome -SummaryPath $summaryPath -PublicEnvArtifactPath $publicEnvArtifact

    return [pscustomobject]@{
        Token           = $token
        Candidates      = $candidates
        VerifyOutcome   = $verifyOutcome
        SummaryPath     = $summaryPath
        PublicEnvScript = $publicEnvArtifact
    }
}

function Invoke-Phase6HfRuntime {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateSet('Discover', 'Verify', 'Auto')]
        [string]$Mode,

        [string]$PrimaryModel,
        [string]$SecondaryModel,

        [switch]$ForceNewToken
    )

    $token = $null
    $autoResult = $null

    try {
        Enable-HfTls12Support

        switch ($Mode) {
            'Auto' {
                $autoResult = Invoke-AutoVerification -ForceNewToken:$ForceNewToken
                $token = $autoResult.Token
            }
            default {
                $token = Get-HfAccessToken
            }
        }

        switch ($Mode) {
            'Discover' {
                $candidates = Get-HfChatModelCandidates -Token $token
                if ($candidates.Count -eq 0) {
                    throw 'No chat-compatible models were returned by the Hugging Face models endpoint.'
                }
                Write-DiscoverReport -Candidates $candidates
            }
            'Verify' {
                Assert-VerifyParameters -Primary $PrimaryModel -Secondary $SecondaryModel
                $primaryResult = Invoke-HfChatSmokeTest -Token $token -ModelId $PrimaryModel.Trim()
                $secondaryResult = Invoke-HfChatSmokeTest -Token $token -ModelId $SecondaryModel.Trim()
                Write-VerifyReport -PrimaryResult $primaryResult -SecondaryResult $secondaryResult
                if ($primaryResult.Health -ne 'PASS' -or $secondaryResult.Health -ne 'PASS') {
                    throw 'One or both model smoke tests failed.'
                }
            }
        }

        return $autoResult
    }
    finally {
        Clear-HfSecretState -TokenRef ([ref]$token)
    }
}

if ($MyInvocation.InvocationName -ne '.') {
    Invoke-Phase6HfRuntime -Mode $Mode -PrimaryModel $PrimaryModel -SecondaryModel $SecondaryModel -ForceNewToken:$ForceNewToken
}
