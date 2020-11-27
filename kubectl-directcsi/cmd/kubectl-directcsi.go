/*
 * This file is part of MinIO Direct CSI
 * Copyright (C) 2020, MinIO, Inc.
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, version 3,
 * as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License, version 3,
 * along with this program.  If not, see <http://www.gnu.org/licenses/>
 *
 */

package cmd

import (
	"log"

	"github.com/minio/direct-csi/pkg/clientset/scheme"
	"github.com/rakyll/statik/fs"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"

	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	// Statik CRD assets for our plugin
	_ "github.com/minio/kubectl-directcsi/statik"
)

var (
	kubeConfig string
	namespace  string
	kubeClient *kubernetes.Clientset
	driveObj   *apiextensionv1.CustomResourceDefinition
	volObj     *apiextensionv1.CustomResourceDefinition
)

const (
	minioDesc = `
 kubectl plugin to manage MinIO DirectCSI.`
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	emfs, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}
	sch := runtime.NewScheme()
	scheme.AddToScheme(sch)
	apiextensionv1.AddToScheme(sch)
	decode := serializer.NewCodecFactory(sch).UniversalDeserializer().Decode

	contents, err := fs.ReadFile(emfs, "/crd/direct.csi.min.io_directcsidrives.yaml")
	if err != nil {
		log.Fatal(err)
	}

	obj, _, err := decode(contents, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	var ok bool
	driveObj, ok = obj.(*apiextensionv1.CustomResourceDefinition)
	if !ok {
		log.Fatal("Unable to locate Drive CustomResourceDefinition object")
	}

	contents, err = fs.ReadFile(emfs, "/crd/direct.csi.min.io_directcsivolumes.yaml")
	if err != nil {
		log.Fatal(err)
	}

	obj, _, err = decode(contents, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	volObj, ok = obj.(*apiextensionv1.CustomResourceDefinition)
	if !ok {
		log.Fatal("Unable to locate Volume CustomResourceDefinition object")
	}
}

// NewCmdMinIO creates a new root command for kubectl-minio
func NewCmdMinIO(streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "directcsi",
		Short:        "manage MinIO DirectCSI",
		Long:         minioDesc,
		SilenceUsage: true,
	}

	cmd.AddCommand(newInstallCmd(cmd.OutOrStdout(), cmd.ErrOrStderr()))
	cmd.AddCommand(newUninstallCmd(cmd.OutOrStdout(), cmd.ErrOrStderr()))
	cmd.AddCommand(newDrivesCmd(cmd.OutOrStdout(), cmd.ErrOrStderr()))
	cmd.AddCommand(newVolumesCmd(cmd.OutOrStdout(), cmd.ErrOrStderr()))

	return cmd
}
