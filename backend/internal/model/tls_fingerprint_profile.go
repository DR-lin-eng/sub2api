package model

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

// TLSFingerprintProfile models a reusable TLS ClientHello profile.
type TLSFingerprintProfile struct {
	ID                  int64     `json:"id"`
	Name                string    `json:"name"`
	Description         *string   `json:"description"`
	EnableGREASE        bool      `json:"enable_grease"`
	CipherSuites        []uint16  `json:"cipher_suites"`
	Curves              []uint16  `json:"curves"`
	PointFormats        []uint16  `json:"point_formats"`
	SignatureAlgorithms []uint16  `json:"signature_algorithms"`
	ALPNProtocols       []string  `json:"alpn_protocols"`
	SupportedVersions   []uint16  `json:"supported_versions"`
	KeyShareGroups      []uint16  `json:"key_share_groups"`
	PSKModes            []uint16  `json:"psk_modes"`
	Extensions          []uint16  `json:"extensions"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (p *TLSFingerprintProfile) Validate() error {
	if p == nil || p.Name == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	return nil
}

func (p *TLSFingerprintProfile) ToTLSProfile() *tlsfingerprint.Profile {
	if p == nil {
		return nil
	}
	pointFormats := make([]uint8, 0, len(p.PointFormats))
	for _, value := range p.PointFormats {
		pointFormats = append(pointFormats, uint8(value))
	}
	return &tlsfingerprint.Profile{
		Name:         p.Name,
		EnableGREASE: p.EnableGREASE,
		CipherSuites: p.CipherSuites,
		Curves:       p.Curves,
		PointFormats: pointFormats,
	}
}
