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
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	accessKeyEnv = consts.AppCapsName + "_ACCESS_KEY"
	secretKeyEnv = consts.AppCapsName + "_SECRET_KEY"
)

// Credential represents access and secret key pairs for authentication.
type Credential struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

// ToSecretData converts to kubernetes secret data.
func (cred Credential) ToSecretData() map[string][]byte {
	return map[string][]byte{
		"accessKey": []byte(cred.AccessKey),
		"secretKey": []byte(cred.SecretKey),
	}
}

func getCredentialFromEnv() (cred *Credential, err error) {
	if accessKey, found := os.LookupEnv(accessKeyEnv); found {
		if secretKey, found := os.LookupEnv(secretKeyEnv); found {
			return &Credential{
				AccessKey: accessKey,
				SecretKey: secretKey,
			}, nil
		}
	}

	return nil, fmt.Errorf("credential not set in env vars")
}

func getCredentialFromConfig(configFile string) (*Credential, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cred Credential
	if err = json.NewDecoder(file).Decode(&cred); err != nil {
		return nil, err
	}
	return &cred, nil
}

func getCredentialFromSecrets(ctx context.Context) (*Credential, error) {
	secrets, err := k8s.KubeClient().CoreV1().Secrets(consts.Namespace).Get(ctx, consts.CredentialsSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	accessKey, found := secrets.Data["accessKey"]
	if !found {
		return nil, fmt.Errorf("access key not found in secrets")
	}

	secretKey, found := secrets.Data["secretKey"]
	if !found {
		return nil, fmt.Errorf("secret key not found in secrets")
	}

	return &Credential{
		AccessKey: string(accessKey),
		SecretKey: string(secretKey),
	}, nil
}

// GetCredential fetches from environment variable, configFile or kubernetes secrets.
func GetCredential(ctx context.Context, configFile string) (*Credential, error) {
	if cred, err := getCredentialFromEnv(); err == nil {
		return cred, nil
	}

	if cred, err := getCredentialFromConfig(configFile); err == nil {
		return cred, nil
	}

	return getCredentialFromSecrets(ctx)
}
