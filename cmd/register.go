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

package cmd

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-tools/pkg/crd"
	crdmarkers "sigs.k8s.io/controller-tools/pkg/crd/markers"
	"sigs.k8s.io/controller-tools/pkg/genall"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/markers"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/golang/glog"
	"github.com/minio/direct-csi/pkg/utils"
)

func registerCRDs(ctx context.Context) error {
	roots, err := loader.LoadRoots("github.com/minio/direct-csi/pkg/apis/direct.csi.min.io/v1alpha1")
	if err != nil {
		return err
	}

	defn := markers.Must(markers.MakeDefinition("crd", markers.DescribesPackage, crd.Generator{}))
	optionsRegistry := &markers.Registry{}
	if err := optionsRegistry.Register(defn); err != nil {
		return err
	}
	if err := genall.RegisterOptionsMarkers(optionsRegistry); err != nil {
		return err
	}
	if err := crdmarkers.Register(optionsRegistry); err != nil {
		return err
	}
	parser := &crd.Parser{
		Collector: &markers.Collector{
			Registry: optionsRegistry,
		},
		Checker: &loader.TypeChecker{},
	}
	crd.AddKnownTypes(parser)
	for _, root := range roots {
		parser.NeedPackage(root)
	}

	metav1Pkg := crd.FindMetav1(roots)
	if metav1Pkg == nil {
		// no objects in the roots, since nothing imported metav1
		return fmt.Errorf("no objects found in all roots")
	}

	// TODO: allow selecting a specific object
	kubeKinds := crd.FindKubeKinds(parser, metav1Pkg)
	if len(kubeKinds) == 0 {
		// no objects in the roots
		return fmt.Errorf("no kube kind-objects found in all roots")
	}
	crdClient := utils.GetCRDClient()

	for groupKind := range kubeKinds {
		parser.NeedCRDFor(groupKind, func() *int {
			i := 256
			return &i
		}())
		crdRaw := parser.CustomResourceDefinitions[groupKind]
		glog.Infof("creating CRD: %v", groupKind)
		if _, err := crdClient.Create(ctx, &crdRaw, metav1.CreateOptions{}); err != nil {
			if !errors.IsAlreadyExists(err) {
				return err
			}
		}
	}
	return nil
}
