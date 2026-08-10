# acme-deploy-edgecdn

[中文文档](README_CN.md)

Deploy renewed ACME certificates to edge CDN providers. It can be used from an acme.sh reload command, a lego hook, or a regular CLI invocation.

## Supported providers

| Type | Provider | Action |
| --- | --- | --- |
| `esa` | Alibaba Cloud ESA | Uploads or updates a certificate using the ESA `SetCertificate` API. |
| `eo` | Tencent Cloud EdgeOne | Uploads to SSL Certificate Manager, then binds it to EdgeOne hosts. |

## Quick start

```bash
make build
cp config.yaml.example config.yaml
vi config.yaml

./dist/acme-deploy-edgecdn \
  -config config.yaml \
  -caller cli \
  -domain example.com \
  -cert /path/to/fullchain.pem \
  -key /path/to/privkey.pem
```

## Configuration

The configuration is read from `/etc/acme-deploy-edgecdn.yaml` by default; use `-config` to override it. It contains one or more deployment profiles. `caller` defaults to `cli` when omitted.

```yaml
profiles:
  - type: esa
    caller: acme.sh
    domain: example.com
    esa:
      access_key_id: ""
      access_key_secret: ""
      endpoint: esa.cn-hangzhou.aliyuncs.com
      site_id: 1234567890123

  - type: eo
    caller: lego
    domain: example.net
    eo:
      secret_id: ""
      secret_key: ""
      zone_id: zone-abc123
      hosts:
        - example.net
```

Credentials may instead come from environment variables:

- ESA: `ALICLOUD_ACCESS_KEY_ID`, `ALICLOUD_ACCESS_KEY_SECRET`
- EdgeOne: `TENCENTCLOUD_SECRET_ID`, `TENCENTCLOUD_SECRET_KEY`

Before any provider API call, the program validates that no two profiles have the same `(type, domain)` pair. To run despite duplicate profiles, set `ACME_DEPLOY_ALLOW_DUPLICATE_PROFILES=1`; the warning remains in the log.

## Certificate callers

`-caller` accepts `acme.sh`, `lego`, or `cli` and overrides the caller configured on every profile. `-domain` always takes precedence over the caller's domain. `-cert` and `-key` similarly override its certificate and private-key paths.

| Caller | Domain environment variable | Certificate environment variables |
| --- | --- | --- |
| `acme.sh` | `Le_Domain` | `CERT_FULLCHAIN_PATH`, `CERT_KEY_PATH` |
| `lego` | `LEGO_HOOK_CERT_NAME` | `LEGO_HOOK_CERT_PATH`, `LEGO_HOOK_CERT_KEY_PATH` |
| `cli` | Not applicable | Not applicable |

For `cli`, `-domain`, `-cert`, and `-key` are all required. A profile runs only when the supplied domain matches its `domain` field. A mismatch is logged and skipped.

## acme.sh usage

```bash
acme.sh --install-cert -d example.com \
  --fullchain-file /etc/acme/fullchain.pem \
  --key-file /etc/acme/key.pem \
  --reloadcmd "/path/to/acme-deploy-edgecdn -config /path/to/config.yaml"
```

## lego usage

Configure the profile with `caller: lego` and use the binary as lego's deployment hook. lego v5 passes `LEGO_HOOK_CERT_NAME`, `LEGO_HOOK_CERT_PATH`, and `LEGO_HOOK_CERT_KEY_PATH` to its hooks.
