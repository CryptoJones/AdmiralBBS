package menu

import "testing"

func TestSysopPassOK(t *testing.T) {
	if !sysopPassOK("c0ckbl0cked", "c0ckbl0cked") {
		t.Error("matching password should pass")
	}
	if sysopPassOK("wrong", "c0ckbl0cked") {
		t.Error("non-matching password should fail")
	}
	if sysopPassOK("", "c0ckbl0cked") {
		t.Error("empty entry should fail when a password is configured")
	}
	if sysopPassOK("c0ckbl0cked", "c0ckbl0ckeX") {
		t.Error("near-miss should fail")
	}
}
