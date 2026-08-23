package client

import (
	"context"
	"net/http"
	"testing"
)

// TestDatabaseDecodesIntegerLimits locks in a regression risk found while
// wiring limits_memory_swappiness/limits_cpu_shares: unlike every other
// limits_* field, the standalone-database tables store these two as genuine
// integer columns (confirmed against the migrations), so the API sends them
// as JSON numbers, not strings. Decoding a JSON number into a Go string
// field errors out the whole response, breaking every database read — not
// just these two attributes — so this must stay int64.
func TestDatabaseDecodesIntegerLimits(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"uuid":"db1","limits_memory_swappiness":60,"limits_cpu_shares":1024}`))
	}))
	db, err := c.GetDatabase(context.Background(), "db1")
	if err != nil {
		t.Fatalf("GetDatabase: %v", err)
	}
	if db.LimitsMemorySwappiness != 60 {
		t.Errorf("LimitsMemorySwappiness = %d, want 60", db.LimitsMemorySwappiness)
	}
	if db.LimitsCPUShares != 1024 {
		t.Errorf("LimitsCPUShares = %d, want 1024", db.LimitsCPUShares)
	}
}
