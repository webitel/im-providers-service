package model

type OAuthStart struct {
	MetaAppID   string
	ExtraScopes []string
}

type OAuthCallback struct {
	MetaAppID string
	Code      string
	State     string
}

// LinkedPage carries the minimal page info returned after a successful OAuth flow.
type LinkedPage struct {
	PageID    string
	PageName  string
	PageToken string
}
