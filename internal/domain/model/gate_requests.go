package model

// Search and pagination parameters.
type ListFilter struct {
	Page   int
	Size   int
	Types  []GateType
	Status GateStatus
	Q      string
}

// Meta-specific requests.
type CreateMetaApp struct {
	Name             string
	URI              string
	AppID            string
	AppSecret        string
	OAuthRedirectURI string
	Scopes           []string
	VerifyToken      string
}

type OAuthStart struct {
	MetaAppID   string
	ExtraScopes []string
}

type OAuthCallback struct {
	MetaAppID string
	Code      string
	State     string
}

type CreateFacebook struct {
	Name      string
	Dc        int64
	MetaAppID string
	PageID    string
	PageToken string
	Peer      Peer // Combined Sub and Iss for identity
	Enabled   bool
}

type CreateWhatsApp struct {
	Name          string
	MetaAppID     string
	WABAID        string
	PhoneNumberID string
	AccessToken   string
}

type UpdateMetaApp struct {
	ID               string
	Name             *string
	AppSecret        *string
	OAuthRedirectURI *string
	Scopes           []string
	VerifyToken      *string
}

type UpdateFacebook struct {
	ID        string
	Name      *string
	PageToken *string
	Enabled   *bool
	Peer      *Peer // Optional update for bot identity
}

type UpdateWhatsApp struct {
	ID          string
	Name        *string
	AccessToken *string
}
