# Update MAIL_* env vars on Render miyuna-api (preserves other env vars).
# Resend example:
#   $env:MAIL_SMTP_HOST = 'smtp.resend.com'
#   $env:MAIL_SMTP_PORT = '465'
#   $env:MAIL_SMTP_USER = 'resend'
#   $env:MAIL_SMTP_PASS = 're_...'
#   $env:MAIL_SMTP_TLS = 'true'
#   $env:MAIL_FROM = 'onboarding@resend.dev'
#   .\scripts\deploy\update_render_mail_env.ps1

$ErrorActionPreference = 'Stop'

function Get-RenderApiKey {
    if ($env:RENDER_API_KEY) { return $env:RENDER_API_KEY }
    $cfgPath = Join-Path $env:USERPROFILE '.render\cli.yaml'
    if (-not (Test-Path $cfgPath)) { throw 'Render API key missing. Run: render login' }
    foreach ($line in Get-Content $cfgPath) {
        if ($line -match '^\s*key:\s*(.+)$') { return $Matches[1].Trim() }
    }
    throw 'Could not read Render API key from cli.yaml'
}

$serviceId = if ($env:RENDER_API_SERVICE_ID) { $env:RENDER_API_SERVICE_ID } else { 'srv-dae8m7dbedkc73ajnipg' }
$required = @('MAIL_SMTP_HOST', 'MAIL_SMTP_PORT', 'MAIL_FROM')
foreach ($name in $required) {
    if ([string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($name))) {
        Write-Host "$name is required."
        exit 1
    }
}

$mailVars = @{
    MAIL_SMTP_HOST = $env:MAIL_SMTP_HOST
    MAIL_SMTP_PORT = $env:MAIL_SMTP_PORT
    MAIL_FROM      = $env:MAIL_FROM
    MAIL_SMTP_TLS  = if ($env:MAIL_SMTP_TLS) { $env:MAIL_SMTP_TLS } else { 'true' }
    MAIL_QUEUE_ENABLED = if ($env:MAIL_QUEUE_ENABLED) { $env:MAIL_QUEUE_ENABLED } else { 'true' }
}
if ($env:MAIL_SMTP_USER) { $mailVars['MAIL_SMTP_USER'] = $env:MAIL_SMTP_USER }
if ($env:MAIL_SMTP_PASS) { $mailVars['MAIL_SMTP_PASS'] = $env:MAIL_SMTP_PASS }

$apiKey = Get-RenderApiKey
$headers = @{
    Authorization = "Bearer $apiKey"
    Accept        = 'application/json'
}
$getUri = "https://api.render.com/v1/services/$serviceId/env-vars"
$existing = Invoke-RestMethod -Method Get -Uri $getUri -Headers $headers

$vars = @{}
foreach ($item in $existing) {
    $ev = $item.envVar
    if ($ev.key) { $vars[$ev.key] = [string]$ev.value }
}
foreach ($key in $mailVars.Keys) {
    $vars[$key] = [string]$mailVars[$key]
}

$body = @(
    foreach ($key in ($vars.Keys | Sort-Object)) {
        @{ key = $key; value = $vars[$key] }
    }
) | ConvertTo-Json -Depth 3 -Compress

Invoke-RestMethod -Method Put -Uri $getUri -Headers ($headers + @{ 'Content-Type' = 'application/json' }) -Body $body | Out-Null

Write-Host "Updated MAIL env on $serviceId (total vars: $($vars.Count))"
Write-Host 'Triggering redeploy...'
render deploys create $serviceId --confirm -o json | Out-Null
Write-Host 'Done. Re-register or use Resend email resend to test verification mail.'
