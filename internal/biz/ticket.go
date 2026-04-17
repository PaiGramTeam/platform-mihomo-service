package biz

type ServiceTicketClaims struct {
	ActorType            string
	ActorID              string
	OwnerUserID          uint64
	BindingID            uint64
	Platform             string
	PlatformAccountID    string
	Consumer             string
	ProfileID            uint64
	Scopes               []string
	Audience             string
	BotID                string
	UserID               uint64
	PlatformServiceKey   string
	PlatformAccountRefID uint64

	// PlatformAccountRefID is a read-only legacy alias for BindingID so
	// downstream callers can migrate incrementally. New tickets should use
	// BindingID; if a token still carries platform_account_ref_id, verifier
	// code requires it to match binding_id.
}
