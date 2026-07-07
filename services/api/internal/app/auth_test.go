package app

import "testing"

func TestFallbackLoginDemoAccounts(t *testing.T) {
	auth, ok := fallbackLogin("user_teacher_001", "Teacher@123456")
	if !ok {
		t.Fatal("teacher demo login should succeed")
	}
	if auth.UserID != "user_teacher_001" || auth.TeacherID != "teacher_001" {
		t.Fatalf("unexpected auth context: %+v", auth)
	}
	headers := authHeaders(auth)
	if headers["X-User-ID"] != "user_teacher_001" || headers["X-Teacher-ID"] != "teacher_001" {
		t.Fatalf("unexpected auth headers: %+v", headers)
	}
}

func TestFallbackLoginDemoAccountAliases(t *testing.T) {
	auth, ok := fallbackLogin("13800000001", "Teacher@123456")
	if !ok {
		t.Fatal("teacher mobile demo login should succeed")
	}
	if auth.UserID != "user_teacher_001" || auth.TeacherID != "teacher_001" {
		t.Fatalf("unexpected auth context: %+v", auth)
	}
	if _, ok := fallbackLogin("teacher@example.local", "bad-password"); ok {
		t.Fatal("alias login must still validate password")
	}
}

func TestFallbackLoginRejectsBadPassword(t *testing.T) {
	if _, ok := fallbackLogin("user_teacher_001", "bad-password"); ok {
		t.Fatal("bad password should be rejected")
	}
}

func TestDecodeHeaderValue(t *testing.T) {
	if got := decodeHeaderValue("%E4%BB%BB%E8%AF%BE%E6%95%99%E5%B8%88"); got != "任课教师" {
		t.Fatalf("expected decoded role, got %q", got)
	}
	if got := decodeHeaderValue("teacher_001"); got != "teacher_001" {
		t.Fatalf("expected ascii header unchanged, got %q", got)
	}
}

func TestHashDemoPassword(t *testing.T) {
	if got := hashDemoPassword("Teacher@123456"); len(got) != 64 {
		t.Fatalf("expected sha256 hex hash, got %q", got)
	}
}
