// This file is part of MinIO DirectPV
// Copyright (c) 2021, 2022 MinIO, Inc.
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
	"context"
	"net/http"
	"time"

	"github.com/minio/directpv/pkg/credential"
)

// authMiddleware serves as a middleware handler to terminate un-authorized requests
func authMiddleware(fn func(rw http.ResponseWriter, rq *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cred, err := credential.LoadFromSecret(context.Background())
		if err != nil {
			writeErrorResponse(w, http.StatusNotFound, toAPIError(err, "couldn't load the credential from secret"))
			return
		}
		if err := doesSignatureMatch(r, cred); err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, toAPIError(err, "couldn't verify the signature"))
			return
		}
		// successfully authenticated
		fn(w, r)
	}
}

// doesSignatureMatch verifies the s3v4 signature in the request by comparing the self-calculated s3v4 signature
func doesSignatureMatch(r *http.Request, cred credential.Credential) error {
	// Copy request.
	req := *r

	// Save authorization header.
	v4Auth := req.Header.Get(authorizationHeader)

	// Parse signature version '4' header.
	signV4Values, err := parseSignV4(v4Auth)
	if err != nil {
		return err
	}

	// Extract all the signed headers along with its values.
	extractedSignedHeaders, err := extractSignedHeaders(signV4Values.SignedHeaders, r)
	if err != nil {
		return err
	}

	if signV4Values.Credential.accessKey != cred.AccessKey {
		return errWrongAccessKey
	}

	// Extract date, if not present throw error.
	var date string
	if date = req.Header.Get(amzDateHeaderKey); date == "" {
		if date = r.Header.Get(dateHeaderKey); date == "" {
			return errMissingDateHeader
		}
	}

	// Parse date header.
	t, e := time.Parse(iso8601Format, date)
	if e != nil {
		return errMalformedDate
	}

	// Query string.
	queryStr := req.Form.Encode()

	// Get canonical request.
	canonicalRequest := getCanonicalRequest(extractedSignedHeaders, queryStr, req.URL.Path, req.Method)

	// Get string to sign from canonical request.
	stringToSign := getStringToSign(canonicalRequest, t, signV4Values.Credential.getScope())

	// Get hmac signing key.
	signingKey := getSigningKey(cred.SecretKey, signV4Values.Credential.scope.date,
		signV4Values.Credential.scope.region, "s3")

	// Calculate signature.
	newSignature := getSignature(signingKey, stringToSign)

	// Verify if signature match.
	if !compareSignatureV4(newSignature, signV4Values.Signature) {
		return errSignatureDoesNotMatch
	}

	// Return error none.
	return nil
}
