// This file is part of MinIO Direct CSI
// Copyright (c) 2020 MinIO, Inc.
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

package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	scheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/minio/direct-csi/pkg/utils"
)

func registerCRDs(ctx context.Context) error {
	crdObjs := []runtime.Object{}
	for _, asset := range AssetNames() {
		crdBytes, err := Asset(asset)
		if err != nil {
			return err
		}
		crdObj, err := utils.ParseSingleKubeNativeFromBytes(crdBytes)
		if err != nil {
			return err
		}
		crdObjs = append(crdObjs, crdObj)
	}

	crdClient := utils.GetAPIExtensionsClient()
	for _, crd := range crdObjs {
		res := &apiextensions.CustomResourceDefinition{}
		if err := crdClient.RESTClient().
			Post().
			Resource("customresourcedefinitions").
			VersionedParams(&metav1.CreateOptions{}, scheme.ParameterCodec).
			Body(crd).
			Do(ctx).
			Into(res); err != nil {
			return err
		}
	}
	return nil
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

	crdClient := utils.GetCRDClient()
	for _, crd := range crdNames {
		if err := crdClient.Delete(ctx, crd, metav1.DeleteOptions{}); err != nil {
			return err
		}
	}

	return nil
}
