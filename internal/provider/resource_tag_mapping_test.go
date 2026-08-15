package provider

import (
	"testing"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

// TestFindTagIsCaseInsensitive locks in the reason tag_name is never adopted
// from the API into state (unlike a typical Computed field): Coolify
// normalizes tag names to lowercase server-side (trim + lowercase), so an
// exact-case match against a mixed-case config would never find the tag
// Coolify just told us it attached — and adopting the lowercased value would
// hit the same "provider produced inconsistent result" class of bug this
// session already fixed five times elsewhere (Required, not Computed).
func TestFindTagIsCaseInsensitive(t *testing.T) {
	tags := []client.Tag{
		{UUID: "t1", Name: "production"},
		{UUID: "t2", Name: "staging"},
	}

	got := findTag(tags, "Production")
	if got == nil || got.UUID != "t1" {
		t.Errorf("findTag(%q) = %v, want tag t1", "Production", got)
	}

	got = findTag(tags, "  STAGING  ")
	if got == nil || got.UUID != "t2" {
		t.Errorf("findTag(%q) = %v, want tag t2", "  STAGING  ", got)
	}

	if findTag(tags, "nonexistent") != nil {
		t.Error("findTag(nonexistent) should be nil")
	}
}
