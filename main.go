package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/acme-deploy-edgecdn/provider"
)

var version = "dev"

func main() {
	var (
		configPath  = flag.String("config", "", "path to config file (required)")
		certPath    = flag.String("cert", "", "path to fullchain certificate PEM (overrides CERT_FULLCHAIN_PATH)")
		keyPath     = flag.String("key", "", "path to private key PEM (overrides CERT_KEY_PATH)")
		domain      = flag.String("domain", "", "domain name (overrides Le_Domain)")
		showVersion = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	if *configPath == "" {
		log.Fatal("--config is required")
	}

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	certData, err := loadCertData(*certPath, *keyPath, *domain)
	if err != nil {
		log.Fatalf("loading certificate data: %v", err)
	}

	p, err := buildProvider(cfg)
	if err != nil {
		log.Fatalf("initializing provider: %v", err)
	}

	result, err := p.Deploy(context.Background(), certData)
	if err != nil {
		log.Fatalf("deploy failed: %v", err)
	}

	log.Printf("deploy succeeded: cert_id=%s request_id=%s", result.CertID, result.RequestID)
}

func loadCertData(certFlag, keyFlag, domainFlag string) (*provider.CertData, error) {
	certPath := certFlag
	if certPath == "" {
		certPath = os.Getenv("CERT_FULLCHAIN_PATH")
	}
	if certPath == "" {
		return nil, fmt.Errorf("certificate path not provided: use --cert flag or set CERT_FULLCHAIN_PATH")
	}

	keyPath := keyFlag
	if keyPath == "" {
		keyPath = os.Getenv("CERT_KEY_PATH")
	}
	if keyPath == "" {
		return nil, fmt.Errorf("private key path not provided: use --key flag or set CERT_KEY_PATH")
	}

	domain := domainFlag
	if domain == "" {
		domain = os.Getenv("Le_Domain")
	}
	if domain == "" {
		return nil, fmt.Errorf("domain not provided: use --domain flag or set Le_Domain")
	}

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("reading certificate file %s: %w", certPath, err)
	}

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading private key file %s: %w", keyPath, err)
	}

	return &provider.CertData{
		Domain:     domain,
		FullChain:  string(certPEM),
		PrivateKey: string(keyPEM),
	}, nil
}

func buildProvider(cfg *Config) (provider.Provider, error) {
	switch cfg.Provider {
	case "alicloud-esa":
		return provider.NewAlicloudESA(provider.AlicloudESAConfig{
			AccessKeyID:     cfg.AlicloudESA.AccessKeyID,
			AccessKeySecret: cfg.AlicloudESA.AccessKeySecret,
			Endpoint:        cfg.AlicloudESA.Endpoint,
			SiteID:          cfg.AlicloudESA.SiteID,
			CertName:        cfg.AlicloudESA.CertName,
			Region:          cfg.AlicloudESA.Region,
		})
	case "tencentcloud-teo":
		return provider.NewTencentCloudTEO(provider.TencentCloudTEOConfig{
			SecretID:    cfg.TencentCloudTEO.SecretID,
			SecretKey:   cfg.TencentCloudTEO.SecretKey,
			ZoneID:      cfg.TencentCloudTEO.ZoneID,
			Hosts:       cfg.TencentCloudTEO.Hosts,
			CertName:    cfg.TencentCloudTEO.CertName,
			Region:      cfg.TencentCloudTEO.Region,
			SSLEndpoint: cfg.TencentCloudTEO.SSLEndpoint,
			TEOEndpoint: cfg.TencentCloudTEO.TEOEndpoint,
		})
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}
