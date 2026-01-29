param(
    [string]$CodexCmd = "codex",
    [string[]]$CodexArgs = @("--suggest"),
    [string]$TestCommand = "",
    [switch]$AutoPass
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$ralphDir = Join-Path $repoRoot "ralph"
$prdPath = Join-Path $ralphDir "prd.json"
$promptPath = Join-Path $ralphDir "prompt.md"
$progressPath = Join-Path $ralphDir "progress.txt"
$lastPromptPath = Join-Path $ralphDir "last_prompt.md"

if (-not (Test-Path $ralphDir)) {
    New-Item -ItemType Directory -Force -Path $ralphDir | Out-Null
}

if (-not (Test-Path $promptPath)) {
@"
# Ralph loop prompt (Codex)

You are running inside a loop. Each run is a fresh Codex session.

Requirements:
- Implement only the selected story.
- If you discover blockers, write them to ralph/progress.txt.
- If you finish, update ralph/progress.txt with key context for the next run.
- If you finish, set the story's "passes" to true in ralph/prd.json.
"@ | Set-Content -Path $promptPath -Encoding utf8
}

if (-not (Test-Path $progressPath)) {
    "" | Set-Content -Path $progressPath -Encoding utf8
}

if (-not (Test-Path $prdPath)) {
@"
{
  "stories": [
    {
      "id": "example-1",
      "title": "Add a small improvement",
      "prompt": "Describe the change to implement here.",
      "passes": false
    }
  ]
}
"@ | Set-Content -Path $prdPath -Encoding utf8
}

function Get-Stories {
    param([object]$Prd)
    if ($null -eq $Prd) { return @() }
    if ($Prd -is [System.Collections.IEnumerable] -and -not ($Prd -is [string])) {
        return @($Prd)
    }
    if ($Prd.PSObject.Properties.Name -contains "stories") {
        return @($Prd.stories)
    }
    if ($Prd.PSObject.Properties.Name -contains "items") {
        return @($Prd.items)
    }
    return @()
}

$prdRaw = Get-Content -Raw -Path $prdPath
$prd = $prdRaw | ConvertFrom-Json
$stories = Get-Stories -Prd $prd

if ($stories.Count -eq 0) {
    Write-Host "No stories found in ralph/prd.json."
    exit 1
}

$storyIndex = -1
for ($i = 0; $i -lt $stories.Count; $i++) {
    if ($stories[$i].passes -ne $true) {
        $storyIndex = $i
        break
    }
}

if ($storyIndex -lt 0) {
    Write-Host "All stories already pass."
    exit 0
}

$story = $stories[$storyIndex]
$storyText = $story | ConvertTo-Json -Depth 10
$promptHeader = Get-Content -Raw -Path $promptPath
$progress = Get-Content -Raw -Path $progressPath

$fullPrompt = @"
$promptHeader

Repository root: $repoRoot
Story:
$storyText

Progress so far:
$progress
"@

$fullPrompt | Set-Content -Path $lastPromptPath -Encoding utf8

Write-Host "Running Codex for story index $storyIndex ($($story.id))..."
$fullPrompt | & $CodexCmd @CodexArgs

if ($LASTEXITCODE -ne 0) {
    Write-Host "Codex exited with code $LASTEXITCODE."
    exit $LASTEXITCODE
}

if ($TestCommand -ne "") {
    Write-Host "Running tests: $TestCommand"
    & $env:ComSpec /c $TestCommand
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Tests failed with code $LASTEXITCODE."
        exit $LASTEXITCODE
    }
}

if ($AutoPass) {
    $stories[$storyIndex].passes = $true
    $prd | ConvertTo-Json -Depth 10 | Set-Content -Path $prdPath -Encoding utf8
    Write-Host "Marked story as passes=true."
} else {
    Write-Host "AutoPass disabled; leaving story status unchanged."
}
