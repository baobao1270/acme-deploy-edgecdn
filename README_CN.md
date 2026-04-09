# acme-deploy-edgecdn

将 ACME 证书自动部署到边缘 CDN 服务商。可作为 [acme.sh](https://github.com/acmesh-official/acme.sh) 的 `--reloadcmd` 使用，在证书续期后自动完成部署。

## 支持的服务商

| 服务商                      | 配置键             | 说明                                                                  |
| -------------------------- | ------------------ | -------------------------------------------------------------------- |
| 阿里云 ESA                  | `alicloud-esa`     | 通过 ESA `SetCertificate` 接口上传或更新证书                             |
| 腾讯云 EdgeOne (TEO)        | `tencentcloud-teo` | 先上传至 SSL 证书管理，再通过 `ModifyHostsCertificate` 绑定到 EdgeOne 域名 |

## 快速开始

```bash
# 构建
make build

# 发布构建（所有平台）
make release

# 复制并编辑配置文件
cp config.yaml.example config.yaml
vi config.yaml

# 部署证书
./dist/acme-deploy-edgecdn \
  --config config.yaml \
  --cert /path/to/fullchain.pem \
  --key /path/to/privkey.pem \
  --domain example.com
```

## 配合 acme.sh 使用

```bash
acme.sh --install-cert -d example.com \
  --fullchain-file /etc/acme/fullchain.pem \
  --key-file /etc/acme/key.pem \
  --reloadcmd "/path/to/acme-deploy-edgecdn --config /path/to/config.yaml"
```

作为 reloadcmd 调用时，工具会从 acme.sh 设置的环境变量 `CERT_FULLCHAIN_PATH`、`CERT_KEY_PATH` 和 `Le_Domain` 中读取参数，因此可以省略 `--cert`、`--key` 和 `--domain` 标志。

## 配置

所有配置项参见 [config.yaml.example](config.yaml.example)。凭证也可以通过环境变量提供：

- **腾讯云 TEO：** `TENCENTCLOUD_SECRET_ID` / `TENCENTCLOUD_SECRET_KEY`
- **阿里云 ESA：** `ALICLOUD_ACCESS_KEY_ID` / `ALICLOUD_ACCESS_KEY_SECRET`
