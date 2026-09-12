package systemversion

import (
	"errors"
	"testing"
)

func TestCanManageFailsClosed(t *testing.T) {
	if !CanManage(true, nil, nil) {
		t.Fatal("root must be allowed")
	}
	if CanManage(false, []string{"admin"}, nil) {
		t.Fatal("role name is not authority")
	}
	enforce := func(args ...interface{}) (bool, error) {
		return args[0] == "operators" && args[1] == "/api/v1/system/updates/check" && args[2] == "POST", nil
	}
	if !CanManage(false, []string{"viewer", "operators"}, enforce) {
		t.Fatal("delegated user denied")
	}
	if CanManage(false, []string{"viewer"}, enforce) {
		t.Fatal("viewer allowed")
	}
	if CanManage(false, []string{"operators"}, func(...interface{}) (bool, error) {
		return true, errors.New("policy unavailable")
	}) {
		t.Fatal("enforcement error must deny")
	}
}
