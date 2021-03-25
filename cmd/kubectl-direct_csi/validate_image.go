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

func parseImage(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseImage1, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseImage1, false, nil
	}

	return parseImage, false, ErrInvalid("[a-zA-Z]", r)
}

func parseImage1(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseImage1, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseImage1, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseImage1, false, nil
	}
	if r == '-' {
		return parseSym1, true, nil
	}
	if r == '.' {
		return parsePeriod1, true, nil
	}
	if r == '_' {
		return parseUnderscore1, true, nil
	}
	if r == ':' {
		return parseTag1, true, nil
	}

	return parseImage, false, ErrInvalid("[a-zA-Z-:._0-9]", r)
}

func parsePeriod1(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseImage1, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseImage1, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseImage1, false, nil
	}
	if r == '-' {
		return parseImage, false, ErrInvalid("[a-zA-Z0-9]", r)
	}
	if r == '.' {
		return parseImage, false, ErrInvalid("[a-zA-Z0-9]", r)
	}
	if r == '_' {
		return parseImage, false, ErrInvalid("[a-zA-Z0-9]", r)
	}
	if r == ':' {
		return parseImage, false, ErrInvalid("[a-zA-Z0-9]", r)
	}

	return parseImage, false, ErrInvalid("[a-zA-Z-._0-9]", r)
}

func parseSym1(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseImage1, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseImage1, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseImage1, false, nil
	}
	if r == '-' {
		return parseSym1, true, nil
	}
	if r == '_' {
		return parseImage, false, ErrInvalid("[a-zA-Z-0-9]", r)
	}
	if r == '.' {
		return parseImage, false, ErrInvalid("[a-zA-Z-0-9]", r)
	}
	if r == ':' {
		return parseImage, false, ErrInvalid("[a-zA-Z-0-9]", r)
	}

	return parseImage, false, ErrInvalid("[a-zA-Z-0-9]", r)
}

func parseUnderscore1(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseImage1, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseImage1, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseImage1, false, nil
	}
	if r == '_' {
		return parseUnderscore2, true, nil
	}
	if r == '-' {
		return parseImage, false, ErrInvalid("a-zA-Z_0-9", r)
	}
	if r == '.' {
		return parseImage, false, ErrInvalid("[a-zA-Z_0-9]", r)
	}
	if r == ':' {
		return parseImage, false, ErrInvalid("[a-zA-Z_0-9]", r)
	}

	return parseImage, false, ErrInvalid("[a-zA-Z_0-9]", r)
}

func parseUnderscore2(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseImage1, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseImage1, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseImage1, false, nil
	}
	if r == '.' {
		return parseImage, false, ErrInvalid("[a-zA-Z0-9]", r)
	}
	if r == '-' {
		return parseImage, false, ErrInvalid("[a-zA-Z0-9]", r)
	}
	// max 2 consecutive underscores
	if r == '_' {
		return parseImage, false, ErrInvalid("[a-zA-Z0-9]", r)
	}
	if r == ':' {
		return parseImage, false, ErrInvalid("[a-zA-Z0-9]", r)
	}

	return parseImage, false, ErrInvalid("[a-zA-Z0-9]", r)
}

func parseTag1(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag2, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag2, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag2, false, nil
	}
	if r == '_' {
		return parseTag2, false, nil
	}
	if r == '.' {
		return parseImage, false, ErrInvalid("[a-zA-Z_0-9]", r)
	}
	if r == '-' {
		return parseImage, false, ErrInvalid("[a-zA-Z_0-9]", r)
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_0-9]", r)
}
func parseTag2(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag3, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag3, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag3, false, nil
	}
	if r == '_' {
		return parseTag3, false, nil
	}
	if r == '.' {
		return parseTag3, false, nil
	}
	if r == '-' {
		return parseTag3, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag3(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag4, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag4, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag4, false, nil
	}
	if r == '_' {
		return parseTag4, false, nil
	}
	if r == '.' {
		return parseTag4, false, nil
	}
	if r == '-' {
		return parseTag4, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag4(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag5, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag5, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag5, false, nil
	}
	if r == '_' {
		return parseTag5, false, nil
	}
	if r == '.' {
		return parseTag5, false, nil
	}
	if r == '-' {
		return parseTag5, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag5(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag6, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag6, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag6, false, nil
	}
	if r == '_' {
		return parseTag6, false, nil
	}
	if r == '.' {
		return parseTag6, false, nil
	}
	if r == '-' {
		return parseTag6, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag6(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag7, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag7, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag7, false, nil
	}
	if r == '_' {
		return parseTag7, false, nil
	}
	if r == '.' {
		return parseTag7, false, nil
	}
	if r == '-' {
		return parseTag7, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag7(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag8, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag8, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag8, false, nil
	}
	if r == '_' {
		return parseTag8, false, nil
	}
	if r == '.' {
		return parseTag8, false, nil
	}
	if r == '-' {
		return parseTag8, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag8(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag9, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag9, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag9, false, nil
	}
	if r == '_' {
		return parseTag9, false, nil
	}
	if r == '.' {
		return parseTag9, false, nil
	}
	if r == '-' {
		return parseTag9, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag9(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag10, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag10, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag10, false, nil
	}
	if r == '_' {
		return parseTag10, false, nil
	}
	if r == '.' {
		return parseTag10, false, nil
	}
	if r == '-' {
		return parseTag10, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag10(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag11, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag11, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag11, false, nil
	}
	if r == '_' {
		return parseTag11, false, nil
	}
	if r == '.' {
		return parseTag11, false, nil
	}
	if r == '-' {
		return parseTag11, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag11(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag12, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag12, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag12, false, nil
	}
	if r == '_' {
		return parseTag12, false, nil
	}
	if r == '.' {
		return parseTag12, false, nil
	}
	if r == '-' {
		return parseTag12, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag12(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag13, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag13, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag13, false, nil
	}
	if r == '_' {
		return parseTag13, false, nil
	}
	if r == '.' {
		return parseTag13, false, nil
	}
	if r == '-' {
		return parseTag13, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag13(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag14, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag14, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag14, false, nil
	}
	if r == '_' {
		return parseTag14, false, nil
	}
	if r == '.' {
		return parseTag14, false, nil
	}
	if r == '-' {
		return parseTag14, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag14(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag15, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag15, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag15, false, nil
	}
	if r == '_' {
		return parseTag15, false, nil
	}
	if r == '.' {
		return parseTag15, false, nil
	}
	if r == '-' {
		return parseTag15, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag15(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag16, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag16, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag16, false, nil
	}
	if r == '_' {
		return parseTag16, false, nil
	}
	if r == '.' {
		return parseTag16, false, nil
	}
	if r == '-' {
		return parseTag16, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag16(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag17, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag17, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag17, false, nil
	}
	if r == '_' {
		return parseTag17, false, nil
	}
	if r == '.' {
		return parseTag17, false, nil
	}
	if r == '-' {
		return parseTag17, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag17(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag18, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag18, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag18, false, nil
	}
	if r == '_' {
		return parseTag18, false, nil
	}
	if r == '.' {
		return parseTag18, false, nil
	}
	if r == '-' {
		return parseTag18, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag18(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag19, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag19, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag19, false, nil
	}
	if r == '_' {
		return parseTag19, false, nil
	}
	if r == '.' {
		return parseTag19, false, nil
	}
	if r == '-' {
		return parseTag19, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag19(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag20, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag20, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag20, false, nil
	}
	if r == '_' {
		return parseTag20, false, nil
	}
	if r == '.' {
		return parseTag20, false, nil
	}
	if r == '-' {
		return parseTag20, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag20(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag21, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag21, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag21, false, nil
	}
	if r == '_' {
		return parseTag21, false, nil
	}
	if r == '.' {
		return parseTag21, false, nil
	}
	if r == '-' {
		return parseTag21, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag21(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag22, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag22, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag22, false, nil
	}
	if r == '_' {
		return parseTag22, false, nil
	}
	if r == '.' {
		return parseTag22, false, nil
	}
	if r == '-' {
		return parseTag22, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag22(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag23, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag23, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag23, false, nil
	}
	if r == '_' {
		return parseTag23, false, nil
	}
	if r == '.' {
		return parseTag23, false, nil
	}
	if r == '-' {
		return parseTag23, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag23(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag24, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag24, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag24, false, nil
	}
	if r == '_' {
		return parseTag24, false, nil
	}
	if r == '.' {
		return parseTag24, false, nil
	}
	if r == '-' {
		return parseTag24, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag24(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag25, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag25, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag25, false, nil
	}
	if r == '_' {
		return parseTag25, false, nil
	}
	if r == '.' {
		return parseTag25, false, nil
	}
	if r == '-' {
		return parseTag25, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag25(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag26, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag26, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag26, false, nil
	}
	if r == '_' {
		return parseTag26, false, nil
	}
	if r == '.' {
		return parseTag26, false, nil
	}
	if r == '-' {
		return parseTag26, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag26(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag27, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag27, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag27, false, nil
	}
	if r == '_' {
		return parseTag27, false, nil
	}
	if r == '.' {
		return parseTag27, false, nil
	}
	if r == '-' {
		return parseTag27, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag27(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag28, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag28, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag28, false, nil
	}
	if r == '_' {
		return parseTag28, false, nil
	}
	if r == '.' {
		return parseTag28, false, nil
	}
	if r == '-' {
		return parseTag28, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag28(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag29, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag29, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag29, false, nil
	}
	if r == '_' {
		return parseTag29, false, nil
	}
	if r == '.' {
		return parseTag29, false, nil
	}
	if r == '-' {
		return parseTag29, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag29(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag30, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag30, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag30, false, nil
	}
	if r == '_' {
		return parseTag30, false, nil
	}
	if r == '.' {
		return parseTag30, false, nil
	}
	if r == '-' {
		return parseTag30, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag30(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag31, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag31, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag31, false, nil
	}
	if r == '_' {
		return parseTag31, false, nil
	}
	if r == '.' {
		return parseTag31, false, nil
	}
	if r == '-' {
		return parseTag31, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag31(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag32, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag32, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag32, false, nil
	}
	if r == '_' {
		return parseTag32, false, nil
	}
	if r == '.' {
		return parseTag32, false, nil
	}
	if r == '-' {
		return parseTag32, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag32(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag33, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag33, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag33, false, nil
	}
	if r == '_' {
		return parseTag33, false, nil
	}
	if r == '.' {
		return parseTag33, false, nil
	}
	if r == '-' {
		return parseTag33, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag33(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag34, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag34, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag34, false, nil
	}
	if r == '_' {
		return parseTag34, false, nil
	}
	if r == '.' {
		return parseTag34, false, nil
	}
	if r == '-' {
		return parseTag34, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag34(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag35, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag35, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag35, false, nil
	}
	if r == '_' {
		return parseTag35, false, nil
	}
	if r == '.' {
		return parseTag35, false, nil
	}
	if r == '-' {
		return parseTag35, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag35(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag36, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag36, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag36, false, nil
	}
	if r == '_' {
		return parseTag36, false, nil
	}
	if r == '.' {
		return parseTag36, false, nil
	}
	if r == '-' {
		return parseTag36, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag36(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag37, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag37, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag37, false, nil
	}
	if r == '_' {
		return parseTag37, false, nil
	}
	if r == '.' {
		return parseTag37, false, nil
	}
	if r == '-' {
		return parseTag37, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag37(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag38, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag38, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag38, false, nil
	}
	if r == '_' {
		return parseTag38, false, nil
	}
	if r == '.' {
		return parseTag38, false, nil
	}
	if r == '-' {
		return parseTag38, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag38(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag39, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag39, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag39, false, nil
	}
	if r == '_' {
		return parseTag39, false, nil
	}
	if r == '.' {
		return parseTag39, false, nil
	}
	if r == '-' {
		return parseTag39, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag39(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag40, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag40, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag40, false, nil
	}
	if r == '_' {
		return parseTag40, false, nil
	}
	if r == '.' {
		return parseTag40, false, nil
	}
	if r == '-' {
		return parseTag40, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag40(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag41, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag41, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag41, false, nil
	}
	if r == '_' {
		return parseTag41, false, nil
	}
	if r == '.' {
		return parseTag41, false, nil
	}
	if r == '-' {
		return parseTag41, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag41(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag42, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag42, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag42, false, nil
	}
	if r == '_' {
		return parseTag42, false, nil
	}
	if r == '.' {
		return parseTag42, false, nil
	}
	if r == '-' {
		return parseTag42, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag42(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag43, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag43, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag43, false, nil
	}
	if r == '_' {
		return parseTag43, false, nil
	}
	if r == '.' {
		return parseTag43, false, nil
	}
	if r == '-' {
		return parseTag43, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag43(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag44, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag44, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag44, false, nil
	}
	if r == '_' {
		return parseTag44, false, nil
	}
	if r == '.' {
		return parseTag44, false, nil
	}
	if r == '-' {
		return parseTag44, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag44(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag45, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag45, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag45, false, nil
	}
	if r == '_' {
		return parseTag45, false, nil
	}
	if r == '.' {
		return parseTag45, false, nil
	}
	if r == '-' {
		return parseTag45, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag45(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag46, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag46, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag46, false, nil
	}
	if r == '_' {
		return parseTag46, false, nil
	}
	if r == '.' {
		return parseTag46, false, nil
	}
	if r == '-' {
		return parseTag46, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag46(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag47, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag47, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag47, false, nil
	}
	if r == '_' {
		return parseTag47, false, nil
	}
	if r == '.' {
		return parseTag47, false, nil
	}
	if r == '-' {
		return parseTag47, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag47(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag48, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag48, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag48, false, nil
	}
	if r == '_' {
		return parseTag48, false, nil
	}
	if r == '.' {
		return parseTag48, false, nil
	}
	if r == '-' {
		return parseTag48, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag48(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag49, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag49, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag49, false, nil
	}
	if r == '_' {
		return parseTag49, false, nil
	}
	if r == '.' {
		return parseTag49, false, nil
	}
	if r == '-' {
		return parseTag49, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag49(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag50, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag50, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag50, false, nil
	}
	if r == '_' {
		return parseTag50, false, nil
	}
	if r == '.' {
		return parseTag50, false, nil
	}
	if r == '-' {
		return parseTag50, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag50(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag51, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag51, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag51, false, nil
	}
	if r == '_' {
		return parseTag51, false, nil
	}
	if r == '.' {
		return parseTag51, false, nil
	}
	if r == '-' {
		return parseTag51, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag51(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag52, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag52, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag52, false, nil
	}
	if r == '_' {
		return parseTag52, false, nil
	}
	if r == '.' {
		return parseTag52, false, nil
	}
	if r == '-' {
		return parseTag52, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag52(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag53, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag53, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag53, false, nil
	}
	if r == '_' {
		return parseTag53, false, nil
	}
	if r == '.' {
		return parseTag53, false, nil
	}
	if r == '-' {
		return parseTag53, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag53(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag54, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag54, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag54, false, nil
	}
	if r == '_' {
		return parseTag54, false, nil
	}
	if r == '.' {
		return parseTag54, false, nil
	}
	if r == '-' {
		return parseTag54, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag54(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag55, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag55, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag55, false, nil
	}
	if r == '_' {
		return parseTag55, false, nil
	}
	if r == '.' {
		return parseTag55, false, nil
	}
	if r == '-' {
		return parseTag55, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag55(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag56, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag56, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag56, false, nil
	}
	if r == '_' {
		return parseTag56, false, nil
	}
	if r == '.' {
		return parseTag56, false, nil
	}
	if r == '-' {
		return parseTag56, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag56(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag57, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag57, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag57, false, nil
	}
	if r == '_' {
		return parseTag57, false, nil
	}
	if r == '.' {
		return parseTag57, false, nil
	}
	if r == '-' {
		return parseTag57, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag57(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag58, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag58, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag58, false, nil
	}
	if r == '_' {
		return parseTag58, false, nil
	}
	if r == '.' {
		return parseTag58, false, nil
	}
	if r == '-' {
		return parseTag58, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag58(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag59, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag59, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag59, false, nil
	}
	if r == '_' {
		return parseTag59, false, nil
	}
	if r == '.' {
		return parseTag59, false, nil
	}
	if r == '-' {
		return parseTag59, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag59(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag60, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag60, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag60, false, nil
	}
	if r == '_' {
		return parseTag60, false, nil
	}
	if r == '.' {
		return parseTag60, false, nil
	}
	if r == '-' {
		return parseTag60, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag60(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag61, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag61, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag61, false, nil
	}
	if r == '_' {
		return parseTag61, false, nil
	}
	if r == '.' {
		return parseTag61, false, nil
	}
	if r == '-' {
		return parseTag61, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag61(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag62, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag62, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag62, false, nil
	}
	if r == '_' {
		return parseTag62, false, nil
	}
	if r == '.' {
		return parseTag62, false, nil
	}
	if r == '-' {
		return parseTag62, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag62(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag63, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag63, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag63, false, nil
	}
	if r == '_' {
		return parseTag63, false, nil
	}
	if r == '.' {
		return parseTag63, false, nil
	}
	if r == '-' {
		return parseTag63, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag63(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag64, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag64, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag64, false, nil
	}
	if r == '_' {
		return parseTag64, false, nil
	}
	if r == '.' {
		return parseTag64, false, nil
	}
	if r == '-' {
		return parseTag64, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag64(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag65, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag65, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag65, false, nil
	}
	if r == '_' {
		return parseTag65, false, nil
	}
	if r == '.' {
		return parseTag65, false, nil
	}
	if r == '-' {
		return parseTag65, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag65(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag66, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag66, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag66, false, nil
	}
	if r == '_' {
		return parseTag66, false, nil
	}
	if r == '.' {
		return parseTag66, false, nil
	}
	if r == '-' {
		return parseTag66, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag66(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag67, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag67, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag67, false, nil
	}
	if r == '_' {
		return parseTag67, false, nil
	}
	if r == '.' {
		return parseTag67, false, nil
	}
	if r == '-' {
		return parseTag67, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag67(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag68, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag68, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag68, false, nil
	}
	if r == '_' {
		return parseTag68, false, nil
	}
	if r == '.' {
		return parseTag68, false, nil
	}
	if r == '-' {
		return parseTag68, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag68(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag69, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag69, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag69, false, nil
	}
	if r == '_' {
		return parseTag69, false, nil
	}
	if r == '.' {
		return parseTag69, false, nil
	}
	if r == '-' {
		return parseTag69, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag69(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag70, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag70, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag70, false, nil
	}
	if r == '_' {
		return parseTag70, false, nil
	}
	if r == '.' {
		return parseTag70, false, nil
	}
	if r == '-' {
		return parseTag70, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag70(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag71, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag71, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag71, false, nil
	}
	if r == '_' {
		return parseTag71, false, nil
	}
	if r == '.' {
		return parseTag71, false, nil
	}
	if r == '-' {
		return parseTag71, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag71(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag72, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag72, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag72, false, nil
	}
	if r == '_' {
		return parseTag72, false, nil
	}
	if r == '.' {
		return parseTag72, false, nil
	}
	if r == '-' {
		return parseTag72, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag72(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag73, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag73, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag73, false, nil
	}
	if r == '_' {
		return parseTag73, false, nil
	}
	if r == '.' {
		return parseTag73, false, nil
	}
	if r == '-' {
		return parseTag73, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag73(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag74, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag74, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag74, false, nil
	}
	if r == '_' {
		return parseTag74, false, nil
	}
	if r == '.' {
		return parseTag74, false, nil
	}
	if r == '-' {
		return parseTag74, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag74(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag75, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag75, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag75, false, nil
	}
	if r == '_' {
		return parseTag75, false, nil
	}
	if r == '.' {
		return parseTag75, false, nil
	}
	if r == '-' {
		return parseTag75, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag75(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag76, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag76, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag76, false, nil
	}
	if r == '_' {
		return parseTag76, false, nil
	}
	if r == '.' {
		return parseTag76, false, nil
	}
	if r == '-' {
		return parseTag76, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag76(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag77, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag77, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag77, false, nil
	}
	if r == '_' {
		return parseTag77, false, nil
	}
	if r == '.' {
		return parseTag77, false, nil
	}
	if r == '-' {
		return parseTag77, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag77(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag78, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag78, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag78, false, nil
	}
	if r == '_' {
		return parseTag78, false, nil
	}
	if r == '.' {
		return parseTag78, false, nil
	}
	if r == '-' {
		return parseTag78, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag78(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag79, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag79, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag79, false, nil
	}
	if r == '_' {
		return parseTag79, false, nil
	}
	if r == '.' {
		return parseTag79, false, nil
	}
	if r == '-' {
		return parseTag79, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag79(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag80, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag80, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag80, false, nil
	}
	if r == '_' {
		return parseTag80, false, nil
	}
	if r == '.' {
		return parseTag80, false, nil
	}
	if r == '-' {
		return parseTag80, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag80(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag81, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag81, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag81, false, nil
	}
	if r == '_' {
		return parseTag81, false, nil
	}
	if r == '.' {
		return parseTag81, false, nil
	}
	if r == '-' {
		return parseTag81, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag81(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag82, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag82, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag82, false, nil
	}
	if r == '_' {
		return parseTag82, false, nil
	}
	if r == '.' {
		return parseTag82, false, nil
	}
	if r == '-' {
		return parseTag82, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag82(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag83, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag83, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag83, false, nil
	}
	if r == '_' {
		return parseTag83, false, nil
	}
	if r == '.' {
		return parseTag83, false, nil
	}
	if r == '-' {
		return parseTag83, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag83(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag84, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag84, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag84, false, nil
	}
	if r == '_' {
		return parseTag84, false, nil
	}
	if r == '.' {
		return parseTag84, false, nil
	}
	if r == '-' {
		return parseTag84, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag84(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag85, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag85, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag85, false, nil
	}
	if r == '_' {
		return parseTag85, false, nil
	}
	if r == '.' {
		return parseTag85, false, nil
	}
	if r == '-' {
		return parseTag85, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag85(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag86, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag86, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag86, false, nil
	}
	if r == '_' {
		return parseTag86, false, nil
	}
	if r == '.' {
		return parseTag86, false, nil
	}
	if r == '-' {
		return parseTag86, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag86(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag87, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag87, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag87, false, nil
	}
	if r == '_' {
		return parseTag87, false, nil
	}
	if r == '.' {
		return parseTag87, false, nil
	}
	if r == '-' {
		return parseTag87, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag87(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag88, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag88, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag88, false, nil
	}
	if r == '_' {
		return parseTag88, false, nil
	}
	if r == '.' {
		return parseTag88, false, nil
	}
	if r == '-' {
		return parseTag88, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag88(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag89, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag89, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag89, false, nil
	}
	if r == '_' {
		return parseTag89, false, nil
	}
	if r == '.' {
		return parseTag89, false, nil
	}
	if r == '-' {
		return parseTag89, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag89(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag90, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag90, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag90, false, nil
	}
	if r == '_' {
		return parseTag90, false, nil
	}
	if r == '.' {
		return parseTag90, false, nil
	}
	if r == '-' {
		return parseTag90, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag90(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag91, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag91, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag91, false, nil
	}
	if r == '_' {
		return parseTag91, false, nil
	}
	if r == '.' {
		return parseTag91, false, nil
	}
	if r == '-' {
		return parseTag91, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag91(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag92, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag92, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag92, false, nil
	}
	if r == '_' {
		return parseTag92, false, nil
	}
	if r == '.' {
		return parseTag92, false, nil
	}
	if r == '-' {
		return parseTag92, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag92(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag93, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag93, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag93, false, nil
	}
	if r == '_' {
		return parseTag93, false, nil
	}
	if r == '.' {
		return parseTag93, false, nil
	}
	if r == '-' {
		return parseTag93, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag93(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag94, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag94, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag94, false, nil
	}
	if r == '_' {
		return parseTag94, false, nil
	}
	if r == '.' {
		return parseTag94, false, nil
	}
	if r == '-' {
		return parseTag94, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag94(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag95, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag95, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag95, false, nil
	}
	if r == '_' {
		return parseTag95, false, nil
	}
	if r == '.' {
		return parseTag95, false, nil
	}
	if r == '-' {
		return parseTag95, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag95(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag96, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag96, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag96, false, nil
	}
	if r == '_' {
		return parseTag96, false, nil
	}
	if r == '.' {
		return parseTag96, false, nil
	}
	if r == '-' {
		return parseTag96, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag96(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag97, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag97, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag97, false, nil
	}
	if r == '_' {
		return parseTag97, false, nil
	}
	if r == '.' {
		return parseTag97, false, nil
	}
	if r == '-' {
		return parseTag97, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag97(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag98, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag98, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag98, false, nil
	}
	if r == '_' {
		return parseTag98, false, nil
	}
	if r == '.' {
		return parseTag98, false, nil
	}
	if r == '-' {
		return parseTag98, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag98(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag99, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag99, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag99, false, nil
	}
	if r == '_' {
		return parseTag99, false, nil
	}
	if r == '.' {
		return parseTag99, false, nil
	}
	if r == '-' {
		return parseTag99, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag99(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag100, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag100, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag100, false, nil
	}
	if r == '_' {
		return parseTag100, false, nil
	}
	if r == '.' {
		return parseTag100, false, nil
	}
	if r == '-' {
		return parseTag100, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag100(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag101, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag101, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag101, false, nil
	}
	if r == '_' {
		return parseTag101, false, nil
	}
	if r == '.' {
		return parseTag101, false, nil
	}
	if r == '-' {
		return parseTag101, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag101(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag102, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag102, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag102, false, nil
	}
	if r == '_' {
		return parseTag102, false, nil
	}
	if r == '.' {
		return parseTag102, false, nil
	}
	if r == '-' {
		return parseTag102, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag102(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag103, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag103, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag103, false, nil
	}
	if r == '_' {
		return parseTag103, false, nil
	}
	if r == '.' {
		return parseTag103, false, nil
	}
	if r == '-' {
		return parseTag103, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag103(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag104, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag104, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag104, false, nil
	}
	if r == '_' {
		return parseTag104, false, nil
	}
	if r == '.' {
		return parseTag104, false, nil
	}
	if r == '-' {
		return parseTag104, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag104(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag105, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag105, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag105, false, nil
	}
	if r == '_' {
		return parseTag105, false, nil
	}
	if r == '.' {
		return parseTag105, false, nil
	}
	if r == '-' {
		return parseTag105, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag105(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag106, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag106, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag106, false, nil
	}
	if r == '_' {
		return parseTag106, false, nil
	}
	if r == '.' {
		return parseTag106, false, nil
	}
	if r == '-' {
		return parseTag106, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag106(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag107, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag107, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag107, false, nil
	}
	if r == '_' {
		return parseTag107, false, nil
	}
	if r == '.' {
		return parseTag107, false, nil
	}
	if r == '-' {
		return parseTag107, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag107(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag108, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag108, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag108, false, nil
	}
	if r == '_' {
		return parseTag108, false, nil
	}
	if r == '.' {
		return parseTag108, false, nil
	}
	if r == '-' {
		return parseTag108, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag108(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag109, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag109, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag109, false, nil
	}
	if r == '_' {
		return parseTag109, false, nil
	}
	if r == '.' {
		return parseTag109, false, nil
	}
	if r == '-' {
		return parseTag109, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag109(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag110, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag110, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag110, false, nil
	}
	if r == '_' {
		return parseTag110, false, nil
	}
	if r == '.' {
		return parseTag110, false, nil
	}
	if r == '-' {
		return parseTag110, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag110(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag111, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag111, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag111, false, nil
	}
	if r == '_' {
		return parseTag111, false, nil
	}
	if r == '.' {
		return parseTag111, false, nil
	}
	if r == '-' {
		return parseTag111, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag111(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag112, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag112, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag112, false, nil
	}
	if r == '_' {
		return parseTag112, false, nil
	}
	if r == '.' {
		return parseTag112, false, nil
	}
	if r == '-' {
		return parseTag112, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag112(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag113, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag113, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag113, false, nil
	}
	if r == '_' {
		return parseTag113, false, nil
	}
	if r == '.' {
		return parseTag113, false, nil
	}
	if r == '-' {
		return parseTag113, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag113(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag114, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag114, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag114, false, nil
	}
	if r == '_' {
		return parseTag114, false, nil
	}
	if r == '.' {
		return parseTag114, false, nil
	}
	if r == '-' {
		return parseTag114, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag114(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag115, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag115, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag115, false, nil
	}
	if r == '_' {
		return parseTag115, false, nil
	}
	if r == '.' {
		return parseTag115, false, nil
	}
	if r == '-' {
		return parseTag115, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag115(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag116, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag116, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag116, false, nil
	}
	if r == '_' {
		return parseTag116, false, nil
	}
	if r == '.' {
		return parseTag116, false, nil
	}
	if r == '-' {
		return parseTag116, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag116(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag117, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag117, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag117, false, nil
	}
	if r == '_' {
		return parseTag117, false, nil
	}
	if r == '.' {
		return parseTag117, false, nil
	}
	if r == '-' {
		return parseTag117, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag117(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag118, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag118, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag118, false, nil
	}
	if r == '_' {
		return parseTag118, false, nil
	}
	if r == '.' {
		return parseTag118, false, nil
	}
	if r == '-' {
		return parseTag118, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag118(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag119, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag119, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag119, false, nil
	}
	if r == '_' {
		return parseTag119, false, nil
	}
	if r == '.' {
		return parseTag119, false, nil
	}
	if r == '-' {
		return parseTag119, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag119(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag120, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag120, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag120, false, nil
	}
	if r == '_' {
		return parseTag120, false, nil
	}
	if r == '.' {
		return parseTag120, false, nil
	}
	if r == '-' {
		return parseTag120, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag120(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag121, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag121, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag121, false, nil
	}
	if r == '_' {
		return parseTag121, false, nil
	}
	if r == '.' {
		return parseTag121, false, nil
	}
	if r == '-' {
		return parseTag121, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag121(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag122, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag122, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag122, false, nil
	}
	if r == '_' {
		return parseTag122, false, nil
	}
	if r == '.' {
		return parseTag122, false, nil
	}
	if r == '-' {
		return parseTag122, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag122(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag123, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag123, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag123, false, nil
	}
	if r == '_' {
		return parseTag123, false, nil
	}
	if r == '.' {
		return parseTag123, false, nil
	}
	if r == '-' {
		return parseTag123, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag123(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag124, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag124, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag124, false, nil
	}
	if r == '_' {
		return parseTag124, false, nil
	}
	if r == '.' {
		return parseTag124, false, nil
	}
	if r == '-' {
		return parseTag124, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag124(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag125, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag125, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag125, false, nil
	}
	if r == '_' {
		return parseTag125, false, nil
	}
	if r == '.' {
		return parseTag125, false, nil
	}
	if r == '-' {
		return parseTag125, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag125(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag126, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag126, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag126, false, nil
	}
	if r == '_' {
		return parseTag126, false, nil
	}
	if r == '.' {
		return parseTag126, false, nil
	}
	if r == '-' {
		return parseTag126, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag126(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag127, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag127, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag127, false, nil
	}
	if r == '_' {
		return parseTag127, false, nil
	}
	if r == '.' {
		return parseTag127, false, nil
	}
	if r == '-' {
		return parseTag127, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
func parseTag127(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseTag128, false, nil
	}
	if r >= 'A' && r <= 'Z' {
		return parseTag128, false, nil
	}
	if r >= '0' && r <= '9' {
		return parseTag128, false, nil
	}
	if r == '_' {
		return parseTag128, false, nil
	}
	if r == '.' {
		return parseTag128, false, nil
	}
	if r == '-' {
		return parseTag128, false, nil
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}

func parseTag128(r rune) (interface{}, bool, error) {
	if r >= 'a' && r <= 'z' {
		return parseImage, false, ErrEndExpected
	}
	if r >= 'A' && r <= 'Z' {
		return parseImage, false, ErrEndExpected
	}
	if r >= '0' && r <= '9' {
		return parseImage, false, ErrEndExpected
	}
	if r == '_' {
		return parseImage, false, ErrEndExpected
	}
	if r == '.' {
		return parseImage, false, ErrEndExpected
	}
	if r == '-' {
		return parseImage, false, ErrEndExpected
	}
	return parseImage, false, ErrInvalid("[a-zA-Z_-.0-9]", r)
}
