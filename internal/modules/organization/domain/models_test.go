package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestSlugifyProducesStableHumanFriendlySlugs(t *testing.T) {
	if got := Slugify("ApexVoid Technologies / Vietnam Sales"); got != "apexvoid-technologies-vietnam-sales" {
		t.Fatalf("unexpected slug %q", got)
	}
}

func TestWorkspaceRequiresTimezoneAndValidStatus(t *testing.T) {
	workspace := Workspace{OrganizationID: uuid.New(), Name: "Sales", Slug: "sales", Status: StatusActive}
	if err := workspace.Validate(); err == nil {
		t.Fatal("expected timezone validation error")
	}
}
