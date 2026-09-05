# Phase 5 live integration: Go + Python + Redis + Mongo
# Requires Docker services from infra/docker/docker-compose.yml (Mongo 27017, Redis 6379)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location (Join-Path $Root "apps\api")

if (-not (Test-Path "bin")) { New-Item -ItemType Directory -Path "bin" | Out-Null }

Write-Host "Compiling Phase 5 integration tests..."
go test -c ./internal/integration/ -o bin/phase5.integration.test.exe
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "Running GraphQL + security integration tests..."
& .\bin\phase5.integration.test.exe "-test.v" "-test.timeout=5m" "-test.run=TestGraphQL|TestInternal|TestGrant|TestUnavailable|TestOversized|TestForged|TestEventPagination"
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "Running live Go/Python/Redis/Mongo E2E..."
& .\bin\phase5.integration.test.exe "-test.v" "-test.timeout=5m" "-test.run=TestLiveGoPythonRedisMongoE2E|TestE2ECancellation|TestE2EStaleLease"
exit $LASTEXITCODE
