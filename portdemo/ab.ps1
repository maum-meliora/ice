# SPDX-FileCopyrightText: 2026 The Pion community <https://pion.ly>
# SPDX-License-Identifier: MIT
#
# One-shot before/after demo for the port allocation fix.
#
# Positions the Windows UDP ephemeral-port cursor at the edge of a TCP
# excluded range, then runs TestMultiTCPMuxUsage on the pre-fix commit
# (expected: FAIL) and on the fixed code (expected: ok).
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
    Write-Host "--- $Label / cursor walk ---"
    go run (Join-Path $repoRoot "portdemo\walk")
    if ($LASTEXITCODE -ne 0) { return $null }
    Write-Host "--- $Label / TestMultiTCPMuxUsage ---"
    Push-Location $Dir
    go test "-count=1" -timeout 5m -run TestMultiTCPMuxUsage .
    $code = $LASTEXITCODE
    Pop-Location
    return $code
}

# The cursor can drift if another app binds UDP between walk and test,
# letting the old code slip past the range — retry the pair a few times.
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
