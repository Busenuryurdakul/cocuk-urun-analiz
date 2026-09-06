# Provision MongoDB Atlas M0 cluster for Miyuna production (Frankfurt).
# Requires one-time browser login or Atlas API keys.
#
# Option A — interactive (recommended first time):
#   .\scripts\deploy\provision_atlas_cluster.ps1
#
# Option B — API keys (CI / headless):
#   $env:MONGODB_ATLAS_PUBLIC_API_KEY = '...'
#   $env:MONGODB_ATLAS_PRIVATE_API_KEY = '...'
#   $env:MONGODB_ATLAS_ORG_ID = '...'   # required for new projects
#   .\scripts\deploy\provision_atlas_cluster.ps1 -NonInteractive
#
# Option C — one-shot interactive wizard (local terminal):
#   atlas setup --clusterName miyuna-prod --provider AWS --region EU_CENTRAL_1 --username miyuna_app --currentIp --connectWith skip
#
# After success, copy MONGODB_URI and run configure_render_production.ps1.

param(
    [string]$ClusterName = 'miyuna-prod',
    [string]$ProjectName = 'Miyuna',
    [string]$Region = 'EU_CENTRAL_1',
    [string]$DatabaseName = 'miyuna',
    [string]$DbUser = 'miyuna_app',
    [switch]$NonInteractive
)

$ErrorActionPreference = 'Stop'

function Get-AtlasExe {
    $cmd = Get-Command atlas -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }

    $candidates = @(
        'C:\Program Files (x86)\MongoDB Atlas CLI\atlas.exe',
        'C:\Program Files\MongoDB Atlas CLI\atlas.exe',
        (Join-Path $env:USERPROFILE 'tools\mongodb-atlas-cli\atlas.exe')
    )
    foreach ($path in $candidates) {
        if (Test-Path $path) { return $path }
    }

    throw @"
Atlas CLI not found. Install one of:
  winget install -e --id MongoDB.MongoDBAtlasCLI
  https://www.mongodb.com/try/download/atlascli
Then restart the terminal (or refresh PATH) and re-run this script.
"@
}

function New-RandomPassword {
    param([int]$Length = 24)
    $bytes = New-Object byte[] $Length
    [System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
    [Convert]::ToBase64String($bytes).TrimEnd('=').Replace('+', 'x').Replace('/', 'y').Substring(0, $Length)
}

function Ensure-AtlasAuth {
    param([string]$AtlasExe)
    if ($NonInteractive) {
        if (-not $env:MONGODB_ATLAS_PUBLIC_API_KEY -or -not $env:MONGODB_ATLAS_PRIVATE_API_KEY) {
            throw 'NonInteractive mode requires MONGODB_ATLAS_PUBLIC_API_KEY and MONGODB_ATLAS_PRIVATE_API_KEY.'
        }
        return
    }

    Write-Host 'Opening Atlas login in browser (one-time)...'
    & $AtlasExe auth login
}

function Get-OrCreate-Project {
    param([string]$AtlasExe)
    $projectsJson = & $AtlasExe projects list -o json | ConvertFrom-Json
    $existing = $projectsJson.results | Where-Object { $_.name -eq $ProjectName } | Select-Object -First 1
    if ($existing) {
        Write-Host "Using existing project: $ProjectName ($($existing.id))"
        return $existing.id
    }

    $orgId = $env:MONGODB_ATLAS_ORG_ID
    if (-not $orgId) {
        throw 'New project requires MONGODB_ATLAS_ORG_ID (Atlas → Organization Settings).'
    }
    Write-Host "Creating Atlas project: $ProjectName"
    $created = & $AtlasExe projects create $ProjectName --orgId $orgId -o json | ConvertFrom-Json
    return $created.id
}

function Ensure-Cluster {
    param(
        [string]$AtlasExe,
        [string]$ProjectId
    )
    $clustersJson = & $AtlasExe clusters list --projectId $ProjectId -o json | ConvertFrom-Json
    $existing = $clustersJson.results | Where-Object { $_.name -eq $ClusterName } | Select-Object -First 1
    if ($existing) {
        Write-Host "Cluster already exists: $ClusterName (state: $($existing.stateName))"
        return
    }

    Write-Host "Creating M0 cluster '$ClusterName' in $Region ..."
    & $AtlasExe clusters create $ClusterName `
        --projectId $ProjectId `
        --provider AWS `
        --region $Region `
        --tier M0 | Out-Null
}

function Wait-ClusterReady {
    param(
        [string]$AtlasExe,
        [string]$ProjectId
    )
    Write-Host 'Waiting for cluster to become available (2-5 min)...'
    for ($i = 0; $i -lt 60; $i++) {
        $cluster = & $AtlasExe clusters describe $ClusterName --projectId $ProjectId -o json | ConvertFrom-Json
        if ($cluster.stateName -eq 'IDLE') {
            Write-Host 'Cluster is ready.'
            return
        }
        Start-Sleep -Seconds 10
    }
    throw "Cluster '$ClusterName' did not become IDLE in time."
}

function Ensure-NetworkAccess {
    param(
        [string]$AtlasExe,
        [string]$ProjectId
    )
    Write-Host 'Allowing access from anywhere (0.0.0.0/0) for Render + dev...'
    $entries = & $AtlasExe accessLists list --projectId $ProjectId -o json | ConvertFrom-Json
    $hasOpen = $entries.results | Where-Object { $_.cidrBlock -eq '0.0.0.0/0' }
    if (-not $hasOpen) {
        & $AtlasExe accessLists create --projectId $ProjectId --cidr 0.0.0.0/0 --comment 'Render + dev' | Out-Null
    }
}

function Ensure-DbUser {
    param(
        [string]$AtlasExe,
        [string]$ProjectId,
        [ref]$PasswordOut
    )
    $usersJson = & $AtlasExe dbusers list --projectId $ProjectId -o json | ConvertFrom-Json
    $existing = $usersJson.results | Where-Object { $_.username -eq $DbUser } | Select-Object -First 1
    if ($existing) {
        Write-Host "DB user '$DbUser' already exists."
        if ($env:MIYUNA_ATLAS_DB_PASSWORD) {
            $PasswordOut.Value = $env:MIYUNA_ATLAS_DB_PASSWORD
        } else {
            Write-Host "Set MIYUNA_ATLAS_DB_PASSWORD if you need the URI for an existing user, or reset password in Atlas UI."
        }
        return
    }

    $pwd = New-RandomPassword
    $PasswordOut.Value = $pwd
    Write-Host "Creating DB user '$DbUser' ..."
    & $AtlasExe dbusers create `
        --projectId $ProjectId `
        --username $DbUser `
        --password $pwd `
        --role "readWrite@$DatabaseName" | Out-Null
}

function Build-MongoUri {
    param(
        [string]$AtlasExe,
        [string]$ProjectId,
        [string]$Password
    )
    $cluster = & $AtlasExe clusters describe $ClusterName --projectId $ProjectId -o json | ConvertFrom-Json
    $host = $cluster.connectionStrings.standardSrv
    if (-not $host) {
        throw 'Could not read SRV connection string from cluster.'
    }
    # host is like mongodb+srv://cluster0.xxxxx.mongodb.net
    $encodedUser = [uri]::EscapeDataString($DbUser)
    $encodedPass = [uri]::EscapeDataString($Password)
    $base = $host -replace '^mongodb\+srv://', ''
    return "mongodb+srv://${encodedUser}:${encodedPass}@${base}/${DatabaseName}?retryWrites=true&w=majority"
}

$atlasExe = Get-AtlasExe
Write-Host "Using Atlas CLI: $atlasExe"

Ensure-AtlasAuth -AtlasExe $atlasExe

$projectId = if ($env:MONGODB_ATLAS_PROJECT_ID) {
    Write-Host "Using project ID from MONGODB_ATLAS_PROJECT_ID"
    $env:MONGODB_ATLAS_PROJECT_ID
} else {
    Get-OrCreate-Project -AtlasExe $atlasExe
}

Ensure-Cluster -AtlasExe $atlasExe -ProjectId $projectId
Wait-ClusterReady -AtlasExe $atlasExe -ProjectId $projectId
Ensure-NetworkAccess -AtlasExe $atlasExe -ProjectId $projectId

$dbPassword = $null
Ensure-DbUser -AtlasExe $atlasExe -ProjectId $projectId -PasswordOut ([ref]$dbPassword)

if ($dbPassword) {
    $uri = Build-MongoUri -AtlasExe $atlasExe -ProjectId $projectId -Password $dbPassword
    Write-Host ''
    Write-Host '=== Atlas cluster ready ==='
    Write-Host "Project: $ProjectName"
    Write-Host "Cluster: $ClusterName ($Region, M0)"
    Write-Host "Database: $DatabaseName"
    Write-Host ''
    Write-Host 'Set for Render deploy:'
    Write-Host "  `$env:MONGODB_URI = '$uri'"
    Write-Host ''
    Write-Host 'Then run:'
    Write-Host '  .\scripts\deploy\configure_render_production.ps1'
} else {
    Write-Host ''
    Write-Host 'Cluster ready. Build MONGODB_URI from Atlas Dashboard → Connect → Drivers.'
}
