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

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/minio/directpv/pkg/utils"

	"k8s.io/klog/v2"
)

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGSEGV)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		s := <-sigs
		klog.V(1).Infof("Exiting on signal %s %#v", s.String(), s)
		cancel()
		<-time.After(1 * time.Second)
		os.Exit(1)
	}()

	if filepath.Base(os.Args[0]) == "kubectl-direct_csi" {
		fmt.Println(utils.Bold(utils.Yellow("WARNING")), "plugin `direct-csi` will be deprecated in v2.3, please use `directpv` plugin instead")
	}

	if err := Execute(ctx); err != nil {
		fmt.Println(utils.Bold(utils.Red("ERROR")), err)
		os.Exit(1)
	}
}
