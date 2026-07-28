# SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
# SPDX-License-Identifier: MIT
#
# Windows repeated-test runner for the port allocation fix.
#
# Usage (PowerShell, from the repo's portdemo folder):
#   .\run.ps1              # mechanism demo + repeated real tests (-count 30)
#   .\run.ps1 -Count 100   # more repetitions
#   .\run.ps1 -Count 0     # mechanism demo only
#
# Prerequisite: Go 1.24+ on PATH (winget install GoLang.Go).

param(
    [int]$Count = 30
)

$ErrorActionPreference = "Stop"

Write-Host "=== Excluded TCP port ranges on this machine ==="
netsh interface ipv4 show excludedportrange protocol=tcp

Write-Host ""
Write-Host "=== Mechanism demo: old pattern vs new pattern (3000 iterations each) ==="
Push-Location $PSScriptRoot
go run .
Pop-Location

if ($Count -le 0) {
    Write-Host ""
    Write-Host "Count is 0; skipped the repeated real-test run."
    exit 0
}

Write-Host ""
Write-Host "=== Repeated real tests: previously flaky trio, -count=$Count ==="
# -race on Windows needs a C toolchain (gcc); fall back to a plain run without one.
$raceFlag = @()
if (Get-Command gcc -ErrorAction SilentlyContinue) {
    $raceFlag = @("-race")
} else {
    Write-Host "gcc not found; running without -race (fine for the port-bind check)"
}
Push-Location (Join-Path $PSScriptRoot "..")
go test @raceFlag "-count=$Count" -timeout 60m `
    -run 'TestMultiTCPMuxUsage|TestTURNConcurrency|TestMultiUDPMuxUsage' .
Pop-Location
