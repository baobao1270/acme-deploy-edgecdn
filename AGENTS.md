# AI Agent Guide

This file provides working guidance for any AI coding agent contributing to this repository.

## Build and test

```bash
make build                  # Debug build to dist/acme-deploy-edgecdn
make release                # Release builds for supported platforms
make clean                  # Remove generated binaries
go test ./...               # Run unit tests
```

Run the binary directly with explicit CLI input:

```bash
./dist/acme-deploy-edgecdn \
  -config config.yaml \
  -caller cli \
  -domain example.com \
  -cert /path/to/fullchain.pem \
  -key /path/to/privkey.pem
```

## Architecture

This Go CLI deploys renewed ACME certificates to edge CDN providers. `main.go` parses flags, validates the YAML configuration, selects profiles whose domain matches the caller input, then calls the selected provider.

`config.go` defines a profile-oriented configuration. Every profile has a provider `type`, expected `domain`, optional `caller`, and provider-specific configuration. Valid types are `esa` (Alibaba Cloud ESA) and `eo` (Tencent Cloud EdgeOne). The configuration validator rejects duplicate `(type, domain)` tuples unless `ACME_DEPLOY_ALLOW_DUPLICATE_PROFILES=1` is set; it always logs a warning for duplicates.

`provider/provider.go` defines the common `Provider` interface and `CertData`. Provider implementations are isolated under `provider/`:

- `alicloud_esa.go` uploads or updates ESA certificates by name.
- `tencentcloud_teo.go` uploads to Tencent Cloud SSL, binds EdgeOne hosts, then attempts expired-certificate cleanup.

## Certificate callers

Profiles default to `cli` when `caller` is omitted. The `-caller` flag overrides the caller configured on every profile. `-domain` always overrides the caller-provided domain, while `-cert` and `-key` override caller-provided certificate paths.

- `acme.sh`: `Le_Domain`, `CERT_FULLCHAIN_PATH`, `CERT_KEY_PATH`
- `lego`: `LEGO_HOOK_CERT_NAME`, `LEGO_HOOK_CERT_PATH`, `LEGO_HOOK_CERT_KEY_PATH`
- `cli`: requires `-domain`, `-cert`, and `-key`

The input domain must match the profile domain. Mismatches are logged and skipped without calling a provider API.

## Working conventions

Keep provider SDK calls and credentials inside provider/configuration code. Do not log certificate PEM data, private keys, or credential values. Use standard-library unit tests for caller resolution, configuration validation, and certificate parsing; provider API calls should not run in tests.
