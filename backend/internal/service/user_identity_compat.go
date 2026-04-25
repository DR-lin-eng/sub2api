package service

import (
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrIdentityProviderInvalid = infraerrors.BadRequest("IDENTITY_PROVIDER_INVALID", "identity provider is invalid")

type UserAuthIdentityRecord struct {
	ProviderType    string
	ProviderKey     string
	ProviderSubject string
	VerifiedAt      *time.Time
	Issuer          *string
	Metadata        map[string]any
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
