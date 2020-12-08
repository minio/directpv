// +build plugin

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

package main

import (
	"os"

	kctlcmd "github.com/minio/direct-csi/plugin/cmd"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func init() {
	if err := kctlcmd.NewCmdMinIO(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}).Execute(); err != nil {
		os.Exit(1)
	}
}
