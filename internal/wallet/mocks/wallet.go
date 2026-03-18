package mocks

import (
	"context"

	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
)

// MockWallet implements sdkWallet.Interface for testing purposes.
type MockWallet struct {
	sdkWallet.Interface
	ListOutputsFunc                  func(ctx context.Context, args sdkWallet.ListOutputsArgs, originator string) (*sdkWallet.ListOutputsResult, error)
	CreateActionFunc                 func(ctx context.Context, args sdkWallet.CreateActionArgs, originator string) (*sdkWallet.CreateActionResult, error)
	SignActionFunc                   func(ctx context.Context, args sdkWallet.SignActionArgs, originator string) (*sdkWallet.SignActionResult, error)
	GetPublicKeyFunc                 func(ctx context.Context, args sdkWallet.GetPublicKeyArgs, originator string) (*sdkWallet.GetPublicKeyResult, error)
	DecryptFunc                      func(ctx context.Context, args sdkWallet.DecryptArgs, originator string) (*sdkWallet.DecryptResult, error)
	CreateHMACFunc                   func(ctx context.Context, args sdkWallet.CreateHMACArgs, originator string) (*sdkWallet.CreateHMACResult, error)
	VerifyHMACFunc                   func(ctx context.Context, args sdkWallet.VerifyHMACArgs, originator string) (*sdkWallet.VerifyHMACResult, error)
	EncryptFunc                      func(ctx context.Context, args sdkWallet.EncryptArgs, originator string) (*sdkWallet.EncryptResult, error)
	CreateSignatureFunc              func(ctx context.Context, args sdkWallet.CreateSignatureArgs, originator string) (*sdkWallet.CreateSignatureResult, error)
	VerifySignatureFunc              func(ctx context.Context, args sdkWallet.VerifySignatureArgs, originator string) (*sdkWallet.VerifySignatureResult, error)
	AcquireCertificateFunc           func(ctx context.Context, args sdkWallet.AcquireCertificateArgs, originator string) (*sdkWallet.Certificate, error)
	ListCertificatesFunc             func(ctx context.Context, args sdkWallet.ListCertificatesArgs, originator string) (*sdkWallet.ListCertificatesResult, error)
	ProveCertificateFunc             func(ctx context.Context, args sdkWallet.ProveCertificateArgs, originator string) (*sdkWallet.ProveCertificateResult, error)
	RelinquishCertificateFunc        func(ctx context.Context, args sdkWallet.RelinquishCertificateArgs, originator string) (*sdkWallet.RelinquishCertificateResult, error)
	RelinquishOutputFunc             func(ctx context.Context, args sdkWallet.RelinquishOutputArgs, originator string) (*sdkWallet.RelinquishOutputResult, error)
	AbortActionFunc                  func(ctx context.Context, args sdkWallet.AbortActionArgs, originator string) (*sdkWallet.AbortActionResult, error)
	ListActionsFunc                  func(ctx context.Context, args sdkWallet.ListActionsArgs, originator string) (*sdkWallet.ListActionsResult, error)
	InternalizeActionFunc            func(ctx context.Context, args sdkWallet.InternalizeActionArgs, originator string) (*sdkWallet.InternalizeActionResult, error)
	RevealCounterpartyKeyLinkageFunc func(ctx context.Context, args sdkWallet.RevealCounterpartyKeyLinkageArgs, originator string) (*sdkWallet.RevealCounterpartyKeyLinkageResult, error)
	RevealSpecificKeyLinkageFunc     func(ctx context.Context, args sdkWallet.RevealSpecificKeyLinkageArgs, originator string) (*sdkWallet.RevealSpecificKeyLinkageResult, error)
	DiscoverByIdentityKeyFunc        func(ctx context.Context, args sdkWallet.DiscoverByIdentityKeyArgs, originator string) (*sdkWallet.DiscoverCertificatesResult, error)
	DiscoverByAttributesFunc         func(ctx context.Context, args sdkWallet.DiscoverByAttributesArgs, originator string) (*sdkWallet.DiscoverCertificatesResult, error)
	IsAuthenticatedFunc              func(ctx context.Context, args any, originator string) (*sdkWallet.AuthenticatedResult, error)
	WaitForAuthenticationFunc        func(ctx context.Context, args any, originator string) (*sdkWallet.AuthenticatedResult, error)
	GetHeightFunc                    func(ctx context.Context, args any, originator string) (*sdkWallet.GetHeightResult, error)
	GetHeaderForHeightFunc           func(ctx context.Context, args sdkWallet.GetHeaderArgs, originator string) (*sdkWallet.GetHeaderResult, error)
	GetNetworkFunc                   func(ctx context.Context, args any, originator string) (*sdkWallet.GetNetworkResult, error)
	GetVersionFunc                   func(ctx context.Context, args any, originator string) (*sdkWallet.GetVersionResult, error)
}

func (m *MockWallet) ListOutputs(ctx context.Context, args sdkWallet.ListOutputsArgs, originator string) (*sdkWallet.ListOutputsResult, error) {
	if m.ListOutputsFunc != nil {
		return m.ListOutputsFunc(ctx, args, originator)
	}
	return &sdkWallet.ListOutputsResult{}, nil
}

func (m *MockWallet) CreateAction(ctx context.Context, args sdkWallet.CreateActionArgs, originator string) (*sdkWallet.CreateActionResult, error) {
	if m.CreateActionFunc != nil {
		return m.CreateActionFunc(ctx, args, originator)
	}
	return &sdkWallet.CreateActionResult{}, nil
}

func (m *MockWallet) SignAction(ctx context.Context, args sdkWallet.SignActionArgs, originator string) (*sdkWallet.SignActionResult, error) {
	if m.SignActionFunc != nil {
		return m.SignActionFunc(ctx, args, originator)
	}
	return &sdkWallet.SignActionResult{}, nil
}

func (m *MockWallet) GetPublicKey(ctx context.Context, args sdkWallet.GetPublicKeyArgs, originator string) (*sdkWallet.GetPublicKeyResult, error) {
	if m.GetPublicKeyFunc != nil {
		return m.GetPublicKeyFunc(ctx, args, originator)
	}
	return &sdkWallet.GetPublicKeyResult{}, nil
}

func (m *MockWallet) Decrypt(ctx context.Context, args sdkWallet.DecryptArgs, originator string) (*sdkWallet.DecryptResult, error) {
	if m.DecryptFunc != nil {
		return m.DecryptFunc(ctx, args, originator)
	}
	return &sdkWallet.DecryptResult{}, nil
}

func (m *MockWallet) CreateHMAC(ctx context.Context, args sdkWallet.CreateHMACArgs, originator string) (*sdkWallet.CreateHMACResult, error) {
	if m.CreateHMACFunc != nil {
		return m.CreateHMACFunc(ctx, args, originator)
	}
	return &sdkWallet.CreateHMACResult{}, nil
}

func (m *MockWallet) VerifyHMAC(ctx context.Context, args sdkWallet.VerifyHMACArgs, originator string) (*sdkWallet.VerifyHMACResult, error) {
	if m.VerifyHMACFunc != nil {
		return m.VerifyHMACFunc(ctx, args, originator)
	}
	return &sdkWallet.VerifyHMACResult{}, nil
}

func (m *MockWallet) Encrypt(ctx context.Context, args sdkWallet.EncryptArgs, originator string) (*sdkWallet.EncryptResult, error) {
	if m.EncryptFunc != nil {
		return m.EncryptFunc(ctx, args, originator)
	}
	return &sdkWallet.EncryptResult{}, nil
}

func (m *MockWallet) CreateSignature(ctx context.Context, args sdkWallet.CreateSignatureArgs, originator string) (*sdkWallet.CreateSignatureResult, error) {
	if m.CreateSignatureFunc != nil {
		return m.CreateSignatureFunc(ctx, args, originator)
	}
	return &sdkWallet.CreateSignatureResult{}, nil
}

func (m *MockWallet) VerifySignature(ctx context.Context, args sdkWallet.VerifySignatureArgs, originator string) (*sdkWallet.VerifySignatureResult, error) {
	if m.VerifySignatureFunc != nil {
		return m.VerifySignatureFunc(ctx, args, originator)
	}
	return &sdkWallet.VerifySignatureResult{}, nil
}

func (m *MockWallet) AcquireCertificate(ctx context.Context, args sdkWallet.AcquireCertificateArgs, originator string) (*sdkWallet.Certificate, error) {
	if m.AcquireCertificateFunc != nil {
		return m.AcquireCertificateFunc(ctx, args, originator)
	}
	return &sdkWallet.Certificate{}, nil
}

func (m *MockWallet) ListCertificates(ctx context.Context, args sdkWallet.ListCertificatesArgs, originator string) (*sdkWallet.ListCertificatesResult, error) {
	if m.ListCertificatesFunc != nil {
		return m.ListCertificatesFunc(ctx, args, originator)
	}
	return &sdkWallet.ListCertificatesResult{}, nil
}

func (m *MockWallet) ProveCertificate(ctx context.Context, args sdkWallet.ProveCertificateArgs, originator string) (*sdkWallet.ProveCertificateResult, error) {
	if m.ProveCertificateFunc != nil {
		return m.ProveCertificateFunc(ctx, args, originator)
	}
	return &sdkWallet.ProveCertificateResult{}, nil
}

func (m *MockWallet) RelinquishCertificate(ctx context.Context, args sdkWallet.RelinquishCertificateArgs, originator string) (*sdkWallet.RelinquishCertificateResult, error) {
	if m.RelinquishCertificateFunc != nil {
		return m.RelinquishCertificateFunc(ctx, args, originator)
	}
	return &sdkWallet.RelinquishCertificateResult{}, nil
}

func (m *MockWallet) RelinquishOutput(ctx context.Context, args sdkWallet.RelinquishOutputArgs, originator string) (*sdkWallet.RelinquishOutputResult, error) {
	if m.RelinquishOutputFunc != nil {
		return m.RelinquishOutputFunc(ctx, args, originator)
	}
	return &sdkWallet.RelinquishOutputResult{}, nil
}

func (m *MockWallet) AbortAction(ctx context.Context, args sdkWallet.AbortActionArgs, originator string) (*sdkWallet.AbortActionResult, error) {
	if m.AbortActionFunc != nil {
		return m.AbortActionFunc(ctx, args, originator)
	}
	return &sdkWallet.AbortActionResult{}, nil
}

func (m *MockWallet) ListActions(ctx context.Context, args sdkWallet.ListActionsArgs, originator string) (*sdkWallet.ListActionsResult, error) {
	if m.ListActionsFunc != nil {
		return m.ListActionsFunc(ctx, args, originator)
	}
	return &sdkWallet.ListActionsResult{}, nil
}

func (m *MockWallet) InternalizeAction(ctx context.Context, args sdkWallet.InternalizeActionArgs, originator string) (*sdkWallet.InternalizeActionResult, error) {
	if m.InternalizeActionFunc != nil {
		return m.InternalizeActionFunc(ctx, args, originator)
	}
	return &sdkWallet.InternalizeActionResult{}, nil
}

func (m *MockWallet) RevealCounterpartyKeyLinkage(ctx context.Context, args sdkWallet.RevealCounterpartyKeyLinkageArgs, originator string) (*sdkWallet.RevealCounterpartyKeyLinkageResult, error) {
	if m.RevealCounterpartyKeyLinkageFunc != nil {
		return m.RevealCounterpartyKeyLinkageFunc(ctx, args, originator)
	}
	return &sdkWallet.RevealCounterpartyKeyLinkageResult{}, nil
}

func (m *MockWallet) RevealSpecificKeyLinkage(ctx context.Context, args sdkWallet.RevealSpecificKeyLinkageArgs, originator string) (*sdkWallet.RevealSpecificKeyLinkageResult, error) {
	if m.RevealSpecificKeyLinkageFunc != nil {
		return m.RevealSpecificKeyLinkageFunc(ctx, args, originator)
	}
	return &sdkWallet.RevealSpecificKeyLinkageResult{}, nil
}

func (m *MockWallet) DiscoverByIdentityKey(ctx context.Context, args sdkWallet.DiscoverByIdentityKeyArgs, originator string) (*sdkWallet.DiscoverCertificatesResult, error) {
	if m.DiscoverByIdentityKeyFunc != nil {
		return m.DiscoverByIdentityKeyFunc(ctx, args, originator)
	}
	return &sdkWallet.DiscoverCertificatesResult{}, nil
}

func (m *MockWallet) DiscoverByAttributes(ctx context.Context, args sdkWallet.DiscoverByAttributesArgs, originator string) (*sdkWallet.DiscoverCertificatesResult, error) {
	if m.DiscoverByAttributesFunc != nil {
		return m.DiscoverByAttributesFunc(ctx, args, originator)
	}
	return &sdkWallet.DiscoverCertificatesResult{}, nil
}

func (m *MockWallet) IsAuthenticated(ctx context.Context, args any, originator string) (*sdkWallet.AuthenticatedResult, error) {
	if m.IsAuthenticatedFunc != nil {
		return m.IsAuthenticatedFunc(ctx, args, originator)
	}
	return &sdkWallet.AuthenticatedResult{}, nil
}

func (m *MockWallet) WaitForAuthentication(ctx context.Context, args any, originator string) (*sdkWallet.AuthenticatedResult, error) {
	if m.WaitForAuthenticationFunc != nil {
		return m.WaitForAuthenticationFunc(ctx, args, originator)
	}
	return &sdkWallet.AuthenticatedResult{}, nil
}

func (m *MockWallet) GetHeight(ctx context.Context, args any, originator string) (*sdkWallet.GetHeightResult, error) {
	if m.GetHeightFunc != nil {
		return m.GetHeightFunc(ctx, args, originator)
	}
	return &sdkWallet.GetHeightResult{}, nil
}

func (m *MockWallet) GetHeaderForHeight(ctx context.Context, args sdkWallet.GetHeaderArgs, originator string) (*sdkWallet.GetHeaderResult, error) {
	if m.GetHeaderForHeightFunc != nil {
		return m.GetHeaderForHeightFunc(ctx, args, originator)
	}
	return &sdkWallet.GetHeaderResult{}, nil
}

func (m *MockWallet) GetNetwork(ctx context.Context, args any, originator string) (*sdkWallet.GetNetworkResult, error) {
	if m.GetNetworkFunc != nil {
		return m.GetNetworkFunc(ctx, args, originator)
	}
	return &sdkWallet.GetNetworkResult{}, nil
}

func (m *MockWallet) GetVersion(ctx context.Context, args any, originator string) (*sdkWallet.GetVersionResult, error) {
	if m.GetVersionFunc != nil {
		return m.GetVersionFunc(ctx, args, originator)
	}
	return &sdkWallet.GetVersionResult{}, nil
}
