# Push Kaggle fine-tune notebook and start a GPU session.
# Requires: py -m pip install kaggle && kaggle auth login (or KAGGLE_API_TOKEN)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $Root

Write-Host "Checking Kaggle auth..."
py -m kaggle kernels list -p 1 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) {
    Write-Host ""
    Write-Host "Kaggle auth required. Run ONE of:"
    Write-Host "  py -m kaggle auth login"
    Write-Host "  `$env:KAGGLE_API_TOKEN = '<token from https://www.kaggle.com/settings/api>'"
    exit 1
}

Write-Host "Pushing kernel and starting GPU run..."
py -m kaggle kernels push -p . --accelerator gpu

$slug = (Get-Content kernel-metadata.json | ConvertFrom-Json).id
Write-Host ""
Write-Host "Monitor: https://www.kaggle.com/code/$slug"
Write-Host "Status:  py -m kaggle kernels status $slug"
Write-Host "Logs:    py -m kaggle kernels logs $slug -v"
