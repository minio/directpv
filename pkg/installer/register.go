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

package installer

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	directcsi "github.com/minio/directpv/pkg/apis/direct.csi.min.io/v1beta4"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/converter"
	"github.com/minio/directpv/pkg/utils"

	"k8s.io/apiextensions-apiserver/pkg/apihelpers"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

var (
	errEmptyCABundle = errors.New("CA bundle is empty")
)

func parseSingleKubeNativeFromBytes(data []byte) (runtime.Object, error) {
	obj := map[string]interface{}{}
	err := yaml.Unmarshal(data, &obj)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{
		Object: obj,
	}, nil
}

func registerCRDs(ctx context.Context, c *Config) error {
	crdObjs := []runtime.Object{}
	for _, asset := range AssetNames() {
		crdBytes, err := Asset(asset)
		if err != nil {
			return err
		}
		crdObj, err := parseSingleKubeNativeFromBytes(crdBytes)
		if err != nil {
			return err
		}
		crdObjs = append(crdObjs, crdObj)
	}

	crdClient := client.GetCRDClient()
	for _, crd := range crdObjs {
		var crdObj apiextensions.CustomResourceDefinition
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(crd.(*unstructured.Unstructured).Object, &crdObj); err != nil {
			return err
		}

		existingCRD, err := crdClient.Get(ctx, crdObj.Name, metav1.GetOptions{})
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				return err
			}

			if err := setConversionWebhook(ctx, &crdObj, c); err != nil {
				return err
			}

			if c.DryRun {
				utils.UpdateLabels(&crdObj, map[utils.LabelKey]utils.LabelValue{utils.VersionLabelKey: directcsi.Version})
			} else {
				if _, err := crdClient.Create(ctx, &crdObj, metav1.CreateOptions{}); err != nil {
					return err
				}
			}

			if err := c.postProc(crdObj); err != nil {
				return err
			}
			continue
		}
		if err := syncCRD(ctx, existingCRD, crdObj, c); err != nil {
			return err
		}
	}
	return nil
}

func syncCRD(ctx context.Context, existingCRD *apiextensions.CustomResourceDefinition, newCRD apiextensions.CustomResourceDefinition, c *Config) error {
	existingCRDStorageVersion, err := apihelpers.GetCRDStorageVersion(existingCRD)
	if err != nil {
		return err
	}

	var versionEntryFound bool
	if existingCRDStorageVersion != directcsi.Version {
		// Set all the existing versions to false
		func() {
			for i := range existingCRD.Spec.Versions {
				if existingCRD.Spec.Versions[i].Name != directcsi.Version {
					existingCRD.Spec.Versions[i].Storage = false
				} else {
					existingCRD.Spec.Versions[i].Storage = true
					versionEntryFound = true
				}
			}
		}()

		if !versionEntryFound {
			latestVersionObject, err := getLatestCRDVersionObject(newCRD)
			if err != nil {
				return err
			}
			existingCRD.Spec.Versions = append(existingCRD.Spec.Versions, latestVersionObject)
		}
	}

	if err := setConversionWebhook(ctx, existingCRD, c); err != nil {
		return err
	}

	if c.DryRun {
		utils.UpdateLabels(existingCRD, map[utils.LabelKey]utils.LabelValue{utils.VersionLabelKey: directcsi.Version})
		existingCRD.TypeMeta = newCRD.TypeMeta
	} else {
		crdClient := client.GetCRDClient()
		if _, err := crdClient.Update(ctx, existingCRD, metav1.UpdateOptions{}); err != nil {
			return err
		}
		klog.V(5).Infof("'%s' CRD successfully updated to '%s'", existingCRD.Name, utils.Bold(directcsi.Version))
	}

	return c.postProc(existingCRD)
}

func setConversionWebhook(ctx context.Context, crdObj *apiextensions.CustomResourceDefinition, c *Config) error {

	getServiceRef := func() *apiextensions.ServiceReference {
		path := func() string {
			switch crdObj.Name {
			case driveCRDName:
				return converter.DriveHandlerPath
			case volumeCRDName:
				return converter.VolumeHandlerPath
			default:
				panic("unknown crd name found")
			}
		}()
		port := int32(conversionWebhookPort)

		return &apiextensions.ServiceReference{
			Namespace: c.namespace(),
			Name:      c.serviceName(),
			Path:      &path,
			Port:      &port,
		}
	}

	getWebhookClientConfig := func() (*apiextensions.WebhookClientConfig, error) {
		caBundle, err := GetConversionCABundle(ctx, c)
		if err != nil {
			return nil, err
		}
		return &apiextensions.WebhookClientConfig{
			Service:  getServiceRef(),
			CABundle: []byte(caBundle),
		}, nil
	}

	getWebhookConversionSettings := func() (*apiextensions.WebhookConversion, error) {
		webhookClientConfig, err := getWebhookClientConfig()
		if err != nil {
			return nil, err
		}
		return &apiextensions.WebhookConversion{
			ClientConfig:             webhookClientConfig,
			ConversionReviewVersions: []string{"v1"},
		}, nil
	}

	getConversionSettings := func() (*apiextensions.CustomResourceConversion, error) {
		webhookConversionSettings, err := getWebhookConversionSettings()
		if err != nil {
			return nil, err
		}
		return &apiextensions.CustomResourceConversion{
			Strategy: apiextensions.WebhookConverter,
			Webhook:  webhookConversionSettings,
		}, nil
	}

	conversionSettings, err := getConversionSettings()
	if err != nil {
		return err
	}

	crdObj.Spec.Conversion = conversionSettings
	return nil
}

func GetConversionCABundle(ctx context.Context, c *Config) ([]byte, error) {
	getCABundleFromConfig := func() ([]byte, error) {
		conversionCABundle := c.conversionWebhookCaBundle
		if len(conversionCABundle) == 0 {
			return []byte{}, errEmptyCABundle
		}
		return conversionCABundle, nil
	}

	secret, err := client.GetKubeClient().
		CoreV1().
		Secrets(c.namespace()).
		Get(ctx, conversionCACert, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) && c.DryRun {
			return getCABundleFromConfig()
		}
		return []byte{}, err
	}

	for key, value := range secret.Data {
		if key == caCertFileName {
			return value, nil
		}
	}

	return []byte{}, errEmptyCABundle
}

func getLatestCRDVersionObject(newCRD apiextensions.CustomResourceDefinition) (apiextensions.CustomResourceDefinitionVersion, error) {
	for i := range newCRD.Spec.Versions {
		if newCRD.Spec.Versions[i].Name == directcsi.Version {
			return newCRD.Spec.Versions[i], nil
		}
	}

	return apiextensions.CustomResourceDefinitionVersion{}, fmt.Errorf("no version %v foung crd %v", directcsi.Version, newCRD.Name)
}

func unregisterCRDs(ctx context.Context) error {
	crdNames := []string{}
	for _, asset := range AssetNames() {
		base := filepath.Base(asset)
		baseWithoutExtension := strings.Split(base, ".yaml")[0]
		parts := strings.Split(baseWithoutExtension, "_")
		if len(parts) != 2 {
			return fmt.Errorf("invalid crd name: %v", baseWithoutExtension)
		}
		crd := parts[1]
		apiGroup := parts[0]
		crdNames = append(crdNames, fmt.Sprintf("%s.%s", crd, apiGroup))
	}

	crdClient := client.GetCRDClient()
	for _, crd := range crdNames {
		if err := crdClient.Delete(ctx, crd, metav1.DeleteOptions{}); err != nil {
			if !k8serrors.IsNotFound(err) {
				return err
			}
		}
	}

	return nil
}
