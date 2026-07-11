package services

import "testing"

// TestSetDefaultGroupName: newly discovered stocks land in the configured
// group; the setter must override the default.
func TestSetDefaultGroupName(t *testing.T) {
	svc := NewKisAccountSyncService(nil, nil, nil, nil, nil, "")
	svc.SetDefaultGroupName("Toss 자동동기화")
	if svc.defaultGroupName != "Toss 자동동기화" {
		t.Errorf("defaultGroupName = %q", svc.defaultGroupName)
	}
}
