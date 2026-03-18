// This file is part of MinIO DirectPV
// Copyright (c) 2024 MinIO, Inc.
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

package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/minio/directpv/pkg/admin/installer"
	"github.com/minio/directpv/pkg/consts"
	legacyclient "github.com/minio/directpv/pkg/legacy/client"
)

// MigrateArgs denotest the migrate arguments
type MigrateArgs struct {
	Quiet             bool
	Retain            bool
	DrivesBackupFile  string
	VolumesBackupFile string
}

// Migrate migrates the directpv resources
func (client *Client) Migrate(ctx context.Context, args MigrateArgs) error {
	if !args.Retain {
		if args.DrivesBackupFile == "" || args.VolumesBackupFile == "" {
			return errors.New("backup file should not be empty")
		}
		if args.DrivesBackupFile == args.VolumesBackupFile {
			return errors.New("backup filenames are same")
		}
	}
	legacyClient, err := legacyclient.NewClient(client.K8s())
	if err != nil {
		return fmt.Errorf("unable to create legacy client; %w", err)
	}
	if err := installer.Migrate(ctx, &installer.Args{
		Quiet:  args.Quiet,
		Legacy: true,
	}, client.Client, legacyClient); err != nil {
		return fmt.Errorf("migration failed; %w", err)
	}
	if !args.Quiet {
		fmt.Println("Migration successful; Please restart the pods in '" + consts.AppName + "' namespace.")
	}
	if args.Retain {
		return nil
	}
	backupCreated, err := legacyClient.RemoveAllDrives(ctx, args.DrivesBackupFile)
	if err != nil {
		return fmt.Errorf("unable to remove legacy drive CRDs; %w", err)
	}
	if backupCreated && !args.Quiet {
		fmt.Println("Legacy drive CRDs backed up to", args.DrivesBackupFile)
	}
	backupCreated, err = legacyClient.RemoveAllVolumes(ctx, args.VolumesBackupFile)
	if err != nil {
		return fmt.Errorf("unable to remove legacy volume CRDs; %w", err)
	}
	if backupCreated && !args.Quiet {
		fmt.Println("Legacy volume CRDs backed up to", args.VolumesBackupFile)
	}
	return nil
}
