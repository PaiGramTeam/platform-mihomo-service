package biz

type ServiceTicketClaims struct {
	ActorType            string
	ActorID              string
	OwnerUserID          uint64
	BotID                string
	UserID               uint64
	Platform             string
	PlatformServiceKey   string
	PlatformAccountID    string
	PlatformAccountRefID uint64
	Scopes               []string
	Audience             string
}
