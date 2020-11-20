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
	"github.com/spf13/cobra"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"

	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

var (
	kubeConfig string
	namespace  string
	kubeClient *kubernetes.Clientset
	crdObj     *apiextensionv1.CustomResourceDefinition
	crObj      *rbacv1.ClusterRole
)

const (
	minioDesc = `
 kubectl plugin to manage MinIO DirectCSI.`
)

// NewCmdMinIO creates a new root command for kubectl-minio
func NewCmdMinIO(streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "directcsi",
		Short:        "manage MinIO DirectCSI",
		Long:         minioDesc,
		SilenceUsage: true,
	}

	cmd.AddCommand(newInstallCmd(cmd.OutOrStdout(), cmd.ErrOrStderr()))
	cmd.AddCommand(newRemoveCmd(cmd.OutOrStdout(), cmd.ErrOrStderr()))
	cmd.AddCommand(newDrivesCmd(cmd.OutOrStdout(), cmd.ErrOrStderr()))
	cmd.AddCommand(newVolumesCmd(cmd.OutOrStdout(), cmd.ErrOrStderr()))

	return cmd
}
