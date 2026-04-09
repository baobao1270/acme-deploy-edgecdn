# acme-deploy-edgecdn

[中文文档](README_CN.md)

Deploy ACME certificates to edge CDN providers. Designed to run as an [acme.sh](https://github.com/acmesh-official/acme.sh) `--reloadcmd` so certificates are automatically deployed on renewal.

## Supported providers

| Provider                   | Config key        | What it does                                                                                      |
| -------------------------- | ----------------- | ------------------------------------------------------------------------------------------------- |
| Alibaba Cloud ESA          | `alicloud-esa`    | Uploads/updates certificate via ESA `SetCertificate` API                                          |
| TencentCloud EdgeOne (TEO) | `tencentcloud-teo`| Uploads to SSL Certificate Manager, then binds to EdgeOne zone hosts via `ModifyHostsCertificate` |

## Quick start

```bash
# Build
make build

# Release build (all platforms)
make release

# Copy and edit the example config
cp config.yaml.example config.yaml
vi config.yaml

# Deploy a certificate
./dist/acme-deploy-edgecdn \
  --config config.yaml \
  --cert /path/to/fullchain.pem \
  --key /path/to/privkey.pem \
  --domain example.com
```

## Usage with acme.sh

```bash
acme.sh --install-cert -d example.com \
  --fullchain-file /etc/acme/fullchain.pem \
  --key-file /etc/acme/key.pem \
  --reloadcmd "/path/to/acme-deploy-edgecdn --config /path/to/config.yaml"
```

When used as a reloadcmd, the tool reads `CERT_FULLCHAIN_PATH`, `CERT_KEY_PATH`, and `Le_Domain` from environment variables set by acme.sh, so the `--cert`, `--key`, and `--domain` flags can be omitted.

## Configuration

See [config.yaml.example](config.yaml.example) for all options. Credentials can also be provided via environment variables:

- **TencentCloud TEO:** `TENCENTCLOUD_SECRET_ID` / `TENCENTCLOUD_SECRET_KEY`
- **Alibaba Cloud ESA:** `ALICLOUD_ACCESS_KEY_ID` / `ALICLOUD_ACCESS_KEY_SECRET`
