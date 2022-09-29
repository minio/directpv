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

package credential

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"

	"github.com/minio/directpv/pkg/consts"
	"github.com/minio/directpv/pkg/k8s"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/klog/v2"
)

var (
	errSecretKeyEnvNotSet = errors.New(consts.SecretKeyEnv + " environment variable is not set")
	errAccessKeyEnvNotSet = errors.New(consts.AccessKeyEnv + " environment variable is not set")
	errEnvNotFound        = errors.New("no environment variable settings for credentials found")
	errSecretNotFound     = errors.New("credential secret " + consts.CredentialsSecretName + " not found")
	errAccessKeyNotFound  = errors.New("accessKey not found in the secret")
	errSecretKeyNotFound  = errors.New("secretKey not found in the secret")
)

// Credential represents the access and secret key pairs for authentication
type Credential struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

// Load loads the credential from the ENV. If ENV settings are not present, it loads from the config file provided
func Load(configFile string) (Credential, error) {
	cred, err := loadFromEnv()
	if err != nil {
		if err != errEnvNotFound {
			return cred, err
		}
	} else {
		return cred, err
	}
	return loadFromFile(configFile)
}

// LoadFromSecret loads the credential from the k8s secret
func LoadFromSecret(ctx context.Context) (cred Credential, err error) {
	credSecret, err := k8s.KubeClient().CoreV1().Secrets(consts.Namespace).Get(context.Background(), consts.CredentialsSecretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = errSecretNotFound
			return
		}
		return cred, err
	}

	accessKey, ok := credSecret.Data[consts.AccessKeyDataKey]
	if !ok {
		return cred, errAccessKeyNotFound
	}
	cred.AccessKey = string(accessKey)

	secretKey, ok := credSecret.Data[consts.SecretKeyDataKey]
	if !ok {
		return cred, errSecretKeyNotFound
	}
	cred.SecretKey = string(secretKey)

	return cred, nil
}

// loadFromEnv loads the credential from ENVs
func loadFromEnv() (cred Credential, err error) {
	accessKey, accessKeySet := os.LookupEnv(consts.AccessKeyEnv)
	secretKey, secretKeySet := os.LookupEnv(consts.SecretKeyEnv)

	switch {
	case accessKeySet && secretKeySet:
		cred.AccessKey = accessKey
		cred.SecretKey = secretKey
	case accessKeySet && !secretKeySet:
		err = errSecretKeyEnvNotSet
	case secretKeySet && !accessKeySet:
		err = errAccessKeyEnvNotSet
	case !accessKeySet && !secretKeySet:
		err = errEnvNotFound
	}

	return
}

// loadFromFile loads the credential from provided config file
func loadFromFile(configFile string) (Credential, error) {
	cred := Credential{}
	fileData, err := ioutil.ReadFile(configFile)
	if err != nil {
		return cred, err
	}
	err = json.Unmarshal(fileData, &cred)
	if err != nil {
		klog.Infof("\n unable to parse the config file %s: %v", configFile, err)
	}
	return cred, err
}
