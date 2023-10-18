// This file is part of MinIO DirectPV
// Copyright (c) 2023 MinIO, Inc.
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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/minio/directpv/pkg/client"
	"github.com/minio/directpv/pkg/sys"
	"github.com/minio/directpv/pkg/types"
	"github.com/minio/directpv/pkg/xfs"
	xfilepath "github.com/minio/filepath"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

var (
	volumeID   string
	dryRunFlag bool
)

var copyCmd = &cobra.Command{
	Use:           "copy SRC-DRIVE DEST-DRIVE --volume-id VOLUME-ID",
	Short:         "copy the volume data from source drive to destination drive",
	Aliases:       []string{"cp"},
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(c *cobra.Command, args []string) error {
		switch len(args) {
		case 0:
			return errors.New("source and destination DRIVE-IDs should be provided")
		case 1:
			return errors.New("both the source and destination DRIVE-IDs should be provided")
		case 2:
		default:
			return errors.New("invalid syntax")
		}
		if volumeID == "" {
			return errors.New("'--volume-id' should be provided")
		}
		if args[0] == args[1] {
			return errors.New("both the source and destination DRIVE-IDs are same")
		}

		ctx := c.Context()
		srcDrive, err := client.DriveClient().Get(ctx, args[0], metav1.GetOptions{
			TypeMeta: types.NewDriveTypeMeta(),
		})
		if err != nil {
			return err
		}
		destDrive, err := client.DriveClient().Get(ctx, args[1], metav1.GetOptions{
			TypeMeta: types.NewDriveTypeMeta(),
		})
		if err != nil {
			return err
		}
		volume, err := client.VolumeClient().Get(ctx, volumeID, metav1.GetOptions{
			TypeMeta: types.NewVolumeTypeMeta(),
		})
		if err != nil {
			return err
		}
		if !destDrive.VolumeExist(volumeID) {
			return errors.New("volume finalizer not found on the destination drive")
		}
		if volume.GetNodeID() != nodeID {
			return errors.New("the nodeID in the volume doesn't match")
		}
		if err := checkDrive(srcDrive); err != nil {
			klog.ErrorS(err, "unable to check the source drive", "driveID", srcDrive.Name)
			return err
		}
		if err := checkDrive(destDrive); err != nil {
			klog.ErrorS(err, "unable to check the destination drive", "driveID", destDrive.Name)
			return err
		}
		err = startCopy(ctx, srcDrive, destDrive, volume)
		if err != nil {
			klog.ErrorS(err, "unable to copy", "source", srcDrive.Name, "destination", destDrive.Name)
		}
		return err
	},
}

func init() {
	copyCmd.PersistentFlags().StringVar(&volumeID, "volume-id", volumeID, "Set the volumeID of the volume to be copied")
	copyCmd.PersistentFlags().BoolVar(&dryRunFlag, "dry-run", dryRunFlag, "Enable dry-run mode")
}

func checkDrive(drive *types.Drive) error {
	if drive.GetNodeID() != nodeID {
		return errors.New("the nodeID in the drive doesn't match")
	}
	if _, err := os.Lstat(types.GetVolumeRootDir(drive.Status.FSUUID)); err != nil {
		return fmt.Errorf("unable to stat the volume root directory; %v", err)
	}
	if _, err := sys.GetDeviceByFSUUID(drive.Status.FSUUID); err != nil {
		return fmt.Errorf("unable to find device by its FSUUID; %v", err)
	}
	return nil
}

func startCopy(ctx context.Context, srcDrive, destDrive *types.Drive, volume *types.Volume) error {
	if dryRunFlag {
		return nil
	}

	sourcePath := types.GetVolumeDir(srcDrive.Status.FSUUID, volume.Name)
	destPath := types.GetVolumeDir(destDrive.Status.FSUUID, volume.Name)

	if _, err := os.Lstat(sourcePath); err != nil {
		return fmt.Errorf("unable to stat the sourcePath %v; %v", sourcePath, err)
	}
	if err := sys.Mkdir(destPath, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("unable to create the targetPath %v; %v", destPath, err)
	}

	quota := xfs.Quota{
		HardLimit: uint64(volume.Status.TotalCapacity),
		SoftLimit: uint64(volume.Status.TotalCapacity),
	}
	if err := xfs.SetQuota(ctx, "/dev/"+string(destDrive.GetDriveName()), destPath, volume.Name, quota, false); err != nil {
		return fmt.Errorf("unable to set quota on volume data path; %w", err)
	}

	ctxWitCancel, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		printProgress(ctx, srcDrive, destDrive, volume)
	}()
	go func() {
		logProgress(ctxWitCancel, srcDrive, destDrive, volume)
	}()

	return copyData(sourcePath, destPath)
}

func printProgress(ctx context.Context, srcDrive, destDrive *types.Drive, volume *types.Volume) error {
	sourceQ, err := xfs.GetQuota(ctx, "/dev/"+string(srcDrive.GetDriveName()), volume.Name)
	if err != nil {
		klog.ErrorS(err, "unable to get quota of the source drive", "source drive", srcDrive.GetDriveName(), "volume", volume.Name)
		return err
	}
	destQ, err := xfs.GetQuota(ctx, "/dev/"+string(destDrive.GetDriveName()), volume.Name)
	if err != nil {
		klog.ErrorS(err, "unable to get quota of the destination drive", "destination drive", destDrive.GetDriveName(), "volume", volume.Name)
		return err
	}
	fmt.Printf("\nCopied %v/%v", humanize.IBytes(destQ.CurrentSpace), humanize.IBytes(sourceQ.CurrentSpace))
	return nil
}

func logProgress(ctx context.Context, srcDrive, destDrive *types.Drive, volume *types.Volume) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := printProgress(ctx, srcDrive, destDrive, volume); err != nil {
				return
			}
		}
	}
}

func copyData(source, destination string) error {
	visitFn := func(f string, fi os.FileInfo, _ error) error {
		targetPath := filepath.Join(destination, strings.TrimPrefix(f, source))
		switch {
		case fi.Mode()&os.ModeDir != 0:
			return os.MkdirAll(targetPath, fi.Mode().Perm())
		case fi.Mode()&os.ModeType == 0:
			if targetFi, err := os.Lstat(targetPath); err == nil {
				if targetFi.ModTime().Equal(fi.ModTime()) && targetFi.Size() == fi.Size() {
					return nil
				}
			}
			reader, err := os.Open(f)
			if err != nil {
				return err
			}
			writer, err := os.OpenFile(targetPath, os.O_RDWR|os.O_CREATE, 0o755)
			if err != nil {
				return err
			}
			if _, err := io.CopyN(writer, reader, fi.Size()); err != nil {
				return err
			}
			stat, ok := fi.Sys().(*syscall.Stat_t)
			if !ok {
				return fmt.Errorf("unable to get the stat information for %v", f)
			}
			if err := os.Chown(targetPath, int(stat.Uid), int(stat.Gid)); err != nil {
				return fmt.Errorf("unable to set UID and GID to path %v; %v", targetPath, err)
			}
			if err := os.Chmod(targetPath, fi.Mode().Perm()); err != nil {
				return fmt.Errorf("unable to chmod on path %v; %v", targetPath, err)
			}
			return os.Chtimes(targetPath, fi.ModTime(), fi.ModTime())
		case fi.Mode()&os.ModeSymlink != 0:
			// ToDo: Handle symlink
			return nil
		default:
			// unsupported modes
			return nil
		}
	}
	return xfilepath.Walk(source, visitFn)
}
