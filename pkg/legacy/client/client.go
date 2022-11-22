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

package client

import (
	"github.com/minio/directpv/pkg/k8s"
	directcsi "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog/v2"
)

// DirectCSIVersionLabelKey is the version with group and version ...
const DirectCSIVersionLabelKey = directcsi.Group + "/" + directcsi.Version

// DirectCSIDriveTypeMeta gets new direct-csi drive meta.
func DirectCSIDriveTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: DirectCSIVersionLabelKey,
		Kind:       "DirectCSIDrive",
	}
}

// DirectCSIVolumeTypeMeta gets new direct-csi volume meta.
func DirectCSIVolumeTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: DirectCSIVersionLabelKey,
		Kind:       "DirectCSIVolume",
	}
}

// GetGroupKindVersions gets group/version/kind of given versions.
func GetGroupKindVersions(group, kind string, versions ...string) (*schema.GroupVersionKind, error) {
	apiGroupResources, err := restmapper.GetAPIGroupResources(k8s.DiscoveryClient())
	if err != nil {
		klog.V(3).Infof("could not obtain API group resources: %v", err)
		return nil, err
	}
	restMapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources)
	gk := schema.GroupKind{
		Group: group,
		Kind:  kind,
	}
	mapper, err := restMapper.RESTMapping(gk, versions...)
	if err != nil {
		klog.V(3).Infof("could not find valid restmapping: %v", err)
		return nil, err
	}

	gvk := &schema.GroupVersionKind{
		Group:   mapper.Resource.Group,
		Version: mapper.Resource.Version,
		Kind:    mapper.Resource.Resource,
	}
	return gvk, nil
}
