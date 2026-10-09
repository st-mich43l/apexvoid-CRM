package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/st-mich43l/apexvoid-CRM/integration"
)

func TestManifestAuthenticityDoesNotLeakEnrollmentCode(t *testing.T) {
	const code = "local-pairing-secret-long-random-value-123456789"
	body := []byte(`{"manifest_version":"v1"}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-ApexVoid-Enrollment-Code") != "" || r.Header.Get("Authorization") != "" {
			t.Error("discovery transmitted the pairing secret over HTTP")
		}
		mac := hmac.New(sha256.New, []byte(code))
		_, _ = mac.Write([]byte(manifestContext))
		_, _ = mac.Write(body)
		w.Header().Set(manifestSignatureHeader, "sha256="+hex.EncodeToString(mac.Sum(nil)))
		_, _ = w.Write(body)
	}))
	defer server.Close()
	data, err := authenticatedManifest(context.Background(), server.Client(), server.URL, code)
	if err != nil || string(data) != string(body) {
		t.Fatalf("manifest proof failed: %s, %v", data, err)
	}
	if _, err = authenticatedManifest(context.Background(), server.Client(), server.URL, "different-random-setup-secret-for-testing"); err == nil {
		t.Fatal("mismatched enrollment proof was accepted")
	}
}

func TestEnrollmentEncryptionAndTampering(t *testing.T) {
	const code = "local-pairing-secret-long-random-value-123456789"
	clear := []byte(`{"service_credential":"secret","database":{"password":"private"}}`)
	payload, err := sealEnrollment(code, clear)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "private") || strings.Contains(string(payload), code) {
		t.Fatal("enrollment envelope leaks cleartext secrets")
	}
	decrypted, err := unsealEnrollment(code, payload)
	if err != nil || string(decrypted) != string(clear) {
		t.Fatalf("decrypt failed: %v", err)
	}
	if _, err = unsealEnrollment("different-random-setup-secret-for-testing", payload); err == nil {
		t.Fatal("wrong key decrypted envelope")
	}
	var envelope sealedEnrollment
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope.Ciphertext = envelope.Ciphertext[:len(envelope.Ciphertext)-2] + "AA"
	tampered, _ := json.Marshal(envelope)
	if _, err = unsealEnrollment(code, tampered); err == nil {
		t.Fatal("modified enrollment ciphertext accepted")
	}
	if _, err = sealEnrollment("short", clear); err == nil {
		t.Fatal("weak enrollment code accepted")
	}
}

func TestProvisioningEncryptionAndCanonicalOwnership(t *testing.T) {
	const key = "independent-database-provisioning-encryption-key-2026"
	cipher, err := protectProvisioningPassword(key, "long-random-role-secret")
	if err != nil {
		t.Fatal(err)
	}
	clear, err := recoverProvisioningPassword(key, cipher)
	if err != nil || clear != "long-random-role-secret" {
		t.Fatal("database password recovery failed", err)
	}
	if _, err = recoverProvisioningPassword("different-provisioning-encryption-key-2026", cipher); err == nil {
		t.Fatal("wrong master key accepted")
	}
	db, schema, role := canonicalApplicationDatabase("cafe")
	if db != "apexvoid_cafe" || schema != "cafe" || role != "apexvoid_cafe" {
		t.Fatalf("unexpected canonical ownership: %s/%s/%s", db, schema, role)
	}
}

func TestEnrollmentServiceProvesCredentialReceipt(t *testing.T) {
	const code = "local-pairing-secret-long-random-value-123456789"
	credential := strings.Repeat("c", 64)
	clear := []byte(`{"application_id":"cafe","api_contract_version":"v1","service_credential":"`+credential+`","database":{"name":"apexvoid_cafe","schema":"cafe","role":"apexvoid_cafe","password":"`+strings.Repeat("p",40)+`","migration_bundle_version":"0.1.0"}}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		if r.Header.Get("X-ApexVoid-Enrollment-Code") != "" { t.Error("enrollment leaked one-time code in HTTP headers") }
		body,err:=io.ReadAll(r.Body)
		if err!=nil{t.Error(err);w.WriteHeader(400);return}
		values,err:=integration.OpenEnrollment(code,body)
		if err!=nil || values.ServiceCredential!=credential {t.Error("invalid authenticated service enrollment",err);w.WriteHeader(400);return}
		proof,err:=integration.SignEnrollmentAcknowledgement(values.ServiceCredential,r.Header.Get("X-ApexVoid-Enrollment-Challenge"))
		if err!=nil{t.Error(err);w.WriteHeader(400);return}
		w.Header().Set("X-ApexVoid-Enrollment-Ack",proof)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	if err:=postEnrollment(context.Background(),server.Client(),server.URL,code,credential,clear);err!=nil{t.Fatal(err)}
	if err:=postEnrollment(context.Background(),server.Client(),server.URL,code,strings.Repeat("z",64),clear);err==nil{t.Fatal("incorrect credential acknowledgment passed")}
}
