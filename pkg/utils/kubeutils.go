// This file is part of MinIO Direct CSI
// Copyright (c) 2021 MinIO, Inc.
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

package utils

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	// support gcp, azure, and oidc client auth
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

const (
	// 127 or \u007f is the DEL character
	DEL rune = 127
)

func SafeGetLabels(obj metav1.Object) map[string]string {
	l := obj.GetLabels()
	if l == nil {
		return map[string]string{}
	}
	return l
}

func UpdateLabels(obj metav1.Object, labelKVs ...string) {
	labels := SafeGetLabels(obj)
	for i := 0; i < len(labelKVs); i += 2 {
		k := labelKVs[i]
		v := ""
		if len(labelKVs) > i+1 {
			v = labelKVs[i+1]
		}
		sk, sv := SanitizeLabelKV(k, v)
		labels[sk] = sv
	}
	obj.SetLabels(labels)
}

func GetLabelV(obj metav1.Object, key string) string {
	l := SafeGetLabels(obj)
	return l[key]
}

func SetLabelKV(obj metav1.Object, key, value string) {
	labels := SafeGetLabels(obj)

	sk, sv := SanitizeLabelKV(key, value)
	labels[sk] = sv

	obj.SetLabels(labels)
}

// NewObjectMeta - creates a new TypeMeta
// upcoming:
//   - verify API group/version/kind
//   - verify that kubernetes backend support group/version/kind (use discovery client)
func NewTypeMeta(groupVersion, resource string) metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: groupVersion,
		Kind:       resource,
	}
}

// NewObjectMeta - creates a new ObjectMeta with sanitized fields
func NewObjectMeta(
	name string,
	namespace string,
	labels map[string]string,
	annotations map[string]string,
	finalizers []string,
	ownerRefs []metav1.OwnerReference,
) metav1.ObjectMeta {

	return metav1.ObjectMeta{
		Name:            SanitizeKubeResourceName(name),
		Namespace:       SanitizeKubeResourceName(namespace),
		Annotations:     SanitizeLabelMap(annotations),
		Labels:          SanitizeLabelMap(labels),
		Finalizers:      SanitizeFinalizers(finalizers),
		OwnerReferences: ownerRefs,
	}
}

// SanitizeKubeResourceName - Sanitize given name to a valid kubernetes name format.
// RegEx for a kubernetes name is
//
//      ([a-z0-9][-a-z0-9]*)?[a-z0-9]
//
// with a max length of 253
//
// WARNING: This function will truncate to 253 bytes if the input is longer
func SanitizeKubeResourceName(name string) string {
	// alphaNumericLower - [a-z0-9]
	sanitizeAlphaNumericLower := func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		if r >= '0' && r <= '9' {
			return r
		}
		return '0'
	}

	// extAlphaNumericLower - [-a-z0-9]
	sanitizeExtAlphaNumericLower := func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		if r >= '0' && r <= '9' {
			return r
		}
		if r == '.' {
			return '-'
		}
		return '-'
	}

	sanitizeFn := func(strLength int) func(rune) rune {
		initialPos := true
		maxLength := 253
		currLength := 0

		if strLength > maxLength {
			strLength = maxLength
		}

		return func(r rune) rune {
			currLength = currLength + 1
			if initialPos {
				initialPos = false
				return sanitizeAlphaNumericLower(r)
			}

			if currLength == strLength || currLength == maxLength {
				return sanitizeAlphaNumericLower(r)
			}

			if currLength > maxLength {
				return DEL
			}

			return sanitizeExtAlphaNumericLower(r)
		}
	}

	return FmapString(name, sanitizeFn(len(name)))
}

// SanitizeLabelV - Sanitize given label value to valid kubernetes label format.
// RegEx for label value is
//
//      ([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]
//
// with a max length of 63
//
// WARNING: This function will truncate to 63 bytes if the input is longer
func SanitizeLabelV(value string) string {
	_, v := SanitizeLabelKV("", value)
	return v
}

// SanitizeLabelK - Sanitize given label key to valid kubernetes label format.
// RegEx for label key is
//
//      ([A-Za-z0-9][-A-Za-z0-9_.]*[/]?[-A-Za-z0-9_.]*)?[A-Za-z0-9]
//
// with a max length of 63
//
// WARNING: This function will truncate to 63 bytes if the input is longer
func SanitizeLabelK(key string) string {
	k, _ := SanitizeLabelKV(key, "")
	return k
}

func SanitizeLabelKV(key, value string) (string, string) {
	// charFmt - [A-Za-z0-9]
	sanitizeCharFmt := func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r
		}
		if r >= 'a' && r <= 'z' {
			return r
		}
		if r >= '0' && r <= '9' {
			return r
		}
		return 'x'
	}

	// extCharFmt - [-A-Za-z0-9_.]
	sanitizeExtCharFmt := func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r
		}
		if r >= 'a' && r <= 'z' {
			return r
		}
		if r >= '0' && r <= '9' {
			return r
		}
		if r == '.' || r == '_' {
			return r
		}
		return '-'
	}

	sanitizeKeyFn := func(strLength int) func(rune) rune {
		initialPos := true
		separatorFound := false

		maxLength := 63
		currLength := 0

		if strLength > maxLength {
			strLength = maxLength
		}

		return func(r rune) rune {
			currLength = currLength + 1
			if initialPos {
				initialPos = false
				return sanitizeCharFmt(r)
			}

			if currLength == strLength || currLength == maxLength {
				return sanitizeCharFmt(r)
			}

			if currLength > maxLength {
				return DEL
			}

			// one '/' separator is allowed in key names
			if r == '/' {
				if separatorFound {
					return '-'
				} else {
					separatorFound = true
					return r
				}
			}

			return sanitizeExtCharFmt(r)
		}
	}

	sanitizeValFn := func(strLength int) func(rune) rune {
		initialPos := true
		maxLength := 63
		currLength := 0

		if strLength > maxLength {
			strLength = maxLength
		}

		return func(r rune) rune {
			currLength = currLength + 1
			if initialPos {
				initialPos = false
				return sanitizeCharFmt(r)
			}

			if currLength == strLength || currLength == maxLength {
				return sanitizeCharFmt(r)
			}

			if currLength > maxLength {
				return DEL
			}

			return sanitizeExtCharFmt(r)
		}
	}

	return FmapString(key, sanitizeKeyFn(len(key))), FmapString(value, sanitizeValFn(len(value)))
}

func SanitizeLabelMap(kvMap map[string]string) map[string]string {
	retMap := map[string]string{}

	for k, v := range kvMap {
		sk, sv := SanitizeLabelKV(k, v)
		retMap[sk] = sv
	}
	return retMap
}

func SanitizeFinalizers(finalizers []string) []string {
	uniq := map[string]struct{}{}

	for _, f := range finalizers {
		uniq[f] = struct{}{}
	}

	toRet := []string{}
	for k := range uniq {
		toRet = append(toRet, SanitizeLabelK(k))
	}
	return toRet
}

func getKubeConfig() (*rest.Config, string, error) {
	kubeConfig := viper.GetString("kubeconfig")
	if kubeConfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			klog.Infof("could not find home dir: %v", err)
		}
		kubeConfig = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		if config, err = rest.InClusterConfig(); err != nil {
			return nil, "", err
		}
	}
	config.QPS = float32(MaxThreadCount / 2)
	config.Burst = MaxThreadCount
	return config, "", nil
}

func GetKubeConfig() (*rest.Config, error) {
	config, _, err := getKubeConfig()
	return config, err
}

func GetGroupKindVersions(group, kind string, versions ...string) (*schema.GroupVersionKind, error) {
	discoveryClient := GetDiscoveryClient()
	apiGroupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		klog.Errorf("could not obtain API group resources: %v", err)
		return nil, err
	}
	restMapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources)
	gk := schema.GroupKind{
		Group: group,
		Kind:  kind,
	}
	mapper, err := restMapper.RESTMapping(gk, versions...)
	if err != nil {
		klog.Errorf("could not find valid restmapping: %v", err)
		return nil, err
	}

	gvk := &schema.GroupVersionKind{
		Group:   mapper.Resource.Group,
		Version: mapper.Resource.Version,
		Kind:    mapper.Resource.Resource,
	}
	return gvk, nil
}
