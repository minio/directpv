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

package converter

const (
	versionV1Alpha1 = "direct.csi.min.io/v1alpha1"
	versionV1Beta1  = "direct.csi.min.io/v1beta1"
	versionV1Beta2  = "direct.csi.min.io/v1beta2"
	versionV1Beta3  = "direct.csi.min.io/v1beta3"
	versionV1Beta4  = "direct.csi.min.io/v1beta4"
	versionV1Beta5  = "direct.csi.min.io/v1beta5"
)

var supportedVersions = []string{
	versionV1Alpha1,
	versionV1Beta1,
	versionV1Beta2,
	versionV1Beta3,
	versionV1Beta4,
	versionV1Beta5,
} // ordered

type crdKind string

const (
	driveCRDKind  crdKind = "DirectCSIDrive"
	volumeCRDKind crdKind = "DirectCSIVolume"
)
