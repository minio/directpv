// This file is part of MinIO DirectPV
// Copyright (c) 2022 MinIO, Inc.
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

package sys

import (
	"errors"
	"os"
	"path"
)

// Mkdir is a util to mkdir with some special error handling
func Mkdir(name string, perm os.FileMode) (err error) {
	if err = os.Mkdir(name, perm); err != nil && errors.Is(err, os.ErrExist) {
		// If the device has Input/Output error, mkdir fails with "file exists" (https://github.com/golang/go/issues/8283)
		// Doing a Stat to confirm if we see I/O errors on the drive
		if _, err = os.Stat(path.Dir(name)); err == nil {
			return os.ErrExist
		}
	}
	return err
}
