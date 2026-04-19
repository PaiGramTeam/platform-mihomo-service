# Integration Test Scripts

These scripts wrap the current integration-test workflow for `platform-mihomo-service`.

## Environment

1. Use `.env.integration.example` as the template for `.env.integration.local`.
2. Set the required MySQL variables with the approved `PAI_TEST_DATABASE_*` names.
3. Redis variables are optional for now and only need to be set when a test requires Redis.

The scripts and the raw Go commands intentionally run with `GOWORK=off` so local `go.work` changes do not leak into normal integration runs.

`go.work` is only for explicit unpublished `proto-contracts` verification.

## Commands

- `./scripts/integration.sh doctor`
- `./scripts/integration.sh test`
- `./scripts/integration.sh deps-up`
- `./scripts/integration.sh deps-down`
- `pwsh -File ./scripts/integration.ps1 doctor`
- `pwsh -File ./scripts/integration.ps1 test`
- `pwsh -File ./scripts/integration.ps1 deps-up`
- `pwsh -File ./scripts/integration.ps1 deps-down`

`doctor` checks the configured dependencies only. It does not start Docker.

`test` runs `go run ./cmd/integration-doctor` first and then executes the integration suite. It does not start Docker.

`deps-up` and `deps-down` manage the optional local stack through `docker-compose.integration.yml`.

## Raw Go Command

Run the suite directly without the wrapper scripts:

```sh
GOWORK=off go test -tags=integration ./integration/...
```

If you want the same preflight check used by the scripts, run:

```sh
GOWORK=off go run ./cmd/integration-doctor
```
