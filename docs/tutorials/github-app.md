# Setting Up the GitHub App for PR Checks

This tutorial covers setting up arx's PR check functionality — automated architecture audits on every pull request, with GitHub Check Runs and optional auto-approve.

## Prerequisites

- A GitHub repository with arx configured (`arx.yaml`)
- `arx baseline` already run (to suppress existing violations)
- Go 1.25+ installed on the CI runner

## Step 1: Create a GitHub App

1. Go to **Settings → Developer settings → GitHub Apps → New GitHub App**
2. Fill in:
   - **GitHub App name**: `arx-architecture-check`
   - **Homepage URL**: Your repo URL
   - **Webhook URL**: `https://your-server.example.com/api/github-webhook`
   - **Webhook secret**: Generate a random secret (save this!)
3. **Permissions**:
   - **Pull requests**: Read & write (to create check runs)
   - **Checks**: Read & write
   - **Contents**: Read
   - **Metadata**: Read (automatic)
4. **Subscribe to events**: `Pull request`
5. **Create App**

After creation, note the **App ID** and generate a **private key** (download the `.pem` file).

## Step 2: Configure arx server with webhook support

Start `arx server` with the webhook configuration:

```bash
arx server --port 8080 &
```

For production, you'll want to configure the PR check service. The server needs:

- The webhook secret (for HMAC verification)
- A configured PR check service

Example using the build-in integration (the server auto-configures when running with the proper setup):

```bash
# The server reads arx.yaml and starts with PR check support
# GitHub webhook is available at POST /api/github-webhook
arx server
```

## Step 3: Configure the webhook endpoint

Use a reverse proxy (nginx, Caddy) or a tunneling service (ngrok, Cloudflare Tunnel) to expose the webhook:

```nginx
# nginx configuration
server {
    listen 443 ssl;
    server_name arx-bot.example.com;

    location /api/github-webhook {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Step 4: Configure the repo webhook

In your GitHub repo:

1. Go to **Settings → Webhooks → Add webhook**
2. **Payload URL**: `https://arx-bot.example.com/api/github-webhook`
3. **Content type**: `application/json`
4. **Secret**: The webhook secret from Step 1
5. **Events**: Select **Pull requests**
6. **Active**: Yes

## Step 5: Test with a PR

1. Create a branch with an architecture violation
2. Open a PR against `main`
3. The webhook fires → arx checks the PR → posts a Check Run

Check Run output:

```
✅ arx — Architecture Audit

**Summary:**
- 2 violations found
- 1 error, 1 warning

**New violations (PR-introduced):**
| File | Line | Rule | Severity |
|------|------|------|----------|
| internal/domain/user.go | 42 | domain-no-infra | error |

**Conclusion:** FAILURE — new violations must be fixed.
```

## Step 6: Auto-approve (optional)

When the PR check passes (no new violations), arx can auto-approve:

```bash
arx pr-check --base main --head feature/branch --approve
```

This requires the GitHub App to have **Pull requests: Write** permission.

The auto-approve action is logged:

```
✅ No violations — auto-approve triggered
```

## Step 7: Using arx pr-check in CI

For environments without the full webhook setup, run `arx pr-check` directly in CI:

```yaml
# .github/workflows/arx-pr.yml
name: Architecture PR Check

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Need full history for diff

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.25"

      - name: Install arx
        run: go install github.com/pauvalls/arx/cmd/arx@latest

      - name: PR architecture check
        id: arx
        run: |
          arx pr-check \
            --base origin/${{ github.base_ref }} \
            --head ${{ github.head_ref }} \
            --json \
            --verbose
```

## Configuration Reference

The PR check behavior is controlled by `.github/arx-config.yaml` in your repo:

```yaml
# .github/arx-config.yaml
severity_thresholds:
  error: failure
  warning: neutral
  info: success

auto_approve: true           # Auto-approve PRs with no violations
auto_approve_on: success     # Only auto-approve when conclusion is "success"
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Webhook returns 401 | Wrong secret | Verify webhook secret matches `arx server` config |
| Check Run not created | Missing permissions | Ensure App has Checks: Write permission |
| Wrong violations reported | Diff filtering issue | Verify `--base` and `--head` are correct |
| Timeout on large PRs | Too many files changed | Consider diff sampling or increased timeout |
