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

package rest

import "errors"

var (
	errNoSubsetsFound      = errors.New("no subsets found for the node service")
	errNoEndpointsFound    = errors.New("no endpoints found for the node service")
	errNodeAPIPortNotFound = errors.New("api port for the node endpoints not found")
	errMountFailure        = errors.New("could not mount the drive")
	errUDevDataMismatch    = errors.New("udev data isn't matching")
	errForceRequired       = errors.New("force flag is required for formatting")
	errDuplicateDevice     = errors.New("found duplicate devices for drive")
)

type apiError struct {
	Description string `json:"description,omitempty"`
	Message     string `json:"message"`
}

func toAPIError(err error, message string) apiError {
	return apiError{
		Description: err.Error(),
		Message:     message,
	}
}
