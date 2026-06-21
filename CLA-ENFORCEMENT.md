# CLA Enforcement Setup (GitHub Rulesets)

## Step 1 — Navigate to Rulesets

1. Open your repo: `https://github.com/Linky-Link-Linky/Agent-Nervous-System`
2. Click **Settings** (top nav bar)
3. In the left sidebar, click **Rules** → **Rulesets**

## Step 2 — Create the Ruleset

1. Click **New ruleset** → **New branch ruleset**

## Step 3 — Configure

| Field | Value |
|---|---|
| **Ruleset name** | `CLA Enforcement` |
| **Enforcement status** | `Active` |
| **Target branches** | `Add target` → `refs/heads/master` (or just `master`) |
| **Branch rules** | Check **Require status check to pass** |
| **Status checks** | Click **Add check** → type `CLA` → select `CLA Assistant / cla-check` |
| **Bypass list** | Leave empty (or add yourself if you want to bypass) |
| **Pull request** | Also check **Require a pull request before merging** → **Require approvals** = 1 |

## Step 4 — Save

Click **Create** at the bottom.

## How it works

Now every PR to `master` will show a `CLA Assistant / cla-check` status check. It blocks merging until the contributor comments:

> I have read the CLA and I hereby sign it.

on the PR. The workflow then records the signature in `.github/cla-contributors.json` and the check passes.

## Manual override (before merge)

If someone needs to merge without the check (e.g., you as the owner), you can bypass the ruleset by adding your account to the **Bypass list** in the ruleset settings.
