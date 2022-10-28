// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/minio/directpv/pkg/utils"
)

const (
	signerDateLayout = "20060102"
	iso8601UTCLayout = "20060102T150405Z"
)

var multiSpaceRegex = regexp.MustCompile("( +)")

func sha256Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func sumHMAC(key, data []byte) []byte {
	hash := hmac.New(sha256.New, key)
	hash.Write(data)
	return hash.Sum(nil)
}

func getScope(serviceName, region string, date time.Time) string {
	return date.Format(signerDateLayout) + "/" + region + "/" + serviceName + "/aws4_request"
}

func getCanonicalHeaders(headers http.Header, includeAcceptEncoding, includeUserAgent bool) (string, string) {
	signedHeaders := []string{}
	m := map[string]string{}
	for key, values := range headers {
		key = strings.ToLower(key)
		if key == "authorization" {
			continue
		}

		if key == "accept-encoding" && !includeAcceptEncoding {
			continue
		}

		if key == "user-agent" && !includeUserAgent {
			continue
		}

		signedHeaders = append(signedHeaders, key)

		var canonicalValues []string
		for _, value := range values {
			canonicalValues = append(canonicalValues, multiSpaceRegex.ReplaceAllString(value, " "))
		}
		m[key] = strings.Join(canonicalValues, ",")
	}

	sort.Strings(signedHeaders)

	var canonicalHeaders []string
	for _, key := range signedHeaders {
		canonicalHeaders = append(canonicalHeaders, key+":"+m[key])
	}

	return strings.Join(canonicalHeaders, "\n"), strings.Join(signedHeaders, ";")
}

func getCanonicalQueryString(query url.Values) string {
	var keys []string
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var canonicalValues []string
	for _, key := range keys {
		for _, value := range query[key] {
			canonicalValues = append(canonicalValues, key+"="+value)
		}
	}
	return strings.Join(canonicalValues, "&")
}

func getCanonicalRequestHash(canonicalHeaders, signedHeaders, canonicalQueryString, method, escapedPath, contentSha256 string) string {
	// CanonicalRequest =
	//   HTTPRequestMethod + '\n' +
	//   CanonicalURI + '\n' +
	//   CanonicalQueryString + '\n' +
	//   CanonicalHeaders + '\n' +
	//   SignedHeaders + '\n' +
	//   HexEncode(Hash(RequestPayload))

	canonicalRequest := strings.Join(
		[]string{
			method,
			escapedPath,
			canonicalQueryString,
			canonicalHeaders + "\n",
			signedHeaders,
			contentSha256,
		},
		"\n",
	)

	return sha256Hash([]byte(canonicalRequest))
}

func getStringToSign(date time.Time, scope, canonicalRequestHash string) string {
	return strings.Join(
		[]string{
			"AWS4-HMAC-SHA256",
			date.Format(iso8601UTCLayout),
			scope,
			canonicalRequestHash,
		},
		"\n",
	)
}

func getSigningKey(serviceName, secretKey, region string, date time.Time) []byte {
	aws4SecretKey := []byte("AWS4" + secretKey)
	dateKey := sumHMAC(aws4SecretKey, []byte(date.Format(signerDateLayout)))
	dateRegionKey := sumHMAC(dateKey, []byte(region))
	dateRegionServiceKey := sumHMAC(dateRegionKey, []byte(serviceName))
	return sumHMAC(dateRegionServiceKey, []byte("aws4_request"))
}

func getSignature(signingKey []byte, stringToSign string) string {
	hash := sumHMAC(signingKey, []byte(stringToSign))
	return strings.ToLower(hex.EncodeToString(hash))
}

func getAuthorization(accessKey, scope, signedHeaders, signature string) string {
	return "AWS4-HMAC-SHA256 Credential=" + accessKey + "/" + scope + ", SignedHeaders=" + signedHeaders + ", Signature=" + signature
}

func signV4(serviceName string, headers http.Header, query url.Values, escapedPath, method, region, accessKey, secretKey, contentSha256 string, date time.Time) string {
	scope := getScope(serviceName, region, date)
	canonicalHeaders, signedHeaders := getCanonicalHeaders(headers, false, false)
	canonicalQueryString := getCanonicalQueryString(query)
	canonicalRequestHash := getCanonicalRequestHash(canonicalHeaders, signedHeaders, canonicalQueryString, method, escapedPath, contentSha256)
	stringToSign := getStringToSign(date, scope, canonicalRequestHash)
	signingKey := getSigningKey(serviceName, secretKey, region, date)
	signature := getSignature(signingKey, stringToSign)
	return getAuthorization(accessKey, scope, signedHeaders, signature)
}

func signV4CSI(headers http.Header, escapedPath string, cred *Credential, contentSha256 string, date time.Time) string {
	return signV4("CSI", headers, nil, escapedPath, http.MethodPost, "", cred.AccessKey, cred.SecretKey, contentSha256, date)
}

func parseAuthorization(headers http.Header) (passedAccessKey, passedScope, passedSignedHeaders, passedSignature string, err error) {
	auth, err := getHeaderValue(headers, "Authorization")
	if err != nil {
		return
	}

	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 ") {
		err = fmt.Errorf("invalid authorization")
		return
	}

	tokens := strings.Split(strings.TrimPrefix(auth, "AWS4-HMAC-SHA256 "), ", ")
	if len(tokens) != 3 {
		err = fmt.Errorf("invalid authorization")
		return
	}

	if !strings.HasPrefix(tokens[0], "Credential=") {
		err = fmt.Errorf("invalid authorization")
		return
	}
	values := strings.SplitN(strings.TrimPrefix(tokens[0], "Credential="), "/", 2)
	if len(values) != 2 {
		err = fmt.Errorf("invalid authorization")
		return
	}
	passedAccessKey = values[0]
	if passedAccessKey == "" {
		err = fmt.Errorf("invalid authorization")
		return
	}
	passedScope = values[1]
	if passedScope == "" {
		err = fmt.Errorf("invalid authorization")
		return
	}

	if !strings.HasPrefix(tokens[1], "SignedHeaders=") {
		err = fmt.Errorf("invalid authorization")
		return
	}
	passedSignedHeaders = strings.TrimPrefix(tokens[1], "SignedHeaders=")
	if passedSignedHeaders == "" {
		err = fmt.Errorf("invalid authorization")
		return
	}

	if !strings.HasPrefix(tokens[2], "Signature=") {
		err = fmt.Errorf("invalid authorization")
		return
	}
	passedSignature = strings.TrimPrefix(tokens[2], "Signature=")
	if passedSignature == "" {
		err = fmt.Errorf("invalid authorization")
	}

	return
}

func parseXAmzDate(headers http.Header) (date time.Time, err error) {
	value, err := getHeaderValue(headers, "x-amz-date")
	if err != nil {
		return
	}
	return time.Parse(iso8601UTCLayout, value)
}

func checkSignV4(serviceName string, headers http.Header, query url.Values, escapedPath, method, region, accessKey, secretKey, contentSha256 string) error {
	passedAccessKey, passedScope, passedSignedHeaders, passedSignature, err := parseAuthorization(headers)
	if err != nil {
		return err
	}

	if accessKey != passedAccessKey {
		return fmt.Errorf("invalid access key")
	}

	passedSignedHeaderKeys := strings.Split(passedSignedHeaders, ";")
	if !utils.Contains(passedSignedHeaderKeys, "x-amz-date") {
		return fmt.Errorf("x-amz-date header must be signed")
	}
	if !utils.Contains(passedSignedHeaderKeys, "host") {
		return fmt.Errorf("host header must be signed")
	}

	date, err := parseXAmzDate(headers)
	if err != nil {
		return err
	}

	utcNow := time.Now().UTC()
	if utcNow.Before(date) {
		return fmt.Errorf("request time too far")
	}
	if utcNow.Sub(date).Seconds() > 3 {
		return fmt.Errorf("request time too far")
	}

	if _, err := getHeaderValue(headers, "host"); err != nil {
		return err
	}

	scope := getScope(serviceName, region, date)
	if scope != passedScope {
		return fmt.Errorf("invalid scope")
	}

	canonicalHeaders, signedHeaders := getCanonicalHeaders(
		headers,
		utils.Contains(passedSignedHeaderKeys, "accept-encoding"),
		utils.Contains(passedSignedHeaderKeys, "user-agent"),
	)
	canonicalQueryString := getCanonicalQueryString(query)
	canonicalRequestHash := getCanonicalRequestHash(canonicalHeaders, signedHeaders, canonicalQueryString, method, escapedPath, contentSha256)
	if signedHeaders != passedSignedHeaders {
		return fmt.Errorf("all headers must be signed")
	}

	stringToSign := getStringToSign(date, passedScope, canonicalRequestHash)
	signingKey := getSigningKey(serviceName, secretKey, region, date)
	signature := getSignature(signingKey, stringToSign)
	if signature != passedSignature {
		return fmt.Errorf("signature does not match")
	}

	return nil
}

func checkSignV4CSI(headers http.Header, escapedPath string, cred *Credential, contentSha256 string) error {
	return checkSignV4("CSI", headers, nil, escapedPath, http.MethodPost, "", cred.AccessKey, cred.SecretKey, contentSha256)
}
