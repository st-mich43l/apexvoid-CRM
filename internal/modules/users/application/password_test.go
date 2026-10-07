package application

import "testing"

func TestPasswordHasher(t *testing.T) {
	hasher := PasswordHasher{MinLength: 12, MaxLength: 128}
	hash, err := hasher.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "correct horse battery staple" {
		t.Fatal("password was not hashed")
	}
	if !hasher.Verify(hash, "correct horse battery staple") {
		t.Fatal("hashed password did not verify")
	}
	if hasher.Verify(hash, "wrong password") {
		t.Fatal("wrong password verified")
	}
}

func TestPasswordHasherPolicy(t *testing.T) {
	hasher := PasswordHasher{MinLength: 12, MaxLength: 128}
	if _, err := hasher.Hash("too-short"); err == nil {
		t.Fatal("expected short password to fail policy")
	}
}

func TestPasswordHasherBootstrapCredentialIsHashed(t *testing.T) {
	hasher := PasswordHasher{MinLength: 12, MaxLength: 128}
	hash, err := hasher.HashBootstrap("admin")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "admin" || !hasher.Verify(hash, "admin") {
		t.Fatal("bootstrap password was not hashed correctly")
	}
}
