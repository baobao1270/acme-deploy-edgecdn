package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Provider        string              `yaml:"provider"`
	AlicloudESA     AlicloudESAConf     `yaml:"alicloud-esa"`
	TencentCloudTEO TencentCloudTEOConf `yaml:"tencentcloud-teo"`
}

type AlicloudESAConf struct {
	AccessKeyID     string `yaml:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret"`
	Endpoint        string `yaml:"endpoint"`
	SiteID          int64  `yaml:"site_id"`
	CertName        string `yaml:"cert_name"`
	Region          string `yaml:"region"`
}

type TencentCloudTEOConf struct {
	SecretID    string   `yaml:"secret_id"`
	SecretKey   string   `yaml:"secret_key"`
	ZoneID      string   `yaml:"zone_id"`
	Hosts       []string `yaml:"hosts"`
	CertName    string   `yaml:"cert_name"`
	Region      string   `yaml:"region"`
	SSLEndpoint string   `yaml:"ssl_endpoint"`
	TEOEndpoint string   `yaml:"teo_endpoint"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Env var fallback for credentials
	if cfg.AlicloudESA.AccessKeyID == "" {
		cfg.AlicloudESA.AccessKeyID = os.Getenv("ALICLOUD_ACCESS_KEY_ID")
	}
	if cfg.AlicloudESA.AccessKeySecret == "" {
		cfg.AlicloudESA.AccessKeySecret = os.Getenv("ALICLOUD_ACCESS_KEY_SECRET")
	}

	if cfg.TencentCloudTEO.SecretID == "" {
		cfg.TencentCloudTEO.SecretID = os.Getenv("TENCENTCLOUD_SECRET_ID")
	}
	if cfg.TencentCloudTEO.SecretKey == "" {
		cfg.TencentCloudTEO.SecretKey = os.Getenv("TENCENTCLOUD_SECRET_KEY")
	}

	return &cfg, nil
}
