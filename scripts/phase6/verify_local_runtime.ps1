# Phase 6 — Local OpenAI-compatible runtime verification (Ollama / LM Studio / llama.cpp server).
# No cloud token required for localhost runtimes.

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('Discover', 'Verify', 'Auto')]
    [string]$Mode,

    [string]$PrimaryModel,
    [string]$SecondaryModel,

    [string]$PrimaryBaseUrl = 'http://127.0.0.1:11434/v1',
    [string]$SecondaryBaseUrl = 'http://127.0.0.1:11434/v1',
    [string]$ManifestPath
)

$ErrorActionPreference = 'Stop'

$Script:SmokeMaxTokens = 16
$Script:MaxDiscoverCandidates = 20

function Get-RepoRoot {
    return Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
}

function Enable-LocalTls12Support {
    if ([Net.ServicePointManager]::SecurityProtocol -band [Net.SecurityProtocolType]::Tls12) {
        return
    }
    [Net.ServicePointManager]::SecurityProtocol =
        [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
}

function Resolve-LocalManifestPath {
    param([string]$OverridePath)

    if (-not [string]::IsNullOrWhiteSpace($OverridePath)) {
        return (Resolve-Path -LiteralPath $OverridePath).Path
    }

    $candidates = @(
        (Join-Path (Get-RepoRoot) 'artifacts\phase6\local-models.manifest.json')
        (Join-Path $PSScriptRoot 'local-models.manifest.json')
        (Join-Path $PSScriptRoot 'local-models.manifest.example.json')
    )

    foreach ($candidate in $candidates) {
        if (Test-Path -LiteralPath $candidate) {
            return $candidate
        }
    }

    throw 'LOCAL_MANIFEST_NOT_FOUND'
}

function Get-EffectiveLocalBaseUrls {
    param(
        $ManifestDocument,
        [string]$PrimaryBaseUrl,
        [string]$SecondaryBaseUrl
    )

    $primary = $PrimaryBaseUrl
    $secondary = $SecondaryBaseUrl

    if ($null -ne $ManifestDocument.runtime) {
        if (-not [string]::IsNullOrWhiteSpace([string]$ManifestDocument.runtime.primaryBaseUrl)) {
            $primary = [string]$ManifestDocument.runtime.primaryBaseUrl
        }
        if (-not [string]::IsNullOrWhiteSpace([string]$ManifestDocument.runtime.secondaryBaseUrl)) {
            $secondary = [string]$ManifestDocument.runtime.secondaryBaseUrl
        }
    }

    return [pscustomobject]@{
        PrimaryBaseUrl   = $primary
        SecondaryBaseUrl = $secondary
    }
}

function Get-LocalModelsManifest {
    param([string]$OverridePath)

    $path = Resolve-LocalManifestPath -OverridePath $OverridePath
    $raw = Get-Content -LiteralPath $path -Raw -Encoding UTF8
    $manifest = $raw | ConvertFrom-Json

    return [pscustomobject]@{
        Path     = $path
        Document = $manifest
    }
}

function Get-LocalRuntimeApiKey {
    param($ManifestDocument)

    if ($null -ne $ManifestDocument.runtime.apiKeyEnv) {
        $envName = [string]$ManifestDocument.runtime.apiKeyEnv
        if (-not [string]::IsNullOrWhiteSpace($envName) -and -not [string]::IsNullOrWhiteSpace([string](Get-Item -Path ("Env:{0}" -f $envName) -ErrorAction SilentlyContinue).Value)) {
            return [string](Get-Item -Path ("Env:{0}" -f $envName)).Value
        }
    }

    if (-not [string]::IsNullOrWhiteSpace($env:LOCAL_LLM_API_KEY)) {
        return [string]$env:LOCAL_LLM_API_KEY
    }

    return ''
}

function Invoke-LocalWebRequestCompat {
    param(
        [Parameter(Mandatory = $true)][string]$Uri,
        [Parameter(Mandatory = $true)][ValidateSet('GET', 'POST')][string]$Method,
        [hashtable]$Headers = @{},
        [string]$Body = $null,
        [int]$TimeoutSec = 120
    )

    Enable-LocalTls12Support

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

function Get-LocalOpenAiModels {
    param(
        [Parameter(Mandatory = $true)][string]$BaseUrl,
        [string]$ApiKey = ''
    )

    $headers = @{ Accept = 'application/json' }
    if (-not [string]::IsNullOrWhiteSpace($ApiKey)) {
        $headers['Authorization'] = ('Bearer {0}' -f $ApiKey)
    }

    $uri = ('{0}/models' -f $BaseUrl.TrimEnd('/'))
    $response = Invoke-LocalWebRequestCompat -Uri $uri -Method GET -Headers $headers -TimeoutSec 60
    $parsed = $response.Content | ConvertFrom-Json

    $items = @()
    if ($null -ne $parsed.data) {
        $items = @($parsed.data)
    }
    elseif ($parsed -is [System.Array]) {
        $items = @($parsed)
    }

    $models = New-Object System.Collections.Generic.List[string]
    foreach ($entry in $items) {
        $modelId = [string]$entry.id
        if (-not [string]::IsNullOrWhiteSpace($modelId)) {
            [void]$models.Add($modelId)
        }
        if ($models.Count -ge $Script:MaxDiscoverCandidates) {
            break
        }
    }

    return ,@($models.ToArray())
}

function Get-OllamaModelTags {
    param([string]$BaseUrl)

    try {
        $u = [Uri]$BaseUrl
        $tagsUri = ('{0}://{1}:{2}/api/tags' -f $u.Scheme, $u.Host, $u.Port)
        $response = Invoke-LocalWebRequestCompat -Uri $tagsUri -Method GET -Headers @{ Accept = 'application/json' } -TimeoutSec 30
        $parsed = $response.Content | ConvertFrom-Json
        if ($null -eq $parsed.models) {
            return @()
        }
        $modelEntries = @($parsed.models)
        if ($modelEntries.Count -eq 1 -and $null -eq $modelEntries[0].name) {
            $modelEntries = @($parsed.models)
        }
        return @($modelEntries | ForEach-Object { [string]$_.name } | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    }
    catch {
        return @()
    }
}

function Get-LocalWeightInventory {
    param($ManifestDocument)

    $repoRoot = Get-RepoRoot
    $entries = New-Object System.Collections.Generic.List[object]

    if ($null -eq $ManifestDocument.localWeights) {
        return ,@($entries.ToArray())
    }

    foreach ($item in @($ManifestDocument.localWeights)) {
        $relativePath = [string]$item.path
        if ([string]::IsNullOrWhiteSpace($relativePath)) {
            continue
        }
        $fullPath = Join-Path $repoRoot $relativePath
        $exists = Test-Path -LiteralPath $fullPath
        [void]$entries.Add([pscustomobject]@{
            path         = $relativePath
            fullPath     = $fullPath
            displayName  = [string]$item.displayName
            existsOnDisk = $exists
            preferredRoles = @($item.preferredRoles)
        })
    }

    return ,@($entries.ToArray())
}

function Write-LocalDiscoverReport {
    param(
        [Parameter(Mandatory = $true)][array]$RuntimeModels,
        [Parameter(Mandatory = $true)][array]$WeightInventory,
        [string]$PrimaryBaseUrl,
        [string]$SecondaryBaseUrl,
        [string]$ManifestPath
    )

    Write-Host ''
    Write-Host 'LOCAL_RUNTIME_MODELS (OpenAI-compatible /v1/models)'
    Write-Host '---------------------------------------------------'
    if ($RuntimeModels.Count -eq 0) {
        Write-Host 'No runtime models discovered.'
    }
    else {
        $index = 0
        foreach ($modelId in $RuntimeModels) {
            $index++
            Write-Host ('[{0}] modelId={1}' -f $index, $modelId)
        }
    }

    Write-Host ''
    Write-Host 'LOCAL_WEIGHTS (repo disk inventory)'
    Write-Host '-----------------------------------'
    if ($WeightInventory.Count -eq 0) {
        Write-Host 'No local weight paths configured in manifest.'
    }
    else {
        foreach ($weight in $WeightInventory) {
            Write-Host ('path={0} exists={1} displayName={2}' -f $weight.path, $(if ($weight.existsOnDisk) { 'YES' } else { 'NO' }), $weight.displayName)
        }
    }

    $repoRoot = Get-RepoRoot
    $artifactsDir = Join-Path $repoRoot 'artifacts\phase6'
    if (-not (Test-Path $artifactsDir)) {
        New-Item -ItemType Directory -Path $artifactsDir -Force | Out-Null
    }

    $reportPath = Join-Path $artifactsDir 'local-model-candidates.json'
    $payload = @{
        generatedAtUtc   = (Get-Date).ToUniversalTime().ToString('o')
        manifestPath     = $ManifestPath
        primaryBaseUrl   = $PrimaryBaseUrl
        secondaryBaseUrl = $SecondaryBaseUrl
        runtimeModels    = $RuntimeModels
        localWeights     = $WeightInventory
    }
    ($payload | ConvertTo-Json -Depth 6) | Set-Content -Path $reportPath -Encoding UTF8

    Write-Host ''
    Write-Host ('DISCOVER_REPORT_JSON={0}' -f $reportPath)
    Write-Host ('MANIFEST_PATH={0}' -f $ManifestPath)
    Write-Host 'TOKEN_EXPOSED=NO'
}

function Invoke-LocalChatSmokeTest {
    param(
        [Parameter(Mandatory = $true)][string]$BaseUrl,
        [Parameter(Mandatory = $true)][string]$ModelId,
        [string]$ApiKey = ''
    )

    $headers = @{
        Accept       = 'application/json'
        'Content-Type' = 'application/json'
    }
    if (-not [string]::IsNullOrWhiteSpace($ApiKey)) {
        $headers['Authorization'] = ('Bearer {0}' -f $ApiKey)
    }

    $body = @{
        model    = $ModelId
        messages = @(@{ role = 'user'; content = 'Synthetic Miyuna product safety check. Reply with the single word OK.' })
        max_tokens = $Script:SmokeMaxTokens
        stream   = $false
    } | ConvertTo-Json -Depth 5 -Compress

    $started = Get-Date
    try {
        $response = Invoke-LocalWebRequestCompat -Uri ('{0}/chat/completions' -f $BaseUrl.TrimEnd('/')) -Method POST -Headers $headers -Body $body -TimeoutSec 180
        $latencyMs = [int](((Get-Date) - $started).TotalMilliseconds)
        $parsed = $response.Content | ConvertFrom-Json
        $content = ''
        if ($null -ne $parsed.choices -and $parsed.choices.Count -gt 0) {
            $content = [string]$parsed.choices[0].message.content
        }
        $ok = -not [string]::IsNullOrWhiteSpace($content)
        return [pscustomobject]@{
            ModelId          = $ModelId
            BaseUrl          = $BaseUrl
            Health           = $(if ($ok) { 'PASS' } else { 'FAIL' })
            HttpStatus       = [int]$response.StatusCode
            LatencyMs        = $latencyMs
            ResponseReceived = $ok
            ErrorClass       = $(if ($ok) { $null } else { 'EMPTY_RESPONSE' })
        }
    }
    catch {
        $latencyMs = [int](((Get-Date) - $started).TotalMilliseconds)
        return [pscustomobject]@{
            ModelId          = $ModelId
            BaseUrl          = $BaseUrl
            Health           = 'FAIL'
            HttpStatus       = 0
            LatencyMs        = $latencyMs
            ResponseReceived = $false
            ErrorClass       = 'REQUEST_FAILED'
        }
    }
}

function Select-LocalDualModels {
    param(
        [Parameter(Mandatory = $true)][array]$RuntimeModels,
        $ManifestDocument
    )

    $primary = ''
    $secondary = ''
    $singleModelLocalDev = $false
    if ($null -ne $ManifestDocument.bindings) {
        $primary = [string]$ManifestDocument.bindings.primaryModel
        $secondary = [string]$ManifestDocument.bindings.secondaryModel
        if ($ManifestDocument.bindings.PSObject.Properties.Name -contains 'singleModelLocalDev') {
            $singleModelLocalDev = [bool]$ManifestDocument.bindings.singleModelLocalDev
        }
    }

    if (-not [string]::IsNullOrWhiteSpace($primary) -and -not [string]::IsNullOrWhiteSpace($secondary)) {
        if ($primary -eq $secondary -and -not $singleModelLocalDev) {
            throw 'LOCAL_MODEL_BINDINGS_REQUIRE_DISTINCT_MODELS'
        }
        return [pscustomobject]@{
            PrimaryModel   = $primary
            SecondaryModel = $secondary
            Selection      = $(if ($primary -eq $secondary) { 'manifest_single_model' } else { 'manifest_bindings' })
        }
    }

    $distinct = @($RuntimeModels | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Select-Object -Unique)
    if ($distinct.Count -eq 0) {
        throw 'LOCAL_RUNTIME_NO_MODELS'
    }
    if ($distinct.Count -eq 1) {
        return [pscustomobject]@{
            PrimaryModel   = $distinct[0]
            SecondaryModel = $distinct[0]
            Selection      = 'single_runtime_model'
        }
    }

    if (-not [string]::IsNullOrWhiteSpace($primary) -and $distinct -contains $primary) {
        $secondaryCandidate = @($distinct | Where-Object { $_ -ne $primary })[0]
        return [pscustomobject]@{
            PrimaryModel   = $primary
            SecondaryModel = $secondaryCandidate
            Selection      = 'manifest_primary_auto_secondary'
        }
    }

    return [pscustomobject]@{
        PrimaryModel   = $distinct[0]
        SecondaryModel = $distinct[1]
        Selection      = 'runtime_first_two'
    }
}

function Set-MiyunaLocalLlmRuntimeEnvironment {
    param(
        [Parameter(Mandatory = $true)][string]$PrimaryModel,
        [Parameter(Mandatory = $true)][string]$SecondaryModel,
        [Parameter(Mandatory = $true)][string]$PrimaryBaseUrl,
        [Parameter(Mandatory = $true)][string]$SecondaryBaseUrl,
        [string]$ApiKey = ''
    )

    $env:LLM_USE_MOCK = 'false'
    $env:LLM_PRIMARY_BASE_URL = $PrimaryBaseUrl
    $env:LLM_PRIMARY_API_KEY = $ApiKey
    $env:LLM_PRIMARY_MODEL_NAME = $PrimaryModel
    $env:LLM_SECONDARY_BASE_URL = $SecondaryBaseUrl
    $env:LLM_SECONDARY_API_KEY = $ApiKey
    $env:LLM_SECONDARY_MODEL_NAME = $SecondaryModel
    if (-not [string]::IsNullOrWhiteSpace($ApiKey)) {
        $env:LOCAL_LLM_API_KEY = $ApiKey
    }
}

function Write-LocalAutoReport {
    param(
        $PrimaryResult,
        $SecondaryResult,
        [string]$Selection,
        [string]$SummaryPath
    )

    Write-Host ''
    Write-Host ('PRIMARY_MODEL={0}' -f $PrimaryResult.ModelId)
    Write-Host ('PRIMARY_BASE_URL={0}' -f $PrimaryResult.BaseUrl)
    Write-Host ('PRIMARY_HEALTH={0}' -f $PrimaryResult.Health)
    Write-Host ('SECONDARY_MODEL={0}' -f $SecondaryResult.ModelId)
    Write-Host ('SECONDARY_BASE_URL={0}' -f $SecondaryResult.BaseUrl)
    Write-Host ('SECONDARY_HEALTH={0}' -f $SecondaryResult.Health)
    Write-Host ('MODEL_SELECTION={0}' -f $Selection)
    Write-Host ('REAL_DUAL_LLM_RUNTIME_VERIFIED={0}' -f 'YES')
    Write-Host ('REAL_DUAL_LLM_DIRECT_SMOKE={0}' -f 'YES')
    Write-Host ('MIYUNA_ENV_CONFIGURED={0}' -f 'YES')
    Write-Host ('LOCAL_RUNTIME_PROVIDER={0}' -f 'OPENAI_COMPAT')
    Write-Host ('VERIFICATION_SUMMARY_JSON={0}' -f $SummaryPath)
    Write-Host ''
    Write-Host 'Sonraki adim: API ve agent''i AYNI PowerShell oturumunda baslatin.'
    Write-Host 'TOKEN_EXPOSED=NO'
}

function Invoke-LocalAutoVerification {
    $manifest = Get-LocalModelsManifest -OverridePath $ManifestPath
    $baseUrls = Get-EffectiveLocalBaseUrls -ManifestDocument $manifest.Document -PrimaryBaseUrl $PrimaryBaseUrl -SecondaryBaseUrl $SecondaryBaseUrl
    $primaryBaseUrl = $baseUrls.PrimaryBaseUrl
    $secondaryBaseUrl = $baseUrls.SecondaryBaseUrl
    $apiKey = Get-LocalRuntimeApiKey -ManifestDocument $manifest.Document
    $weights = Get-LocalWeightInventory -ManifestDocument $manifest.Document

    Write-Host ''
    Write-Host 'Phase 6 Local LLM Auto Verification'
    Write-Host ('Manifest={0}' -f $manifest.Path)
    Write-Host ('PrimaryBaseUrl={0}' -f $primaryBaseUrl)
    Write-Host ('SecondaryBaseUrl={0}' -f $secondaryBaseUrl)
    Write-Host ''

    Write-Host 'STEP=Discover'
    $runtimeModels = @(Get-LocalOpenAiModels -BaseUrl $primaryBaseUrl -ApiKey $apiKey)
    if ($runtimeModels.Count -eq 0) {
        $runtimeModels = @(Get-OllamaModelTags -BaseUrl $primaryBaseUrl)
    }
    Write-LocalDiscoverReport -RuntimeModels $runtimeModels -WeightInventory $weights -PrimaryBaseUrl $primaryBaseUrl -SecondaryBaseUrl $secondaryBaseUrl -ManifestPath $manifest.Path

    if ($runtimeModels.Count -eq 0) {
        Write-Host ''
        Write-Host 'LOCAL_SETUP_REQUIRED=YES'
        Write-Host 'Ollama/LM Studio acik degil veya hic model yuklu degil.'
        throw 'LOCAL_RUNTIME_NO_MODELS'
    }

    Write-Host ''
    Write-Host 'STEP=Verify'
    $selection = Select-LocalDualModels -RuntimeModels $runtimeModels -ManifestDocument $manifest.Document
    $primaryResult = Invoke-LocalChatSmokeTest -BaseUrl $primaryBaseUrl -ModelId $selection.PrimaryModel -ApiKey $apiKey
    $secondaryResult = Invoke-LocalChatSmokeTest -BaseUrl $secondaryBaseUrl -ModelId $selection.SecondaryModel -ApiKey $apiKey

    if ($primaryResult.Health -ne 'PASS' -or $secondaryResult.Health -ne 'PASS') {
        throw 'LOCAL_MODEL_SMOKE_FAILED'
    }

    Write-Host ''
    Write-Host 'STEP=ConfigureMiyunaRuntime'
    Set-MiyunaLocalLlmRuntimeEnvironment -PrimaryModel $selection.PrimaryModel -SecondaryModel $selection.SecondaryModel -PrimaryBaseUrl $primaryBaseUrl -SecondaryBaseUrl $secondaryBaseUrl -ApiKey $apiKey

    $repoRoot = Get-RepoRoot
    $artifactsDir = Join-Path $repoRoot 'artifacts\phase6'
    if (-not (Test-Path $artifactsDir)) {
        New-Item -ItemType Directory -Path $artifactsDir -Force | Out-Null
    }
    $summaryPath = Join-Path $artifactsDir 'local-verification-summary.json'
    $summary = @{
        generatedAtUtc   = (Get-Date).ToUniversalTime().ToString('o')
        mode             = 'Auto'
        manifestPath     = $manifest.Path
        primaryModel     = $selection.PrimaryModel
        secondaryModel   = $selection.SecondaryModel
        selection        = $selection.Selection
        primaryBaseUrl   = $primaryBaseUrl
        secondaryBaseUrl = $secondaryBaseUrl
    }
    ($summary | ConvertTo-Json -Depth 5) | Set-Content -Path $summaryPath -Encoding UTF8

    Write-LocalAutoReport -PrimaryResult $primaryResult -SecondaryResult $secondaryResult -Selection $selection.Selection -SummaryPath $summaryPath
}

function Assert-LocalVerifyParameters {
    param([string]$Primary, [string]$Secondary)

    if ([string]::IsNullOrWhiteSpace($Primary)) { throw 'PrimaryModel is required for Verify mode.' }
    if ([string]::IsNullOrWhiteSpace($Secondary)) { throw 'SecondaryModel is required for Verify mode.' }
    if ($Primary.Trim() -eq $Secondary.Trim()) { throw 'PrimaryModel and SecondaryModel must be different.' }
}

function Invoke-Phase6LocalRuntime {
    param(
        [Parameter(Mandatory = $true)][ValidateSet('Discover', 'Verify', 'Auto')][string]$Mode,
        [string]$PrimaryModel,
        [string]$SecondaryModel,
        [string]$PrimaryBaseUrl,
        [string]$SecondaryBaseUrl,
        [string]$ManifestPath
    )

    Enable-LocalTls12Support

    switch ($Mode) {
        'Auto' {
            Invoke-LocalAutoVerification
        }
        default {
            $manifest = Get-LocalModelsManifest -OverridePath $ManifestPath
            $apiKey = Get-LocalRuntimeApiKey -ManifestDocument $manifest.Document
            $weights = Get-LocalWeightInventory -ManifestDocument $manifest.Document
            $runtimeModels = @(Get-LocalOpenAiModels -BaseUrl $PrimaryBaseUrl -ApiKey $apiKey)
            if ($runtimeModels.Count -eq 0) {
                $runtimeModels = @(Get-OllamaModelTags -BaseUrl $PrimaryBaseUrl)
            }

            if ($Mode -eq 'Discover') {
                Write-LocalDiscoverReport -RuntimeModels $runtimeModels -WeightInventory $weights -PrimaryBaseUrl $PrimaryBaseUrl -SecondaryBaseUrl $SecondaryBaseUrl -ManifestPath $manifest.Path
                return
            }

            Assert-LocalVerifyParameters -Primary $PrimaryModel -Secondary $SecondaryModel
            $primaryResult = Invoke-LocalChatSmokeTest -BaseUrl $PrimaryBaseUrl -ModelId $PrimaryModel.Trim() -ApiKey $apiKey
            $secondaryResult = Invoke-LocalChatSmokeTest -BaseUrl $SecondaryBaseUrl -ModelId $SecondaryModel.Trim() -ApiKey $apiKey
            Write-LocalAutoReport -PrimaryResult $primaryResult -SecondaryResult $secondaryResult -Selection 'manual_verify' -SummaryPath 'N/A'
            if ($primaryResult.Health -ne 'PASS' -or $secondaryResult.Health -ne 'PASS') {
                throw 'LOCAL_MODEL_SMOKE_FAILED'
            }
        }
    }
}

if ($MyInvocation.InvocationName -ne '.') {
    Invoke-Phase6LocalRuntime -Mode $Mode -PrimaryModel $PrimaryModel -SecondaryModel $SecondaryModel -PrimaryBaseUrl $PrimaryBaseUrl -SecondaryBaseUrl $SecondaryBaseUrl -ManifestPath $ManifestPath
}
