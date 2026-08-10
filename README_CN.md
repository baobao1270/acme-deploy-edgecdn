# acme-deploy-edgecdn

将续期后的 ACME 证书部署到边缘 CDN 服务商。它可以由 acme.sh 的 reload command、lego hook，或普通 CLI 调用。

## 支持的服务商

| 类型 | 服务商 | 操作 |
| --- | --- | --- |
| `esa` | 阿里云 ESA | 使用 ESA `SetCertificate` API 上传或更新证书。 |
| `eo` | 腾讯云 EdgeOne | 上传到 SSL 证书管理服务，再绑定到 EdgeOne 域名。 |

## 快速开始

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

## 配置文件

默认读取 `/etc/acme-deploy-edgecdn.yaml`；使用 `-config` 可以覆盖配置文件路径。配置文件包含一个或多个部署 profile。省略 `caller` 时，默认值是 `cli`。

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

凭证也可以使用环境变量提供：

- ESA：`ALICLOUD_ACCESS_KEY_ID`、`ALICLOUD_ACCESS_KEY_SECRET`
- EdgeOne：`TENCENTCLOUD_SECRET_ID`、`TENCENTCLOUD_SECRET_KEY`

调用任何服务商 API 前，程序会检查是否存在相同的 `(type, domain)` profile。发现重复时默认拒绝运行；设置 `ACME_DEPLOY_ALLOW_DUPLICATE_PROFILES=1` 可以继续执行，但警告仍会写入日志。

## 证书调用方

`-caller` 支持 `acme.sh`、`lego`、`cli`，并覆盖每个 profile 中的 `caller`。`-domain` 永远优先于调用方提供的域名；`-cert` 和 `-key` 同样优先于调用方提供的证书与私钥路径。

| 调用方 | 域名环境变量 | 证书环境变量 |
| --- | --- | --- |
| `acme.sh` | `Le_Domain` | `CERT_FULLCHAIN_PATH`、`CERT_KEY_PATH` |
| `lego` | `LEGO_HOOK_CERT_NAME` | `LEGO_HOOK_CERT_PATH`、`LEGO_HOOK_CERT_KEY_PATH` |
| `cli` | 不适用 | 不适用 |

`cli` 模式必须填写 `-domain`、`-cert`、`-key`。只有输入域名与 profile 的 `domain` 一致时才会实际部署；不一致会记录日志并跳过。

## 配合 acme.sh 使用

```bash
acme.sh --install-cert -d example.com \
  --fullchain-file /etc/acme/fullchain.pem \
  --key-file /etc/acme/key.pem \
  --reloadcmd "/path/to/acme-deploy-edgecdn -config /path/to/config.yaml"
```

## 配合 lego 使用

将 profile 的 `caller` 设置为 `lego`，再将本程序用作 lego 的部署 hook。lego v5 会向 hook 传入 `LEGO_HOOK_CERT_NAME`、`LEGO_HOOK_CERT_PATH`、`LEGO_HOOK_CERT_KEY_PATH`。
