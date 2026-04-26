package ctxkeys

// CtxKey ctx key struct.
type CtxKey struct {
	Name string
}

// String CtxKey string.
func (c CtxKey) String() string {
	return c.Name
}

var (
	// XRequestID request_id
	XRequestID = CtxKey{"x-request-id"}

	// ClientIP  client_ip
	ClientIP = CtxKey{"client_ip"}

	// RequestMethod request method
	RequestMethod = CtxKey{"request_method"}

	// RequestURI request uri
	RequestURI = CtxKey{"request_uri"}

	// UserAgent request ua
	UserAgent = CtxKey{"request_ua"}

	// TimeLocal time local
	TimeLocal = CtxKey{"time_local"}

	// CurHostname current hostname
	CurHostname = CtxKey{"hostname"}

	// FullStack full stack
	FullStack = CtxKey{"full_stack"}
)
