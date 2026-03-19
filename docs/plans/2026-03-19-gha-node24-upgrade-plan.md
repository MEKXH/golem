# GitHub Actions Node 24 Upgrade Plan

1. Update `.github/workflows/ci.yml`
   - upgrade `actions/checkout` to `v5`
   - upgrade `actions/setup-go` to `v6`

2. Update `.github/workflows/release.yml`
   - upgrade `actions/checkout` to `v5`
   - upgrade `actions/setup-go` to `v6`
   - upgrade `actions/upload-artifact` to `v6`
   - upgrade `actions/download-artifact` to `v8`
   - add `FORCE_JAVASCRIPT_ACTIONS_TO_NODE24=true` at workflow scope

3. Verify
   - review workflow diffs
   - run `go test ./...`
