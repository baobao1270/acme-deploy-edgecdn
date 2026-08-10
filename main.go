package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/acme-deploy-edgecdn/provider"
)

var version = "dev"

type certificateInput struct {
	Domain   string
	CertPath string
	KeyPath  string
}

func main() {
	var (
		configPath  = flag.String("config", "/etc/acme-deploy-edgecdn.yaml", "path to config file")
		certPath    = flag.String("cert", "", "path to fullchain certificate PEM (overrides caller environment)")
		keyPath     = flag.String("key", "", "path to private key PEM (overrides caller environment)")
		domain      = flag.String("domain", "", "domain name (overrides caller environment)")
		callerFlag  = flag.String("caller", "", "certificate caller: acme.sh, lego, or cli (overrides profile caller)")
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

	var callerOverride Caller
	var err error
	if *callerFlag != "" {
		callerOverride, err = parseCaller(Caller(*callerFlag))
		if err != nil {
			log.Fatalf("parsing caller: %v", err)
		}
	}

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}
	warnings, err := cfg.Validate(os.Getenv(allowDuplicateProfilesEnv) == "1")
	for _, warning := range warnings {
		log.Printf("WARNING: %s", warning)
	}
	if err != nil {
		log.Fatalf("validating config: %v", err)
	}

	deployed := 0
	for i, profile := range cfg.Profiles {
		caller := callerOverride
		if caller == "" {
			caller, err = parseCaller(profile.Caller)
			if err != nil {
				log.Fatalf("profile %d: %v", i+1, err)
			}
		}

		inputDomain, err := resolveDomain(caller, *domain)
		if err != nil {
			log.Fatalf("profile %d: resolving domain: %v", i+1, err)
		}
		if inputDomain == "" {
			log.Printf("skipping profile %d: caller %q did not provide a domain", i+1, caller)
			continue
		}
		if normalizeDomain(inputDomain) != normalizeDomain(profile.Domain) {
			log.Printf("skipping profile %d: input domain %q does not match configured domain %q", i+1, inputDomain, profile.Domain)
			continue
		}
		resolvedCertPath, resolvedKeyPath, err := resolveCertificatePaths(caller, *certPath, *keyPath)
		if err != nil {
			log.Fatalf("profile %d: resolving certificate paths: %v", i+1, err)
		}

		certData, err := loadCertData(certificateInput{
			Domain:   inputDomain,
			CertPath: resolvedCertPath,
			KeyPath:  resolvedKeyPath,
		})
		if err != nil {
			log.Fatalf("profile %d: loading certificate data: %v", i+1, err)
		}

		p, err := buildProvider(profile)
		if err != nil {
			log.Fatalf("profile %d: initializing provider: %v", i+1, err)
		}
		result, err := p.Deploy(context.Background(), certData)
		if err != nil {
			log.Fatalf("profile %d: deploy failed: %v", i+1, err)
		}
		deployed++
		log.Printf("profile %d deploy succeeded: cert_id=%s request_id=%s", i+1, result.CertID, result.RequestID)
	}

	if deployed == 0 {
		log.Printf("no profile matched the supplied domain; nothing deployed")
	}
}

func resolveDomain(caller Caller, domainFlag string) (string, error) {
	if domainFlag != "" {
		return domainFlag, nil
	}
	switch caller {
	case CallerAcmeSH:
		return os.Getenv("Le_Domain"), nil
	case CallerLego:
		return os.Getenv("LEGO_HOOK_CERT_NAME"), nil
	case CallerCLI:
		return "", fmt.Errorf("--domain is required when caller is cli")
	default:
		return "", fmt.Errorf("unsupported caller %q", caller)
	}
}

func resolveCertificatePaths(caller Caller, certFlag, keyFlag string) (string, string, error) {
	certPath, keyPath := certFlag, keyFlag
	if caller == CallerCLI {
		if certPath == "" {
			return "", "", fmt.Errorf("--cert is required when caller is cli")
		}
		if keyPath == "" {
			return "", "", fmt.Errorf("--key is required when caller is cli")
		}
		return certPath, keyPath, nil
	}

	var certEnv, keyEnv string
	switch caller {
	case CallerAcmeSH:
		certEnv, keyEnv = "CERT_FULLCHAIN_PATH", "CERT_KEY_PATH"
	case CallerLego:
		certEnv, keyEnv = "LEGO_HOOK_CERT_PATH", "LEGO_HOOK_CERT_KEY_PATH"
	default:
		return "", "", fmt.Errorf("unsupported caller %q", caller)
	}
	if certPath == "" {
		certPath = os.Getenv(certEnv)
	}
	if keyPath == "" {
		keyPath = os.Getenv(keyEnv)
	}
	if certPath == "" {
		return "", "", fmt.Errorf("certificate path not provided: use --cert or set %s", certEnv)
	}
	if keyPath == "" {
		return "", "", fmt.Errorf("private key path not provided: use --key or set %s", keyEnv)
	}
	return certPath, keyPath, nil
}

func loadCertData(input certificateInput) (*provider.CertData, error) {
	certPEM, err := os.ReadFile(input.CertPath)
	if err != nil {
		return nil, fmt.Errorf("reading certificate file %s: %w", input.CertPath, err)
	}
	keyPEM, err := os.ReadFile(input.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("reading private key file %s: %w", input.KeyPath, err)
	}

	return &provider.CertData{
		Domain:     input.Domain,
		FullChain:  string(certPEM),
		PrivateKey: string(keyPEM),
	}, nil
}

func buildProvider(profile Profile) (provider.Provider, error) {
	switch strings.ToLower(strings.TrimSpace(profile.Type)) {
	case "esa":
		return provider.NewAlicloudESA(provider.AlicloudESAConfig{
			AccessKeyID:     profile.ESA.AccessKeyID,
			AccessKeySecret: profile.ESA.AccessKeySecret,
			Endpoint:        profile.ESA.Endpoint,
			SiteID:          profile.ESA.SiteID,
			CertName:        profile.ESA.CertName,
			Region:          profile.ESA.Region,
		})
	case "eo":
		return provider.NewTencentCloudTEO(provider.TencentCloudTEOConfig{
			SecretID:    profile.EO.SecretID,
			SecretKey:   profile.EO.SecretKey,
			ZoneID:      profile.EO.ZoneID,
			Hosts:       profile.EO.Hosts,
			CertName:    profile.EO.CertName,
			Region:      profile.EO.Region,
			SSLEndpoint: profile.EO.SSLEndpoint,
			TEOEndpoint: profile.EO.TEOEndpoint,
		})
	default:
		return nil, fmt.Errorf("unknown provider type: %s", profile.Type)
	}
}
