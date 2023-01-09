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

package auth

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/minio/directpv/models"
)

// LoginNoContentCode is the HTTP code returned for type LoginNoContent
const LoginNoContentCode int = 204

/*
LoginNoContent A successful login.

swagger:response loginNoContent
*/
type LoginNoContent struct {
}

// NewLoginNoContent creates LoginNoContent with default headers values
func NewLoginNoContent() *LoginNoContent {

	return &LoginNoContent{}
}

// WriteResponse to the client
func (o *LoginNoContent) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(204)
}

/*
LoginDefault Generic error response.

swagger:response loginDefault
*/
type LoginDefault struct {
	_statusCode int

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewLoginDefault creates LoginDefault with default headers values
func NewLoginDefault(code int) *LoginDefault {
	if code <= 0 {
		code = 500
	}

	return &LoginDefault{
		_statusCode: code,
	}
}

// WithStatusCode adds the status to the login default response
func (o *LoginDefault) WithStatusCode(code int) *LoginDefault {
	o._statusCode = code
	return o
}

// SetStatusCode sets the status to the login default response
func (o *LoginDefault) SetStatusCode(code int) {
	o._statusCode = code
}

// WithPayload adds the payload to the login default response
func (o *LoginDefault) WithPayload(payload *models.Error) *LoginDefault {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the login default response
func (o *LoginDefault) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *LoginDefault) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(o._statusCode)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
