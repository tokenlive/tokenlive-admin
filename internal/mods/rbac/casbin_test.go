package rbac

import "testing"

func TestUninitializedEnforcerIsUnavailable(t *testing.T) {
	for _, value := range []*Casbinx{nil, {}} {
		if got := value.GetEnforcer(); got != nil {
			t.Fatalf("uninitialized enforcer = %v", got)
		}
	}
}
