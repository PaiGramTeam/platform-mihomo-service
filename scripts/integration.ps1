param(
    [Parameter(Position = 0)]
    [string]$Command
)

$ErrorActionPreference = 'Stop'

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$ComposeFile = Join-Path $RepoRoot 'docker-compose.integration.yml'

if ([string]::IsNullOrWhiteSpace($Command)) {
    $Command = 'test'
}

function Show-Usage {
    Write-Host 'Usage: ./scripts/integration.ps1 [doctor|test|deps-up|deps-down]'
}

function Invoke-Go {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Arguments
    )

    $previousGoWork = $env:GOWORK
    try {
        $env:GOWORK = 'off'
        & go @Arguments
        if ($LASTEXITCODE -ne 0) {
            exit $LASTEXITCODE
        }
    }
    finally {
        if ($null -eq $previousGoWork) {
            Remove-Item Env:GOWORK -ErrorAction SilentlyContinue
        }
        else {
            $env:GOWORK = $previousGoWork
        }
    }
}

function Assert-DockerCompose {
    $docker = Get-Command docker -ErrorAction SilentlyContinue
    if ($null -eq $docker) {
        throw 'Docker is required but was not found in PATH.'
    }

    & docker compose version | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw 'Docker Compose is required and docker compose is not available.'
    }
}

Push-Location $RepoRoot
try {
    switch ($Command) {
        'doctor' {
            Invoke-Go -Arguments @('run', './cmd/integration-doctor')
        }
        'test' {
            Invoke-Go -Arguments @('run', './cmd/integration-doctor')
            Invoke-Go -Arguments @('test', '-tags=integration', './integration/...')
        }
        'deps-up' {
            Assert-DockerCompose
            & docker compose -f $ComposeFile up -d
            if ($LASTEXITCODE -ne 0) {
                exit $LASTEXITCODE
            }
        }
        'deps-down' {
            Assert-DockerCompose
            & docker compose -f $ComposeFile down --remove-orphans
            if ($LASTEXITCODE -ne 0) {
                exit $LASTEXITCODE
            }
        }
        default {
            Show-Usage
            exit 1
        }
    }
}
finally {
    Pop-Location
}
