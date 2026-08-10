package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const allowDuplicateProfilesEnv = "ACME_DEPLOY_ALLOW_DUPLICATE_PROFILES"

type Caller string

const (
	CallerAcmeSH Caller = "acme.sh"
	CallerLego   Caller = "lego"
	CallerCLI    Caller = "cli"
)

type Config struct {
	Profiles []Profile `yaml:"profiles"`
}

type Profile struct {
	Type   string              `yaml:"type"`
	Caller Caller              `yaml:"caller"`
	Domain string              `yaml:"domain"`
	ESA    AlicloudESAConf     `yaml:"esa"`
	EO     TencentCloudTEOConf `yaml:"eo"`
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

	for i := range cfg.Profiles {
		profile := &cfg.Profiles[i]
		if profile.ESA.AccessKeyID == "" {
			profile.ESA.AccessKeyID = os.Getenv("ALICLOUD_ACCESS_KEY_ID")
		}
		if profile.ESA.AccessKeySecret == "" {
			profile.ESA.AccessKeySecret = os.Getenv("ALICLOUD_ACCESS_KEY_SECRET")
		}
		if profile.EO.SecretID == "" {
			profile.EO.SecretID = os.Getenv("TENCENTCLOUD_SECRET_ID")
		}
		if profile.EO.SecretKey == "" {
			profile.EO.SecretKey = os.Getenv("TENCENTCLOUD_SECRET_KEY")
		}
	}

	return &cfg, nil
}

func (cfg *Config) Validate(allowDuplicates bool) ([]string, error) {
	if len(cfg.Profiles) == 0 {
		return nil, fmt.Errorf("profiles must contain at least one profile")
	}

	seen := make(map[string]int, len(cfg.Profiles))
	var warnings []string
	for i, profile := range cfg.Profiles {
		profileType := strings.TrimSpace(strings.ToLower(profile.Type))
		if profileType != "esa" && profileType != "eo" {
			return warnings, fmt.Errorf("profile %d: unsupported type %q (must be esa or eo)", i+1, profile.Type)
		}
		if strings.TrimSpace(profile.Domain) == "" {
			return warnings, fmt.Errorf("profile %d: domain is required", i+1)
		}
		if _, err := parseCaller(profile.Caller); err != nil {
			return warnings, fmt.Errorf("profile %d: %w", i+1, err)
		}

		key := profileType + "\x00" + normalizeDomain(profile.Domain)
		if firstIndex, ok := seen[key]; ok {
			warnings = append(warnings, fmt.Sprintf(
				"duplicate profile tuple (type=%q domain=%q) at profiles %d and %d",
				profileType, profile.Domain, firstIndex+1, i+1,
			))
			continue
		}
		seen[key] = i
	}

	if len(warnings) > 0 && !allowDuplicates {
		return warnings, fmt.Errorf("duplicate profiles are not allowed; set %s=1 to continue anyway", allowDuplicateProfilesEnv)
	}
	return warnings, nil
}

func parseCaller(value Caller) (Caller, error) {
	caller := Caller(strings.TrimSpace(string(value)))
	if caller == "" {
		return CallerCLI, nil
	}
	switch caller {
	case CallerAcmeSH, CallerLego, CallerCLI:
		return caller, nil
	default:
		return "", fmt.Errorf("unsupported caller %q (must be acme.sh, lego, or cli)", value)
	}
}

func normalizeDomain(domain string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
}
