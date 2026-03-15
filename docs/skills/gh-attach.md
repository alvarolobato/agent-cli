# Skill: Uploading Images to GitHub Issues and Pull Requests

**Use when**: You need to attach screenshots, diagrams, or other images to a GitHub Issue or Pull Request comment using the CLI. Prefer **uploading** images with gh-attach (they appear as GitHub-hosted URLs in the body) over committing screenshot files to the repo for PR evidence.

## When a PR requires screenshots (e.g. OpenHASP widget changes)

1. **Capture the right screen**: If the screenshot must show a specific widget (e.g. HVAC, button), navigate to the page that contains it before capturing. See [openhasp-development.md](openhasp-development.md) (troubleshooting: "Screenshot shows wrong page") for how to send `hasp/<plate>/command/page` to navigate to the right page.
2. Take the screenshot (e.g. `ha dev hasp screenshot --plate mac-square`).
3. Upload to the PR with gh-attach and optional body:
   ```bash
   gh attach --pr <number> --repo owner/repo \
     --image ./path/to/mac-square.png \
     --image ./path/to/mac-rect.png \
     --body "**default-square (480×480):** <!-- gh-attach:IMAGE:1 -->

   **default-portrait (320×480):** <!-- gh-attach:IMAGE:2 -->"
   ```
   Or add images to the PR description when creating/editing the PR via the web UI (drag-and-drop); gh-attach is for CLI/automation.

## Why gh-attach exists

GitHub's REST API has no endpoint to upload images. The web UI uploads them to a private CDN via a browser-authenticated session. **gh-attach** automates that browser session with Playwright (headless Chromium) so images can be attached from the CLI, including in CI/headless environments.

- Extension repo: https://github.com/atani/gh-attach
- Article: https://zenn.dev/atani/articles/gh-attach-github-image-upload

## Prerequisites

Both are installed automatically by `scripts/setup-env.sh` and `.claude/hooks/session-start.sh`:

```bash
# 1. Install the gh CLI extension
gh extension install atani/gh-attach

# 2. Install Playwright + Chromium (used by gh-attach in browser mode)
pip install playwright
python3 -m playwright install chromium
```

gh must be authenticated (`gh auth login` or `GH_TOKEN` env var) before gh-attach can upload.

## Basic Usage

```bash
# Attach a single image to an issue
gh attach --issue 42 --image ./screenshot.png

# Attach with a comment body
gh attach --issue 42 --image ./result.png --body "E2E test result"

# Attach to a pull request instead of an issue
gh attach --pr 7 --image ./before.png

# Explicitly set repo (defaults to current git remote)
gh attach --issue 42 --image ./screenshot.png --repo owner/repo-name
```

## Multiple Images with Precise Placement

Use `<!-- gh-attach:IMAGE:N -->` placeholders in the `--body` to control where each image appears:

```bash
gh attach --issue 42 \
  --image ./before.png \
  --image ./after.png \
  --body "**Before:** <!-- gh-attach:IMAGE:1 -->

**After:** <!-- gh-attach:IMAGE:2 -->"
```

Images are numbered starting from 1 in the order they are passed.

## Upload Modes

| Mode | Flag | When to use |
|------|------|-------------|
| Browser (default) | _(none)_ | github.com — uses Playwright/Chromium with your gh session |
| Release | `--release` | Uploads to a GitHub Release asset instead; no browser needed |
| Direct (GHE) | automatic | GitHub Enterprise with `upload/policies` API support |

## Image Sizing

Default display width is 800 px. Override with `--width`:

```bash
gh attach --issue 42 --image ./wide.png --width 1200
```

## GitHub Enterprise

```bash
gh attach --issue 42 --image ./screenshot.png \
  --host github.mycompany.com \
  --repo myorg/myrepo
```

## Workflow: Playwright Screenshots → Issue Comment

A common pattern when writing automated tests or investigating UI bugs:

### 1. Take a screenshot with Playwright (Python)

```python
from playwright.sync_api import sync_playwright

with sync_playwright() as p:
    browser = p.chromium.launch()
    page = browser.new_page()
    page.goto("http://localhost:8080")
    page.screenshot(path="screenshots/dashboard.png")
    browser.close()
```

### 2. Upload the screenshot to the issue

```bash
gh attach --issue 42 \
  --image screenshots/dashboard.png \
  --body "Dashboard screenshot captured by automated test"
```

### 3. Combined in a shell script

```bash
#!/usr/bin/env bash
ISSUE=$1
python3 e2e/capture_screenshots.py          # writes screenshots/
for f in screenshots/*.png; do
    gh attach --issue "$ISSUE" --image "$f" --body "$(basename "$f")"
done
```

## Troubleshooting

**Upload fails / auth error**
: Confirm `gh auth status` shows authenticated. If using `GH_TOKEN`, verify the token has `repo` scope.

**Playwright browser not found**
: Run `python3 -m playwright install chromium`.

**Slow first run**
: Playwright launches a headless browser on first upload — normal, takes ~5 s.

**GitHub Enterprise not detected**
: Pass `--host github.mycompany.com` explicitly.

