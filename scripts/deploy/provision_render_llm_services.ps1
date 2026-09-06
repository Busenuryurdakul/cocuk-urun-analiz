# Provision Miyuna Render private LLM services (Ollama).
#
# Recommended: apply render.yaml Blueprint (sets dockerfile paths + disks):
#   https://dashboard.render.com/blueprint/new?repo=https://github.com/Busenuryurdakul/cocuk-urun-analiz
#
# CLI fallback (requires paid plan; dockerfile path may need Dashboard tweak):
#   .\scripts\deploy\provision_render_llm_services.ps1

param(
    [string]$Branch = 'feature/phase-6-llm-gateway',
    [string]$Repo = 'https://github.com/Busenuryurdakul/cocuk-urun-analiz',
    [string]$Region = 'frankfurt',
    [string]$Plan = 'standard',
    [switch]$SkipBillingProbe
)

$ErrorActionPreference = 'Stop'

function Test-RenderServiceExists {
    param([string]$Name)
    $json = render services list -o json | ConvertFrom-Json
    foreach ($item in $json) {
        $svc = $item.service
        if ($svc -and $svc.name -eq $Name) {
            return $svc.id
        }
    }
    return $null
}

Write-Host '=== Miyuna Render LLM provisioning ===' -ForegroundColor Cyan
Write-Host ''
Write-Host 'Blueprint (recommended):'
Write-Host "  $Repo/tree/$Branch/render.yaml"
Write-Host '  https://dashboard.render.com/blueprint/new?repo=https://github.com/Busenuryurdakul/cocuk-urun-analiz'
Write-Host ''

$primary = Test-RenderServiceExists -Name 'miyuna-llm-primary'
$secondary = Test-RenderServiceExists -Name 'miyuna-llm-secondary'
if ($primary -and $secondary) {
    Write-Host "LLM services already exist:" -ForegroundColor Green
    Write-Host "  miyuna-llm-primary   $primary"
    Write-Host "  miyuna-llm-secondary $secondary"
    Write-Host ''
    Write-Host 'Next: .\scripts\deploy\configure_render_production.ps1 -LlmBackend render'
    exit 0
}

if (-not $SkipBillingProbe) {
    Write-Host 'Probing Render billing for private services...'
    try {
        render services create `
            --name 'miyuna-llm-probe' `
            --type private_service `
            --runtime docker `
            --repo $Repo `
            --branch $Branch `
            --region $Region `
            --plan $Plan `
            --confirm `
            -o json | Out-Null
        Write-Host 'Billing OK — private services can be created.' -ForegroundColor Green
        render services delete miyuna-llm-probe --confirm -o json 2>$null | Out-Null
    } catch {
        if ($_.Exception.Message -match '402|Payment information') {
            Write-Host ''
            Write-Host 'Render billing required for private LLM services.' -ForegroundColor Yellow
            Write-Host '  1. Add card: https://dashboard.render.com/billing'
            Write-Host '  2. Apply Blueprint (link above) — creates miyuna-llm-primary + miyuna-llm-secondary'
            Write-Host '  3. Run: .\scripts\deploy\configure_render_production.ps1 -LlmBackend render'
            Write-Host ''
            Write-Host 'Interim: use Hugging Face without Render LLM:'
            Write-Host '  .\scripts\deploy\configure_render_production.ps1 -LlmBackend huggingface'
            exit 2
        }
        throw
    }
}

Write-Host ''
Write-Host 'Create services via Blueprint Apply in Dashboard (CLI cannot set custom dockerfile paths).' -ForegroundColor Yellow
Write-Host 'After Apply, run:'
Write-Host '  .\scripts\deploy\configure_render_production.ps1 -LlmBackend render'
