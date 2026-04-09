package provider

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tcerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	ssl "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssl/v20191205"
	teo "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/teo/v20220901"
)

type TencentCloudTEO struct {
	sslClient *ssl.Client
	teoClient *teo.Client
	zoneID    string
	hosts     []string
	certName  string
}

type TencentCloudTEOConfig struct {
	SecretID    string
	SecretKey   string
	ZoneID      string
	Hosts       []string
	CertName    string
	Region      string // optional, defaults to empty (uses default endpoint)
	SSLEndpoint string // optional, override ssl endpoint (e.g. ssl.intl.tencentcloudapi.com)
	TEOEndpoint string // optional, override teo endpoint (e.g. teo.intl.tencentcloudapi.com)
}

func NewTencentCloudTEO(cfg TencentCloudTEOConfig) (*TencentCloudTEO, error) {
	if cfg.SecretID == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("secret_id and secret_key are required")
	}
	if cfg.ZoneID == "" {
		return nil, fmt.Errorf("zone_id is required")
	}
	if len(cfg.Hosts) == 0 {
		return nil, fmt.Errorf("hosts is required (at least one domain)")
	}

	credential := common.NewCredential(cfg.SecretID, cfg.SecretKey)

	// SSL client for uploading certificates
	sslProfile := profile.NewClientProfile()
	if cfg.SSLEndpoint != "" {
		sslProfile.HttpProfile.Endpoint = cfg.SSLEndpoint
	}
	sslClient, err := ssl.NewClient(credential, cfg.Region, sslProfile)
	if err != nil {
		return nil, fmt.Errorf("creating SSL client: %w", err)
	}

	// TEO client for deploying certificates to EdgeOne
	teoProfile := profile.NewClientProfile()
	if cfg.TEOEndpoint != "" {
		teoProfile.HttpProfile.Endpoint = cfg.TEOEndpoint
	}
	teoClient, err := teo.NewClient(credential, cfg.Region, teoProfile)
	if err != nil {
		return nil, fmt.Errorf("creating TEO client: %w", err)
	}

	return &TencentCloudTEO{
		sslClient: sslClient,
		teoClient: teoClient,
		zoneID:    cfg.ZoneID,
		hosts:     cfg.Hosts,
		certName:  cfg.CertName,
	}, nil
}

func (t *TencentCloudTEO) Deploy(ctx context.Context, cert *CertData) (*DeployResult, error) {
	certName := t.certName
	if certName == "" {
		certName = "acme-" + cert.Domain
	}
	// Append date suffix so each renewal is distinguishable.
	certNameWithDate := certName + "-" + time.Now().Format("20060102")

	// Step 1: Upload certificate to Tencent Cloud SSL
	certID, err := t.uploadCert(cert, certNameWithDate)
	if err != nil {
		return nil, fmt.Errorf("uploading certificate: %w", err)
	}
	log.Printf("uploaded certificate %q to SSL, cert_id=%s", certNameWithDate, certID)

	// Step 2: Deploy the certificate to EdgeOne hosts
	requestID, err := t.deployCert(certID)
	if err != nil {
		return nil, fmt.Errorf("deploying certificate to EdgeOne: %w", err)
	}

	// Step 3: Clean up expired certificates that share the same alias prefix.
	t.cleanupExpiredCerts(certName, certID)

	return &DeployResult{
		CertID:    certID,
		RequestID: requestID,
	}, nil
}

func (t *TencentCloudTEO) uploadCert(cert *CertData, certName string) (string, error) {
	req := ssl.NewUploadCertificateRequest()
	req.CertificatePublicKey = common.StringPtr(cert.FullChain)
	req.CertificatePrivateKey = common.StringPtr(cert.PrivateKey)
	req.CertificateType = common.StringPtr("SVR")
	req.Alias = common.StringPtr(certName)
	req.Repeatable = common.BoolPtr(false)

	resp, err := t.sslClient.UploadCertificate(req)
	if err != nil {
		// If the certificate already exists (same fingerprint), treat it as a
		// warning rather than a fatal error — reuse the existing cert ID.
		if sdkErr, ok := err.(*tcerrors.TencentCloudSDKError); ok && sdkErr.GetCode() == "FailedOperation.CertificateExists" {
			log.Printf("WARNING: certificate with the same fingerprint already exists, reusing existing cert")
			if resp != nil && resp.Response != nil && resp.Response.RepeatCertId != nil {
				return *resp.Response.RepeatCertId, nil
			}
			return "", fmt.Errorf("certificate already exists but no RepeatCertId returned: %w", err)
		}
		return "", err
	}

	if resp.Response == nil || resp.Response.CertificateId == nil {
		return "", fmt.Errorf("empty certificate ID in response")
	}

	// When Repeatable=false, the API may return RepeatCertId if the cert
	// content is identical to an existing one (without raising an error).
	if resp.Response.RepeatCertId != nil && *resp.Response.RepeatCertId != "" {
		log.Printf("WARNING: certificate with the same fingerprint already exists (repeat_cert_id=%s), reusing", *resp.Response.RepeatCertId)
		return *resp.Response.RepeatCertId, nil
	}

	return *resp.Response.CertificateId, nil
}

// cleanupExpiredCerts searches for certificates whose alias starts with aliasPrefix
// and deletes any that are expired (status=3), skipping the just-deployed cert.
func (t *TencentCloudTEO) cleanupExpiredCerts(aliasPrefix, currentCertID string) {
	req := ssl.NewDescribeCertificatesRequest()
	req.SearchKey = common.StringPtr(aliasPrefix)
	req.Limit = common.Uint64Ptr(100)
	req.CertificateStatus = common.Uint64Ptrs([]uint64{3}) // 3 = expired

	resp, err := t.sslClient.DescribeCertificates(req)
	if err != nil {
		log.Printf("WARNING: failed to list certificates for cleanup: %v", err)
		return
	}
	if resp.Response == nil {
		return
	}

	for _, cert := range resp.Response.Certificates {
		if cert.CertificateId == nil || cert.Alias == nil {
			continue
		}
		certID := *cert.CertificateId
		alias := *cert.Alias
		// Only clean up certs that match our alias prefix and are not the current one.
		if certID == currentCertID || !strings.HasPrefix(alias, aliasPrefix) {
			continue
		}
		delReq := ssl.NewDeleteCertificateRequest()
		delReq.CertificateId = common.StringPtr(certID)
		_, err := t.sslClient.DeleteCertificate(delReq)
		if err != nil {
			log.Printf("WARNING: failed to delete expired certificate %s (%s): %v", certID, alias, err)
		} else {
			log.Printf("cleaned up expired certificate %s (%s)", certID, alias)
		}
	}
}

func (t *TencentCloudTEO) deployCert(certID string) (string, error) {
	req := teo.NewModifyHostsCertificateRequest()
	req.ZoneId = common.StringPtr(t.zoneID)
	req.Hosts = common.StringPtrs(t.hosts)
	req.Mode = common.StringPtr("sslcert")
	req.ServerCertInfo = []*teo.ServerCertInfo{
		{
			CertId: common.StringPtr(certID),
		},
	}

	log.Printf("deploying cert %s to zone %s, hosts=%v", certID, t.zoneID, t.hosts)

	resp, err := t.teoClient.ModifyHostsCertificate(req)
	if err != nil {
		return "", err
	}

	if resp.Response == nil || resp.Response.RequestId == nil {
		return "", nil
	}

	return *resp.Response.RequestId, nil
}
