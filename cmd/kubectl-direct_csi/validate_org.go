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

func parseOrg(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseOrg1, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseOrg1, false, nil
	}

	return parseOrg, false, ErrInvalid("[a-zA-Z]", r)
}

func parseOrg1(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseOrg1, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseOrg1, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseOrg1, false, nil
	}
	if r == '-' {
		return parseOrgSym1, true, nil
	}
	if r == '.' {
		return parseOrgPeriod1, true, nil
	}
	if r == '_' {
		return parseOrgUnderscore1, true, nil
	}

	return parseOrg, false, ErrInvalid("[a-zA-Z-._0-9]", r)
}

func parseOrgPeriod1(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseOrg1, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseOrg1, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseOrg1, false, nil
	}
	if r == '-' {
		return parseOrg, false, ErrInvalid("[a-zA-Z0-9]", r)
	}
	if r == '.' {
		return parseOrg, false, ErrInvalid("[a-zA-Z0-9]", r)
	}
	if r == '_' {
		return parseOrg, false, ErrInvalid("[a-zA-Z0-9]", r)
	}
	if r == ':' {
		return parseOrg, false, ErrInvalid("[a-zA-Z0-9]", r)
	}

	return parseOrg, false, ErrInvalid("[a-zA-Z-._0-9]", r)
}

func parseOrgSym1(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseOrg1, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseOrg1, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseOrg1, false, nil
	}
	if r == '-' {
		return parseOrgSym1, true, nil
	}
	if r == '_' {
		return parseOrg, false, ErrInvalid("[a-zA-Z-0-9]", r)
	}
	if r == '.' {
		return parseOrg, false, ErrInvalid("[a-zA-Z-0-9]", r)
	}
	if r == ':' {
		return parseOrg, false, ErrInvalid("[a-zA-Z-0-9]", r)
	}

	return parseOrg, false, ErrInvalid("[a-zA-Z-0-9]", r)
}

func parseOrgUnderscore1(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseOrg1, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseOrg1, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseOrg1, false, nil
	}
	if r == '_' {
		return parseOrgUnderscore2, true, nil
	}
	if r == '-' {
		return parseOrg, false, ErrInvalid("a-zA-Z_0-9", r)
	}
	if r == '.' {
		return parseOrg, false, ErrInvalid("[a-zA-Z_0-9]", r)
	}
	if r == ':' {
		return parseOrg, false, ErrInvalid("[a-zA-Z_0-9]", r)
	}

	return parseOrg, false, ErrInvalid("[a-zA-Z_0-9]", r)
}

func parseOrgUnderscore2(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseOrg1, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseOrg1, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseOrg1, false, nil
	}
	if r == '.' {
		return parseOrg, false, ErrInvalid("[a-zA-Z0-9]", r)
	}
	if r == '-' {
		return parseOrg, false, ErrInvalid("[a-zA-Z0-9]", r)
	}
	// max 2 consecutive underscores
	if r == '_' {
		return parseOrg, false, ErrInvalid("[a-zA-Z0-9]", r)
	}

	return parseOrg, false, ErrInvalid("[a-zA-Z0-9]", r)
}
