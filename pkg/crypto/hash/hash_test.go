package hash

import (
	"testing"
)

func TestGeneratePassword(t *testing.T) {
	origin := "admin"
	hashPwd, err := GeneratePassword(origin)
	if err != nil {
		t.Error("GeneratePassword Failed: ", err.Error())
	}
	t.Log("test password: ", hashPwd, ",length: ", len(hashPwd))

	if err := CompareHashAndPassword(hashPwd, origin); err != nil {
		t.Error("Unmatched password: ", err.Error())
	}
}

func TestMD5(t *testing.T) {
	origin := "admin"
	hashVal := "21232f297a57a5a743894a0e4a801fc3"
	if v := MD5String(origin); v != hashVal {
		t.Error("Failed to generate MD5 hash: ", v)
	}
}

func TestHMACSHA256StringMatchesGatewayAPIKeyHash(t *testing.T) {
	got := HMACSHA256String("tl_live_example", "pepper")
	want := "06bfbed9282f1dcb96bd25c7bef96d9b49de0be5f3777b44f4f71cfcca8821b1"
	if got != want {
		t.Fatalf("HMACSHA256String() = %q, want %q", got, want)
	}
}
