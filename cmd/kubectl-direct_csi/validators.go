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
	"fmt"
	"net"
	"regexp"
	"strconv"

	"github.com/docker/distribution/reference"
)

type parseFunc func(r rune) (interface{}, bool, error)

var (
	ErrEndExpected     = fmt.Errorf("EOF expected")
	validHostPortRegex = regexp.MustCompile(`^` + reference.DomainRegexp.String() + `$`)
)

/*
 * FROM: https://docs.docker.com/engine/reference/commandline/tag/
 *
 * An image name is made up of slash-separated name components, optionally prefixed
 * by a registry hostname. The hostname must comply with standard DNS rules, but
 * may not contain underscores. If a hostname is present, it may optionally be
 * followed by a port number in the format :8080. If not present, the command uses
 * Docker’s public registry located at registry-1.docker.io by default. Name
 * components may contain lowercase letters, digits and separators. A separator is
 * defined as a period, one or two underscores, or one or more dashes. A name
 * component may not start or end with a separator.
 *
 * A tag name must be valid ASCII and may contain lowercase and uppercase letters,
 * digits, underscores, periods and dashes. A tag name may not start with a period
 * or a dash and may contain a maximum of 128 characters.
 *
 */

func validImage(img string) error {
	var next interface{}
	next = parseImage
	var cont bool
	var err error
	for _, r := range img {
		if err == ErrEndExpected {
			return err
		}
		next, cont, err = next.(func(r rune) (interface{}, bool, error))(r)
		if err != nil {
			return err
		}
	}
	if cont {
		return ErrInvalid("[a-zA-Z_-:.0-9]", '~')
	}
	return nil
}

func validOrg(org string) error {
	var next interface{}
	next = parseOrg
	var cont bool
	var err error
	for _, r := range org {
		if err == ErrEndExpected {
			return err
		}
		next, cont, err = next.(func(r rune) (interface{}, bool, error))(r)
		if err != nil {
			return err
		}
	}
	if cont {
		return ErrInvalid("[a-zA-Z_-:.0-9]", '~')
	}
	return nil
}

func validRegistry(registry string) error {
	host, port, err := net.SplitHostPort(registry)
	if err != nil {
		host = registry
		port = ""
	}
	// If match against the `host:port` pattern fails,
	// it might be `IPv6:port`, which will be captured by net.ParseIP(host)
	if !validHostPortRegex.MatchString(registry) && net.ParseIP(host) == nil {
		return fmt.Errorf("invalid host %q", host)
	}
	if port != "" {
		v, err := strconv.Atoi(port)
		if err != nil {
			return err
		}
		if v < 0 || v > 65535 {
			return fmt.Errorf("invalid port %q", port)
		}
	}
	return nil
}

func ErrInvalid(expected string, r rune) error {
	if r == '~' {
		return fmt.Errorf("expected %s, found EOF", expected)
	}
	return fmt.Errorf("expected %s, found %q", expected, r)
}
