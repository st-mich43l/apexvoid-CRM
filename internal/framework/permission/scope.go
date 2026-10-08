package permission

type Scope string

const (
	ScopePlatform  Scope = "platform"
	ScopeWorkspace Scope = "workspace"
)

func (s Scope) Valid() bool { return s == ScopePlatform || s == ScopeWorkspace }
