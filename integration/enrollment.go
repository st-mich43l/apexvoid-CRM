package integration

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
)

const ManifestSignatureHeader = "X-ApexVoid-Manifest-Signature"
const manifestHMACContext = "apexvoid-manifest-v1\n"
const enrollmentAEADContext = "apexvoid-enrollment-v1"

// SignManifest produces the exact response-header signature required by
// ApexVoid Enterprise during bootstrap discovery. The setup code remains
// local to the service and is NEVER returned in the HTTP response.
func SignManifest(setupCode string, manifest []byte) (string,error) {
	if len(setupCode)<32 {return "",errors.New("a high-entropy setup code of at least 32 characters is required")}
	h:=hmac.New(sha256.New,[]byte(setupCode))
	_,_=h.Write([]byte(manifestHMACContext))
	_,_=h.Write(manifest)
	return "sha256="+hex.EncodeToString(h.Sum(nil)),nil
}

type EnrollmentDatabase struct {
	Name string `json:"name"`
	Schema string `json:"schema"`
	Role string `json:"role"`
	Password string `json:"password"`
	MigrationBundleVersion string `json:"migration_bundle_version"`
}
type EnrollmentCredentials struct {
	ApplicationID string `json:"application_id"`
	ServiceCredential string `json:"service_credential"`
	APIContractVersion string `json:"api_contract_version"`
	Database EnrollmentDatabase `json:"database"`
}
type enrollmentEnvelope struct {
	Version string `json:"version"`
	Nonce string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// OpenEnrollment decrypts a restricted, authenticated one-time POST from
// Enterprise. Services MUST validate application identity, expiry, setup-code
// consumption and credential persistence before returning success.
func OpenEnrollment(setupCode string, data []byte) (EnrollmentCredentials,error) {
	if len(setupCode)<32{return EnrollmentCredentials{},errors.New("setup code is too short")}
	var envelope enrollmentEnvelope
	decoder:=json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err:=decoder.Decode(&envelope);err!=nil||envelope.Version!="v1"{return EnrollmentCredentials{},errors.New("invalid enrollment envelope")}
	var trailing any
	if err:=decoder.Decode(&trailing);err!=io.EOF {
		return EnrollmentCredentials{},errors.New("unexpected additional enrollment payload")
	}
	nonce,err:=base64.RawURLEncoding.DecodeString(envelope.Nonce)
	if err!=nil{return EnrollmentCredentials{},err}
	ciphertext,err:=base64.RawURLEncoding.DecodeString(envelope.Ciphertext)
	if err!=nil{return EnrollmentCredentials{},err}
	key:=sha256.Sum256([]byte(enrollmentAEADContext+":"+setupCode))
	block,err:=aes.NewCipher(key[:]);if err!=nil{return EnrollmentCredentials{},err}
	gcm,err:=cipher.NewGCM(block);if err!=nil{return EnrollmentCredentials{},err}
	if len(nonce)!=gcm.NonceSize(){return EnrollmentCredentials{},errors.New("enrollment nonce is invalid")}
	clear,err:=gcm.Open(nil,nonce,ciphertext,[]byte(enrollmentAEADContext))
	if err!=nil{return EnrollmentCredentials{},errors.New("enrollment authentication failed")}
	var credentials EnrollmentCredentials
	decode:=json.NewDecoder(bytes.NewReader(clear))
	decode.DisallowUnknownFields()
	if err=decode.Decode(&credentials);err!=nil{return EnrollmentCredentials{},errors.New("invalid enrollment credentials")}
	if credentials.ApplicationID==""||len(credentials.ServiceCredential)<32||credentials.APIContractVersion!="v1"||
		credentials.Database.Name==""||credentials.Database.Schema==""||credentials.Database.Role==""||len(credentials.Database.Password)<32 {
		return EnrollmentCredentials{},errors.New("enrollment credentials are incomplete")
	}
	return credentials,nil
}
