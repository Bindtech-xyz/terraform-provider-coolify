package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestStringOrNil(t *testing.T) {
	if stringOrNil(types.StringNull()) != nil {
		t.Error("null → nil expected")
	}
	if stringOrNil(types.StringUnknown()) != nil {
		t.Error("unknown → nil expected")
	}
	if v := stringOrNil(types.StringValue("x")); v == nil || *v != "x" {
		t.Errorf("value → %v", v)
	}
}

func TestInt64AndBoolOrNil(t *testing.T) {
	if int64OrNil(types.Int64Null()) != nil || boolOrNil(types.BoolNull()) != nil {
		t.Error("null → nil expected")
	}
	if v := int64OrNil(types.Int64Value(42)); v == nil || *v != 42 {
		t.Errorf("int64 value → %v", v)
	}
	if v := boolOrNil(types.BoolValue(true)); v == nil || !*v {
		t.Errorf("bool value → %v", v)
	}
}

// keepNullIfEmpty prevents the permanent diff caused by Coolify normalising
// absent strings to "".
func TestKeepNullIfEmpty(t *testing.T) {
	if got := keepNullIfEmpty("", types.StringNull()); !got.IsNull() {
		t.Error("empty API + null prior must stay null")
	}
	if got := keepNullIfEmpty("", types.StringValue("was-set")); got.ValueString() != "" {
		t.Error("empty API + configured prior must adopt empty (practitioner cleared it)")
	}
	if got := keepNullIfEmpty("live", types.StringNull()); got.ValueString() != "live" {
		t.Error("API value must win when present")
	}
}

// keepPriorIfHidden covers sensitive fields hidden from tokens without the
// read:sensitive ability.
func TestKeepPriorIfHidden(t *testing.T) {
	if got := keepPriorIfHidden("", types.StringValue("secret")); got.ValueString() != "secret" {
		t.Error("hidden API value must keep the configured secret")
	}
	if got := keepPriorIfHidden("rotated", types.StringValue("secret")); got.ValueString() != "rotated" {
		t.Error("visible API value must win")
	}
	if got := keepPriorIfHidden("", types.StringNull()); !got.IsNull() && got.ValueString() != "" {
		t.Errorf("hidden + never configured = %v", got)
	}
}
