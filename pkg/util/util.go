// This file is part of MinIO Kubernetes Cloud
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

package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	directcsiclientset "github.com/minio/direct-csi/pkg/clientset"
	"github.com/minio/direct-csi/pkg/util/randomstring"
	clientset "k8s.io/client-go/kubernetes"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/viper"
)

func GetDirectCSIClientOrDie() directcsiclientset.Interface {
	var cfg *rest.Config
	var err error

	kubeConfig := viper.GetString("kube-config")

	if kubeConfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			panic(err)
		}
	} else {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			panic(err)
		}
	}
	return directcsiclientset.NewForConfigOrDie(cfg)
}

func GetKubeClientOrDie() clientset.Interface {
	var cfg *rest.Config
	var err error

	kubeConfig := viper.GetString("kube-config")

	if kubeConfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			panic(err)
		}
	} else {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			panic(err)
		}
	}
	return clientset.NewForConfigOrDie(cfg)
}

func Sanitize(s string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9-]")
	s = re.ReplaceAllString(s, "-")
	if s[len(s)-1] == '-' {
		s = s + "X"
	}
	return s
}

func GetNamespace() string {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}

	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}

	return "default"
}

func GenerateSanitizedUniqueNameFrom(name string) string {
	sanitizedName := Sanitize(name)
	// Max length of name is 255. If needed, cut out last 6 bytes
	// to make room for randomstring
	if len(sanitizedName) >= 255 {
		sanitizedName = sanitizedName[0:249]
	}

	// Get a 5 byte randomstring
	shortUUID := randomstring.New(5)

	// Concatenate sanitizedName (249) and shortUUID (5) with a '-' in between
	// Max length of the returned name cannot be more than 255 bytes
	return fmt.Sprintf("%s-%s", sanitizedName, shortUUID)
}
