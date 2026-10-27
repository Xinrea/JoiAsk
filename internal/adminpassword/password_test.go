package adminpassword

import (
	"crypto/sha256"
	"encoding/hex"
	"joiask-backend/pkg/util"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashAndVerify(t *testing.T) {
	const password = "correct-password"
	first, err := Hash(password)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Hash(password)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("identical passwords must receive independent random salts")
	}
	if cost, err := bcrypt.Cost([]byte(first)); err != nil || cost != bcrypt.DefaultCost {
		t.Fatalf("unexpected bcrypt cost: %d, %v", cost, err)
	}
	for _, tc := range []struct {
		stored, input    string
		matches, migrate bool
	}{
		{first, password, true, false},
		{first, "wrong-password", false, false},
		{first, first, false, false},
		{first, util.Md5v(password), false, false},
		{util.Md5v(password), password, true, true},
		{strings.ToUpper(util.Md5v(password)), password, true, true},
		{util.Md5v(password), util.Md5v(password), false, false},
		{util.Md5v(password), "wrong-password", false, false},
		{"invalid", "invalid", false, false},
		{strings.Repeat("z", 32), strings.Repeat("z", 32), false, false},
		{util.Md5v(""), "", false, false},
	} {
		if matches, migrate := Verify(tc.stored, tc.input); matches != tc.matches || migrate != tc.migrate {
			t.Errorf("Verify(%q, input) = %t, %t; want %t, %t", tc.stored, matches, migrate, tc.matches, tc.migrate)
		}
	}
}

func TestPasswordLength(t *testing.T) {
	if _, err := Hash(""); err != ErrEmptyPassword {
		t.Errorf("Hash with empty password: %v", err)
	}
	if matches, _ := Verify(util.Md5v(""), ""); matches {
		t.Fatal("empty password accepted")
	}
	for _, password := range []string{strings.Repeat("a", 72), strings.Repeat("密", 24)} {
		hash, err := Hash(password)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(hash, sha256Prefix) || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
			t.Fatal("exactly 72 bytes must use untagged, direct bcrypt")
		}
		if matches, _ := Verify(hash, password); !matches {
			t.Fatal("72-byte password rejected")
		}
		if matches, _ := Verify(hash, password+"a"); matches {
			t.Fatal("bcrypt truncation must not accept a longer password")
		}
	}
}

func TestLongPasswordHashAndVerify(t *testing.T) {
	for _, password := range []string{strings.Repeat("a", 73), strings.Repeat("密", 25)} {
		t.Run(password, func(t *testing.T) {
			hash, err := Hash(password)
			if err != nil {
				t.Fatal(err)
			}
			second, err := Hash(password)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(hash, sha256Prefix) || hash == second {
				t.Fatal("long passwords require a versioned format and random salts")
			}
			innerHash := strings.TrimPrefix(hash, sha256Prefix)
			digest := sha256.Sum256([]byte(password))
			prehash := hex.EncodeToString(digest[:])
			if cost, err := bcrypt.Cost([]byte(innerHash)); err != nil || cost != bcrypt.DefaultCost {
				t.Fatalf("unexpected bcrypt cost: %d, %v", cost, err)
			}
			if err := bcrypt.CompareHashAndPassword([]byte(innerHash), []byte(prehash)); err != nil {
				t.Fatalf("long password must use bcrypt(hex(SHA256(password))): %v", err)
			}
			for _, tc := range []struct {
				stored, input    string
				matches, migrate bool
			}{
				{hash, password, true, false},
				{second, password, true, false},
				{util.Md5v(password), password, true, true},
				{hash, password[:72] + "different-tail", false, false},
				{hash, password[:72], false, false},
				{hash, password + "extra", false, false},
				{hash, "", false, false},
				{hash, util.Md5v(password), false, false},
				{hash, prehash, false, false},
				{hash, hash, false, false},
				{hash, innerHash, false, false},
				{innerHash, password, false, false},
				{strings.Replace(hash, "$v1$", "$v2$", 1), password, false, false},
				{sha256Prefix, password, false, false},
				{sha256Prefix + "invalid", password, false, false},
			} {
				if matches, migrate := Verify(tc.stored, tc.input); matches != tc.matches || migrate != tc.migrate {
					t.Errorf("Verify(%q, input) = %t, %t; want %t, %t", tc.stored, matches, migrate, tc.matches, tc.migrate)
				}
			}
		})
	}
}
