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
	"unsafe"

	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/restmapper"

	"github.com/fatih/color"
	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"github.com/minio/direct-csi/pkg/utils"
)

var infoCmd = &cobra.Command{
	Use:          "info",
	Short:        "Info about direct-csi installation",
	SilenceUsage: true,
	RunE: func(c *cobra.Command, args []string) error {
		return info(c.Context(), args)
	},
}

const DefaultIdentity = "direct.csi.min.io"

func info(ctx context.Context, args []string) error {
	utils.Init()

	discoveryClient := utils.GetDiscoveryClient()
	apiGroupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		glog.Errorf("could not obtain API group resources: %v", err)
		return err
	}
	restMapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources)
	gvk := schema.GroupKind{
		Group: "storage.k8s.io",
		Kind:  "CSINode",
	}
	mapper, err := restMapper.RESTMapping(gvk, "v1", "v1beta1", "v1alpha1")
	if err != nil {
		glog.Errorf("could not find valid restmapping: %v", err)
		return err
	}

	metadataClient := utils.GetMetadataClient()
	csiNodes, err := metadataClient.Resource(mapper.Resource).List(ctx, metav1.ListOptions{})
	if err != nil {
		glog.Errorf("could not fetch %s/%s", gvk.Group, gvk.Kind)
		return err
	}

	defer func() {
		// since we are doing unsafe conversions
		if r := recover(); r != nil {
			glog.Errorf("could not find the resource version of CSINode in the k8s cluster")
		}
	}()

	nodeList := []string{}
	for _, csiNodeMeta := range csiNodes.Items {
		if mapper.Resource.Version == "v1" {
			csiNode := (*storagev1.CSINode)(unsafe.Pointer(&csiNodeMeta))
			for _, driver := range csiNode.Spec.Drivers {
				if driver.Name == DefaultIdentity {
					nodeList = append(nodeList, csiNode.Name)
					break
				}
			}
		}
		if mapper.Resource.Version == "v1beta1" {
			csiNode := (*storagev1beta1.CSINode)(unsafe.Pointer(&csiNodeMeta))
			for _, driver := range csiNode.Spec.Drivers {
				if driver.Name == DefaultIdentity {
					nodeList = append(nodeList, csiNode.Name)
					break
				}
			}
		}
		if mapper.Resource.Version == "v1alpha1" {
			// TODO: Query daemonsets to find the directcsi nodes
			return nil
		}
	}

	if len(nodeList) == 0 {
		bold := color.New(color.Bold).SprintFunc()
		fmt.Printf("  DirectCSI installation %s found\n", bold("NOT"))
		return nil
	}
	return nil
}
