package entity

// Scope describes the ownership boundary a future repository must enforce for
// records of an entity. Global is the default for platform metadata and
// identity; workspace records must always receive an explicit workspace scope.
type Scope string

const (
	ScopeGlobal       Scope = "global"
	ScopeOrganization Scope = "organization"
	ScopeWorkspace    Scope = "workspace"
)

func (s Scope) Valid() bool {
	return s == ScopeGlobal || s == ScopeOrganization || s == ScopeWorkspace
}
