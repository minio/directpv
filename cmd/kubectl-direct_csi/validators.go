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
	"strings"

	"github.com/docker/distribution/reference"
	corev1 "k8s.io/api/core/v1"
)

var (
	errEndExpected     = fmt.Errorf("EOF expected")
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
		if err == errEndExpected {
			return err
		}
		next, cont, err = next.(func(r rune) (interface{}, bool, error))(r)
		if err != nil {
			return err
		}
	}
	if cont {
		return errInvalid("[a-zA-Z_-:.0-9]", '~')
	}
	return nil
}

func validOrg(org string) error {
	var next interface{}
	next = parseOrg
	var cont bool
	var err error
	for _, r := range org {
		if err == errEndExpected {
			return err
		}
		next, cont, err = next.(func(r rune) (interface{}, bool, error))(r)
		if err != nil {
			return err
		}
	}
	if cont {
		return errInvalid("[a-zA-Z_-:.0-9]", '~')
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

func errInvalid(expected string, r rune) error {
	if r == '~' {
		return fmt.Errorf("expected %s, found EOF", expected)
	}
	return fmt.Errorf("expected %s, found %q", expected, r)
}

func parseNodeSelector(values []string) (map[string]string, error) {
	nodeSelector := map[string]string{}
	for _, value := range values {
		tokens := strings.Split(value, "=")
		if len(tokens) != 2 {
			return nil, fmt.Errorf("invalid node selector value %v", value)
		}
		if tokens[0] == "" {
			return nil, fmt.Errorf("invalid key in node selector value %v", value)
		}
		nodeSelector[tokens[0]] = tokens[1]
	}
	return nodeSelector, nil
}

func parseTolerations(values []string) ([]corev1.Toleration, error) {
	tolerations := []corev1.Toleration{}
	for _, value := range values {
		var k, v, e string
		tokens := strings.SplitN(value, "=", 2)
		switch len(tokens) {
		case 1:
			k = tokens[0]
			tokens = strings.Split(k, ":")
			switch len(tokens) {
			case 1:
			case 2:
				k, e = tokens[0], tokens[1]
			default:
				if len(tokens) != 2 {
					return nil, fmt.Errorf("invalid toleration %v", value)
				}
			}
		case 2:
			k, v = tokens[0], tokens[1]
		default:
			if len(tokens) != 2 {
				return nil, fmt.Errorf("invalid toleration %v", value)
			}
		}
		if k == "" {
			return nil, fmt.Errorf("invalid key in toleration %v", value)
		}
		if v != "" {
			if tokens = strings.Split(v, ":"); len(tokens) != 2 {
				return nil, fmt.Errorf("invalid value in toleration %v", value)
			}
			v, e = tokens[0], tokens[1]
		}
		effect := corev1.TaintEffect(e)
		switch effect {
		case corev1.TaintEffectNoSchedule, corev1.TaintEffectPreferNoSchedule, corev1.TaintEffectNoExecute:
		default:
			return nil, fmt.Errorf("invalid toleration effect in toleration %v", value)
		}
		operator := corev1.TolerationOpExists
		if v != "" {
			operator = corev1.TolerationOpEqual
		}
		tolerations = append(tolerations, corev1.Toleration{
			Key:      k,
			Operator: operator,
			Value:    v,
			Effect:   effect,
		})
	}

	return tolerations, nil
}
