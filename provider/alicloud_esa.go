package provider

import (
	"context"
	"fmt"
	"log"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	esa "github.com/alibabacloud-go/esa-20240910/v2/client"
	"github.com/alibabacloud-go/tea/dara"
)

type AlicloudESA struct {
	client   *esa.Client
	siteID   int64
	certName string
	region   string
}

type AlicloudESAConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	Endpoint        string
	SiteID          int64
	CertName        string
	Region          string
}

func NewAlicloudESA(cfg AlicloudESAConfig) (*AlicloudESA, error) {
	if cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" {
		return nil, fmt.Errorf("access_key_id and access_key_secret are required")
	}
	if cfg.SiteID == 0 {
		return nil, fmt.Errorf("site_id is required")
	}
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}

	config := &openapi.Config{
		AccessKeyId:     dara.String(cfg.AccessKeyID),
		AccessKeySecret: dara.String(cfg.AccessKeySecret),
		Endpoint:        dara.String(cfg.Endpoint),
	}

	client, err := esa.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("creating ESA client: %w", err)
	}

	return &AlicloudESA{
		client:   client,
		siteID:   cfg.SiteID,
		certName: cfg.CertName,
		region:   cfg.Region,
	}, nil
}

// Deploy uploads or updates a certificate on the ESA site.
// It matches existing certificates by name only — if a certificate with the
// same name already exists, it will be replaced regardless of its SANs or
// other attributes. To deploy multiple distinct certificates for the same
// domain, use different cert_name values in the config.
func (a *AlicloudESA) Deploy(ctx context.Context, cert *CertData) (*DeployResult, error) {
	certName := a.certName
	if certName == "" {
		certName = "acme-" + cert.Domain
	}

	existingID, err := a.findExistingCert(certName)
	if err != nil {
		return nil, fmt.Errorf("searching for existing certificate: %w", err)
	}

	req := &esa.SetCertificateRequest{
		SiteId:      dara.Int64(a.siteID),
		Type:        dara.String("upload"),
		Certificate: dara.String(cert.FullChain),
		PrivateKey:  dara.String(cert.PrivateKey),
		Name:        dara.String(certName),
	}

	if a.region != "" {
		req.Region = dara.String(a.region)
	}

	if existingID != "" {
		req.Id = dara.String(existingID)
		log.Printf("updating existing certificate %q (id=%s) on site %d", certName, existingID, a.siteID)
	} else {
		log.Printf("creating new certificate %q on site %d", certName, a.siteID)
	}

	resp, err := a.client.SetCertificate(req)
	if err != nil {
		return nil, fmt.Errorf("SetCertificate API call failed: %w", err)
	}

	result := &DeployResult{}
	if resp.Body != nil {
		if resp.Body.Id != nil {
			result.CertID = *resp.Body.Id
		}
		if resp.Body.RequestId != nil {
			result.RequestID = *resp.Body.RequestId
		}
	}

	return result, nil
}

// findExistingCert lists all certificates on the site and finds one matching
// the given name with type "upload". We don't use the Keyword filter because
// the ESA API's keyword search doesn't reliably match certificate names.
func (a *AlicloudESA) findExistingCert(certName string) (string, error) {
	var pageNumber int64 = 1
	for {
		req := &esa.ListCertificatesRequest{
			SiteId:     dara.Int64(a.siteID),
			PageSize:   dara.Int64(50),
			PageNumber: dara.Int64(pageNumber),
		}

		resp, err := a.client.ListCertificates(req)
		if err != nil {
			return "", err
		}

		if resp.Body == nil || len(resp.Body.Result) == 0 {
			return "", nil
		}

		for _, cert := range resp.Body.Result {
			if dara.StringValue(cert.Name) == certName && dara.StringValue(cert.Type) == "upload" && dara.StringValue(cert.Id) != "" {
				return *cert.Id, nil
			}
		}

		total := dara.Int64Value(resp.Body.TotalCount)
		if pageNumber*50 >= total {
			return "", nil
		}
		pageNumber++
	}
}
