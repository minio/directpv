// Code generated by go-swagger; DO NOT EDIT.

// This file is part of MinIO KES
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
//

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// Principal principal
//
// swagger:model principal
type Principal struct {

	// s t s access key ID
	STSAccessKeyID string `json:"STSAccessKeyID,omitempty"`

	// s t s secret access key
	STSSecretAccessKey string `json:"STSSecretAccessKey,omitempty"`

	// s t s session token
	STSSessionToken string `json:"STSSessionToken,omitempty"`

	// account access key
	AccountAccessKey string `json:"accountAccessKey,omitempty"`

	// custom style ob
	CustomStyleOb string `json:"customStyleOb,omitempty"`

	// hm
	Hm bool `json:"hm,omitempty"`

	// ob
	Ob bool `json:"ob,omitempty"`
}

// Validate validates this principal
func (m *Principal) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this principal based on context it is used
func (m *Principal) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *Principal) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Principal) UnmarshalBinary(b []byte) error {
	var res Principal
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
