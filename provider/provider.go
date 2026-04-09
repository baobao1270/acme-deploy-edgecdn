package provider

import "context"

type CertData struct {
	Domain     string
	FullChain  string // PEM content (cert + intermediates)
	PrivateKey string // PEM content
}

type DeployResult struct {
	CertID    string
	RequestID string
}

type Provider interface {
	Deploy(ctx context.Context, cert *CertData) (*DeployResult, error)
}
