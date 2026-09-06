# Fix Miyuna production login: set Atlas-backed MONGODB_URI on Render and redeploy API.
# Usage (after resetting password in Atlas UI):
#   $env:MONGODB_PASSWORD = 'YourAtlasPassword'
#   .\scripts\deploy\fix_production_login.ps1
#
# Or paste full URI:
#   $env:MONGODB_URI = 'mongodb+srv://...'
#   .\scripts\deploy\fix_production_login.ps1 -UseFullUri

param(
    [switch]$UseFullUri,
    [string]$MongoUser = 'yurdakulbusenur38_db_user',
    [string]$MongoHost = 'cluster0.wnbyhet.mongodb.net',
    [string]$MongoDatabase = 'miyuna',
    [string]$MongoAuthSource = 'admin'
)

$ErrorActionPreference = 'Stop'
$scriptDir = $PSScriptRoot

if (-not $UseFullUri) {
    if (-not $env:MONGODB_PASSWORD) {
        $secure = Read-Host 'Atlas DB password (input hidden)' -AsSecureString
        $ptr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)
        try {
            $env:MONGODB_PASSWORD = [Runtime.InteropServices.Marshal]::PtrToStringAuto($ptr)
        } finally {
            [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($ptr)
        }
    }
    $env:MONGODB_USER = $MongoUser
    $env:MONGODB_HOST = $MongoHost
    $env:MONGODB_DATABASE = $MongoDatabase
    $env:MONGODB_AUTH_SOURCE = $MongoAuthSource
    $env:MONGODB_URI = python (Join-Path $scriptDir 'build_mongodb_uri.py')
    if ($LASTEXITCODE -ne 0) { throw 'Failed to build MongoDB URI.' }
}

if ([string]::IsNullOrWhiteSpace($env:MONGODB_URI)) {
    throw 'MONGODB_URI is required. Set MONGODB_PASSWORD or MONGODB_URI.'
}

Write-Host 'Updating Render miyuna-api MONGODB_URI...'
& (Join-Path $scriptDir 'update_render_mongodb_uri.ps1')

Write-Host 'Waiting 90s for Render cold start...'
Start-Sleep -Seconds 90

try {
    $health = Invoke-RestMethod -Uri 'https://miyuna-api.onrender.com/health' -TimeoutSec 60
    Write-Host "Health OK: $($health | ConvertTo-Json -Compress)" -ForegroundColor Green
} catch {
    Write-Host "Health check failed: $($_.Exception.Message)" -ForegroundColor Yellow
    Write-Host 'Recent logs:'
    render logs -r srv-dae8m7dbedkc73ajnipg --limit 8 -o text 2>&1 | Select-Object -Last 12
    exit 1
}

Write-Host 'Production API is up. Test login at https://miyuna-web.vercel.app/auth/login'
