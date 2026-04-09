# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
make build                # debug build → dist/acme-deploy-edgecdn
make release              # release builds for all platforms → dist/
make clean                # remove dist/

./dist/acme-deploy-edgecdn --config config.yaml --cert fullchain.pem --key privkey.pem --domain example.com
./dist/acme-deploy-edgecdn --version
```

No tests or linter are configured yet.

## Architecture

This is a Go CLI tool that deploys ACME certificates to edge CDN providers. It's designed to be called as an acme.sh `--reloadcmd` after certificate renewal.

**Module path:** `github.com/acme-deploy-edgecdn` (note: directory is still named `acme-deploy-ncdn`)

**Flow:** `main.go` parses flags → loads YAML config (`config.go`) → builds a provider via `buildProvider()` → calls `provider.Deploy()`.

**Provider interface** (`provider/provider.go`): All providers implement `Deploy(ctx, *CertData) (*DeployResult, error)`. `CertData` holds domain, fullchain PEM, and private key PEM. `DeployResult` returns cert ID and request ID.

**Providers:**

- `alicloud-esa` (`provider/alicloud_esa.go`): Deploys to Alibaba Cloud ESA via `SetCertificate`. Finds existing cert by name and updates in-place (upsert).
- `tencentcloud-teo` (`provider/tencentcloud_teo.go`): Two-step deploy — uploads cert to TencentCloud SSL (`UploadCertificate`), then binds to EdgeOne zone hosts (`ModifyHostsCertificate`). Appends `-YYYYMMDD` date suffix to cert alias. After deploy, cleans up expired certs matching the alias prefix.

**Config** (`config.go`): Single YAML file with a `provider` field selecting which provider block to use. Credentials fall back to environment variables (`TENCENTCLOUD_SECRET_ID`/`TENCENTCLOUD_SECRET_KEY` for TEO, `ALICLOUD_ACCESS_KEY_ID`/`ALICLOUD_ACCESS_KEY_SECRET` for ESA).
