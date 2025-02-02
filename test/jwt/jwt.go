/*
Copyright 2021 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

package jwt

import (
	"crypto"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"gopkg.in/square/go-jose.v2"
)

var HmacSampleSecret []byte

const (
	dummySigningKeyFile = "../testdata/dummyRsaPrivateKey.pem"
	dummySigningKeyType = "RSA PRIVATE KEY"

	tokenLookup = "Authorization"
	authScheme  = "Bearer"
	dummyOid    = "00000000-0000-0000-0000-000000000000"
	expiredDate = "2021-03-11T00:00:00Z"
	validDate   = "2099-03-11T00:00:00Z"
)

var (
	clientID  = os.Getenv("OIDC_CLIENT_ID")
	issuerURL = os.Getenv("OIDC_ISSUER_URL")
)

type Claim struct {
	Key   string
	Value string
}

// BuildAuthHeader builds the authorization header with a JWT bearer token
func BuildAuthHeader(expiredToken bool, signingKeyFile string, signingKeyType string, c Claim) string {
	signedToken := GenerateSignedToken(expiredToken, signingKeyFile, signingKeyType, c)
	return authScheme + " " + signedToken
}

func GenerateDefaultSignedToken() string {
	return GenerateSignedToken(false, "", "", Claim{})
}

// GenerateSignedToken generates and sign a jwt token
func GenerateSignedToken(expiredToken bool, signingKeyFile string, signingKeyType string, c Claim) string {

	if signingKeyFile == "" {
		signingKeyFile = dummySigningKeyFile
	}

	if signingKeyType == "" {
		signingKeyType = dummySigningKeyType
	}

	if c.Key == "" {
		aud := os.Getenv("OIDC_CLIENT_ID")
		c = Claim{Key: "aud", Value: aud}
	}

	dt := newDummyToken(signingKeyFile, signingKeyType)

	if expiredToken == true {
		expiration, _ := time.Parse(time.RFC3339Nano, expiredDate)
		dt.setExpiration(expiration)
	}

	if c.Key != "" {
		dt.setClaim(c)
	}

	signedToken := dt.sign()

	return signedToken
}

// GetSigningKey converts rsaPrivateKey into a private/public JSONWebKey
func GetSigningKey(signingKeyFile string, rsaKeyType string) *jose.JSONWebKey {
	var key *jose.JSONWebKey

	rsaPrivateKey, err := ioutil.ReadFile(signingKeyFile)
	if err != nil {
		panic("Failed to read file " + signingKeyFile)
	}

	block, _ := pem.Decode([]byte(rsaPrivateKey))
	if block == nil {
		panic("Failed to decode pem.")
	}

	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		panic("Failed to parse the key: " + err.Error())
	}

	if rsaKeyType == dummySigningKeyType {
		key = &jose.JSONWebKey{Key: rsaKey, Use: "sig", Algorithm: string(jose.RS256)}
	} else {
		key = &jose.JSONWebKey{Key: rsaKey.Public(), Use: "sig", Algorithm: string(jose.RS256)}
	}

	thumbprint, err := key.Thumbprint(crypto.SHA256)
	if err != nil {
		panic("Failed to compute thumbprint:" + err.Error())
	}

	key.KeyID = hex.EncodeToString(thumbprint)
	return key
}

// dummyToken represent the token claims
type dummyToken struct {
	claims         map[string]string
	signingKeyFile string
	signingKeyType string
}

// newDummyToken
func newDummyToken(signingKeyFile string, signingKeyType string) *dummyToken {
	claims := make(map[string]string)
	claims["exp"] = fmt.Sprint(time.Now().Add(1 * time.Hour).Unix())
	claims["iat"] = fmt.Sprint(time.Now().Unix())
	claims["iss"] = issuerURL
	claims["ipd"] = issuerURL
	claims["aud"] = clientID
	claims["oid"] = dummyOid

	return &dummyToken{
		claims:         claims,
		signingKeyFile: signingKeyFile,
		signingKeyType: signingKeyType,
	}
}

// setExpiration sets the token expiration
func (t *dummyToken) setExpiration(tm time.Time) {
	t.claims["exp"] = fmt.Sprint(tm.Unix())
}

// setClaim sets a token claim
func (t *dummyToken) setClaim(c Claim) {
	t.claims[c.Key] = c.Value
}

// SignToken
func (t *dummyToken) sign() string {
	signingKey := GetSigningKey(t.signingKeyFile, t.signingKeyType)
	signer, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.RS256,
		Key:       signingKey,
	}, nil)
	if err != nil {
		panic(err.Error())
	}

	claimString, err := json.Marshal(t.claims)
	if err != nil {
		panic(err.Error())
	}

	signedToken, err := signer.Sign([]byte(claimString))
	if err != nil {
		panic(err.Error())
	}

	serializedToken, err := signedToken.CompactSerialize()
	if err != nil {
		panic(err.Error())
	}
	return serializedToken
}
