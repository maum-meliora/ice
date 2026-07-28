# SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
# SPDX-License-Identifier: MIT
#
# One-shot before/after demo for the port allocation fix.
#
# Occupies TCP ports just ahead of the UDP ephemeral-port cursor (UDP side
# left free), then runs TestMultiTCPMuxUsage on the pre-fix commit
# (expected: FAIL - its UDP-picked port is TCP-busy) and on the fixed code
# (expected: ok - bind :0 skips busy ports).
#
# Usage: powershell -ExecutionPolicy Bypass -File .\portdemo\ab.ps1

param(
    [string]$BeforeCommit = "f33931e",
    [int]$MaxAttempts = 3
)

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$beforeDir = Join-Path (Split-Path $repoRoot -Parent) "ice-before"

if (-not (Test-Path $beforeDir)) {
    Write-Host "creating worktree for pre-fix commit $BeforeCommit ..."
    git -C $repoRoot worktree add $beforeDir $BeforeCommit
    if ($LASTEXITCODE -ne 0) { Write-Host "worktree add failed"; exit 1 }
}

function Invoke-DemoStep([string]$Label, [string]$Dir) {
    Write-Host ""
    Write-Host "--- $Label / occupying TCP ports ahead of the UDP cursor ---"
    $occupier = Start-Process go -ArgumentList "run", (Join-Path $repoRoot "portdemo\occupy") `
        -NoNewWindow -PassThru
    # Give `go run` time to compile and reach READY (it prints to this console).
    Start-Sleep -Seconds 6
    Write-Host "--- $Label / TestMultiTCPMuxUsage ---"
    Push-Location $Dir
    # Out-Host keeps command output on screen instead of it becoming the
    # function return value (PowerShell functions return all pipeline output).
    go test "-count=1" -timeout 5m -run TestMultiTCPMuxUsage . 2>&1 | Out-Host
    $code = $LASTEXITCODE
    Pop-Location
    Stop-Process -Id $occupier.Id -Force -ErrorAction SilentlyContinue
    return $code
}

# Retry a few times in case heavy background UDP traffic moves the cursor
# out of the occupied window between occupation and the test's port picks.
$beforeCode = $null
for ($i = 1; $i -le $MaxAttempts; $i++) {
    $beforeCode = Invoke-DemoStep "BEFORE old code, attempt $i" $beforeDir
    if ($null -eq $beforeCode) { exit 1 }
    if ($beforeCode -ne 0) { break }
    Write-Host "old code passed this attempt (cursor drifted?) - retrying"
}

$afterCode = Invoke-DemoStep "AFTER fixed code" $repoRoot
if ($null -eq $afterCode) { exit 1 }

Write-Host ""
Write-Host "=== Summary ==="
if ($beforeCode -ne 0) {
    Write-Host "before (old randomPort code): FAIL  <- expected"
} else {
    Write-Host "before (old randomPort code): pass  <- could not trigger in $MaxAttempts attempts"
}
if ($afterCode -eq 0) {
    Write-Host "after  (fixed code):          ok    <- expected"
} else {
    Write-Host "after  (fixed code):          FAIL  <- unexpected, investigate"
}
