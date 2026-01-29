# Codex project instructions

If you need a Ralph-style loop, use the script below (Windows PowerShell):

```
.\scripts\ralph-loop.ps1 -CodexArgs @("--suggest") -TestCommand "go test ./..." -AutoPass
```

Loop inputs live in `ralph/`:
- `ralph/prd.json` list of stories (each has a `passes` flag)
- `ralph/progress.txt` cross-run notes
- `ralph/prompt.md` shared instructions injected into each run

If those files are missing, the script creates minimal defaults.
