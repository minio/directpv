// // This file is part of MinIO Direct CSI
// // Copyright (c) 2021 MinIO, Inc.
// //
// // This program is free software: you can redistribute it and/or modify
// // it under the terms of the GNU Affero General Public License as published by
// // the Free Software Foundation, either version 3 of the License, or
// // (at your option) any later version.
// //
// // This program is distributed in the hope that it will be useful,
// // but WITHOUT ANY WARRANTY; without even the implied warranty of
// // MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// // GNU Affero General Public License for more details.
// //
// // You should have received a copy of the GNU Affero General Public License
// // along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
)

func bindata_read(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	return buf.Bytes(), nil
}

var _home_bala_golang_gopath_src_github_com_minio_direct_csi_config_crd_direct_csi_min_io_directcsidrives_yaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x5d\x6d\x6f\xe3\xb8\xf1\x7f\xef\x4f\x31\xf0\xff\x0f\x6c\xb2\xb5\x94\x4d\xb6\xd8\xde\x19\x58\x2c\xb6\xd9\xa6\x08\xee\xf6\x36\xb8\xe4\xee\x45\xe3\xb4\x37\x96\xc6\x36\x2f\x12\xa9\x23\x29\x27\xbe\xa2\xdf\xbd\x18\x52\xf2\xa3\xa4\xd8\x69\xb7\xf7\x00\xf2\x55\xc4\x87\xe1\x70\x9e\xc9\x5f\x00\xf7\xa2\x28\xea\x61\x21\xbe\x27\x6d\x84\x92\x43\xc0\x42\xd0\xa3\x25\xc9\x5f\x26\xbe\xff\xc2\xc4\x42\x9d\xcc\x4f\x7b\xf7\x42\xa6\x43\x38\x2f\x8d\x55\xf9\xb7\x64\x54\xa9\x13\xfa\x40\x13\x21\x85\x15\x4a\xf6\x72\xb2\x98\xa2\xc5\x61\x0f\x00\xa5\x54\x16\xb9\xdb\xf0\x27\x40\xa2\xa4\xd5\x2a\xcb\x48\x47\x53\x92\xf1\x7d\x39\xa6\x71\x29\xb2\x94\xb4\x23\x5e\x6f\x3d\x7f\x15\xbf\x89\x4f\x7b\x00\x89\x26\xb7\xfc\x46\xe4\x64\x2c\xe6\xc5\x10\x64\x99\x65\x3d\x00\x89\x39\x0d\x21\x15\x9a\x12\x9b\x18\x91\x6a\x31\x27\x13\xfb\xef\x38\x31\x22\xce\x85\x8c\x85\xea\x99\x82\x12\xde\x7b\xaa\x55\x59\xd4\x0b\xd6\x27\x78\x52\x15\x7f\xfe\x6c\x1f\xdc\xa4\xf3\xeb\xcb\x0f\x4c\xd5\x0d\x64\xc2\xd8\xaf\x1a\x06\xbf\x16\xc6\xba\x09\x45\x56\x6a\xcc\x76\x38\x72\x63\x46\xc8\x69\x99\xa1\xde\x1e\xed\x01\x98\x44\x15\x34\x84\xf3\xac\x34\x96\x74\x0f\xa0\x92\x81\xe3\x27\xaa\x4e\x39\x3f\xc5\xac\x98\xe1\xa9\x27\x96\xcc\x28\x47\xcf\x2e\x80\x2a\x48\xbe\xbf\xba\xfc\xfe\xf5\xf5\x46\x37\x40\xa1\x55\x41\xda\x8a\xfa\x64\xbe\xad\xe9\x77\xad\x17\x20\x25\x93\x68\x51\x58\x27\xfd\x17\x4c\xd0\xcf\x82\x94\x15\x4b\x06\xec\x8c\x6a\xd6\x28\xad\x78\x00\x35\x01\x3b\x13\x06\x34\x15\x9a\x0c\x49\xaf\xea\x0d\xc2\xc0\x93\x50\x82\x1a\xff\xc8\x72\x87\x6b\xd2\x4c\x06\xcc\x4c\x95\x59\xca\xf6\x30\x27\x6d\x41\x53\xa2\xa6\x52\xfc\xbc\xa4\x6d\xc0\x2a\xb7\x69\x86\x96\x2a\x11\xaf\x9a\x90\x96\xb4\xc4\x0c\xe6\x98\x95\x34\x00\x94\x29\xe4\xb8\x00\x4d\xbc\x0b\x94\x72\x8d\x9e\x9b\x62\x62\xf8\xa8\x34\x81\x90\x13\x35\x84\x99\xb5\x85\x19\x9e\x9c\x4c\x85\xad\xed\x3a\x51\x79\x5e\x4a\x61\x17\x27\xce\x44\xc5\xb8\xb4\x4a\x9b\x93\x94\xe6\x94\x9d\x18\x31\x8d\x50\x27\x33\x61\x29\xb1\xa5\xa6\x13\x2c\x44\xe4\x58\x97\xce\xb6\xe3\x3c\xfd\x3f\x5d\x79\x82\x79\xb1\xc1\xab\x5d\xb0\x7a\x8d\xd5\x42\x4e\xd7\x06\x9c\x9d\x75\x68\x80\x4d\x0d\x84\x01\xac\x96\xfa\x53\xac\x04\xcd\x5d\x2c\x9d\x6f\xff\x72\x7d\x03\xf5\xd6\x4e\x19\xdb\xd2\x77\x72\x5f\x2d\x34\x2b\x15\xb0\xc0\x84\x9c\x90\xf6\x4a\x9c\x68\x95\x3b\x9a\x24\xd3\x42\x09\x69\xdd\x47\x92\x09\x92\xdb\xe2\x37\xe5\x38\x17\x96\xf5\xfe\x53\x49\xc6\xb2\xae\x62\x38\x77\xce\x0e\x63\x82\xb2\x48\xd1\x52\x1a\xc3\xa5\x84\x73\xcc\x29\x3b\x47\x43\x9f\x5d\x01\x2c\x69\x13\xb1\x60\xf7\x53\xc1\x7a\x9c\xda\x9e\xec\xa5\xb6\x36\x50\x47\x91\x55\x6b\xf6\x2f\xa7\xc9\x3a\x40\x7c\x7a\x90\x94\x6e\x8f\x6e\x69\x9a\x45\x28\x34\xa5\x3b\xb3\x3c\x23\x63\xa5\x32\xc2\x6d\x97\x72\xc1\xe3\x06\x85\xb4\xbb\xd4\x31\x4d\x5d\x1c\xc6\xec\xaa\x95\xc3\x0e\xa9\x74\x4a\x81\x5b\xa5\x73\x4a\x2f\x94\xce\xb1\x81\x81\x76\xc1\x70\x9b\x88\x8c\xcc\xc2\x58\xca\x9b\x46\x9f\x60\x0b\x60\xa2\x74\x42\x5d\x2b\x9b\x05\xc6\x2d\x57\xa5\xb4\x9f\x8a\xb5\x64\xb4\xdd\x84\xa5\xbc\x65\xe8\x49\xc6\xea\x09\xa8\x35\x2e\x1a\xc7\x1f\x23\xce\x76\x5a\x92\x25\x13\x71\x3a\x89\xaa\x15\x56\xe5\x22\x69\x63\xd8\x79\xe2\xb3\x44\x55\x94\x7a\xfa\x2c\x51\xb5\x2a\xbf\xb6\xd5\x4d\xa2\xd1\x96\xc1\xef\xe5\x4e\x16\x6d\x69\xf6\x75\x28\xcc\x32\x95\x70\x44\x39\xc7\x02\x13\x61\x17\xbb\xa7\x9a\x78\x63\xe4\xc4\xf0\xe6\x8f\x2d\x27\xe2\xa4\x31\x75\x39\x76\xbd\x25\x4a\x7a\x87\x69\xd0\x7c\xab\x41\x6c\xb8\x70\xff\xbc\x26\xe1\xca\x1b\x14\xd2\x40\x4a\x16\x45\x66\x98\x2f\x50\x92\x00\x39\x80\x58\x9f\x30\x09\x92\x52\xeb\xdd\xa8\xba\x12\x0d\x2d\x33\xeb\xfb\xab\x4b\xa8\x6b\xac\x18\xa2\x28\x82\x1b\xee\x36\x56\x97\x89\xe5\x04\xc1\x87\x92\x29\xa5\x6e\x27\xaf\x88\x46\xb2\xa5\x61\x26\x38\x13\x3b\x0b\x05\xf4\xe1\x7d\x22\x28\x4b\xa1\x40\x3b\x83\xd8\x2b\x25\x5e\x09\x24\x06\xb8\x50\x1a\xe8\x11\xf3\x22\xa3\x41\xab\x29\xc1\x85\x52\xd7\x6e\x71\xc5\xd8\x3f\xdd\xd0\xc9\x09\x7c\xbb\x4c\x3b\x6e\x37\x35\x36\xa4\xe7\xbe\x1e\x74\x75\x41\x23\xc9\x89\x52\x2f\x4c\x2d\x23\x2f\x8f\xb8\x26\xf8\x95\x54\x0f\xb2\x89\x55\xc7\x07\xea\x16\x83\x1f\xf5\xdf\xcf\x51\x64\x38\xce\x68\xd4\x1f\xc0\xa8\x7f\xa5\xd5\x54\x93\xe1\xc2\x8c\x3b\xb8\x7e\x18\xf5\x3f\xd0\x54\x63\x4a\xe9\xa8\x5f\x6f\xf7\x87\x02\x6d\x32\xfb\x48\x7a\x4a\x5f\xd1\xe2\x2d\x6f\xd2\x4c\x7f\x63\xfe\xb5\xd5\x68\x69\xba\x78\x9b\xf3\xc2\x25\x2d\xf6\xf9\x9b\x45\x41\x6f\x73\x2c\x36\x3a\x3f\x62\xf1\x34\xf5\xa5\x91\x19\xb8\xbd\xe3\xdc\x35\x3f\x8d\x57\x86\xf7\xc3\x8f\x46\xc9\xe1\xa8\xbf\x92\xc8\x40\xe5\x6c\xbe\x85\x5d\x8c\xfa\x8d\x54\x37\x58\x1d\x8e\xfa\x8e\xd9\x51\x1f\x36\x8e\x3c\x1c\xf5\x99\x2d\xee\xd6\xca\xaa\x71\x39\x19\x8e\xfa\xe3\x85\x25\x33\x38\x1d\x68\x2a\x06\x5c\xa0\xbe\x5d\xed\x3a\xea\xff\xd0\x7c\x04\x59\x9f\x58\xd9\x19\x69\x6f\x77\x06\xfe\xd5\xc4\x5a\x77\x02\x01\xc8\xd0\xd8\x1b\x8d\xd2\x88\xfa\x66\xd0\x16\xb3\x37\xdc\x74\x77\x19\xfb\x8f\x2f\x31\x8d\x05\xcb\x1d\xce\x39\xeb\xc3\xb4\x10\x05\xb0\x4b\x2a\xec\x77\x5c\x36\xb1\x8b\x7b\x9b\xe4\xb2\x15\xa5\x3b\x64\x5c\xf9\xaa\xaf\x74\xc7\x04\x0f\x33\xea\x20\x3a\x23\x28\x65\x4a\x3a\x5b\x70\x71\x97\xac\x62\xca\x0c\xe5\x94\xab\x29\xb8\xe4\xa0\x80\xce\xed\xb9\xd2\xba\x67\x5f\x18\xf0\xc2\x76\xaa\xa5\xa9\x2b\x45\x77\x3e\xe6\xc0\x7d\x71\x5c\xf1\xbe\x5f\x91\x77\xc5\x66\x92\x50\x61\xd9\x49\xe2\x16\x82\x75\x98\xe5\xfa\x2e\x62\x8a\xcf\x4d\x96\x39\x19\x83\x6d\xe9\x69\x4b\x71\xd5\x5c\x5f\x0e\xcf\xca\x1c\x25\x68\xc2\x94\xf9\x5c\x8d\xc9\x54\x24\x68\xdb\xb6\xf3\x34\x7d\x48\xc6\xb1\x2a\x7d\xf0\x5b\xe9\xb1\x52\x15\x57\xc4\x63\xe2\x20\xe9\x1c\xa7\x3a\x40\x9b\x30\x72\x7c\xfc\x9a\xe4\xd4\xce\x86\xf0\xfa\xec\x4f\x6f\xbe\x78\xae\x2c\x7c\x54\xa4\xf4\xaf\x24\x49\xbb\xe0\xb8\x97\x58\x76\x97\xad\x55\xf9\xee\x7c\x71\x5d\xe2\xc6\xd3\xe5\x9c\x0e\xfb\xab\x52\xc2\xca\xf2\x1e\xd0\x80\x21\x0b\x63\x34\x94\x42\x59\xb0\x9c\x38\x21\x08\x69\x2c\xca\x84\x06\x20\x26\x87\x6d\x22\x96\x71\x3d\x5b\xc0\xe9\xd9\x00\xc6\x95\x2a\x76\x23\xfa\xed\xe3\x5d\xbc\x7b\xc4\x2e\xca\x5f\x0e\xb6\xf8\x17\x06\x58\xd5\x6a\xe2\xec\x15\x1e\x84\x9d\xf1\x5d\xc9\x65\xe2\xea\x76\xd9\x95\x89\x61\x33\x1b\xd3\xf2\xdc\x4f\x79\x47\x73\x11\xe2\x5b\x2e\xa4\xc8\xcb\x7c\x08\xaf\x3a\xcd\xa5\xb9\x56\xf1\x4d\x13\x9a\x3d\x6d\xc4\x4f\x5d\x95\x25\xc8\xc1\x75\xaa\x31\xcf\xd1\x8a\x04\x44\xca\xf7\xa7\x89\x20\xbd\x8f\x03\xb1\x08\x2a\x82\x5c\x6c\x6c\xc8\xfa\x85\xa9\xa2\xe8\x9a\x4b\x5d\x69\x95\x96\x09\xe9\xed\x2b\xe9\xaa\xa9\x89\xbb\x58\x89\x89\x48\xd6\xd4\xe6\x2e\x72\xce\x17\xfd\xe3\x03\xd0\x23\xab\x6c\x79\x95\xe7\x6c\xdd\x4a\x32\x27\x94\x42\x4e\x4d\xc5\x22\xdf\x6b\x39\xcc\xf9\x14\xff\x30\x23\x97\x7d\xdc\x63\x46\x45\x4b\xbb\x53\x18\x91\x52\xd3\x2d\xac\x6e\x08\xd3\x12\x35\x4a\x4b\x94\x72\xf0\xe4\x80\x51\xd1\x58\x0b\xf0\xb8\xba\xee\x3e\x11\x3b\xc0\x07\x1c\x1f\x82\xf9\xa8\xd5\xd5\xd9\xc5\x9d\x3d\x02\xce\xe9\xab\xb3\x0e\x0b\x5b\xce\x6a\x99\x52\xa0\xb5\xa4\xe5\x10\xfe\x7e\xfb\x3e\xfa\x1b\x46\x3f\xdf\x1d\x55\x7f\xbc\x8a\xbe\xfc\xc7\x60\x78\xf7\x72\xed\xf3\xee\xf8\xdd\xff\x3f\x37\xb4\x35\xd5\xf9\xab\xb6\x61\xaa\x55\xfa\xac\x2b\xe4\xda\x1a\x06\x2e\xb7\xaa\x09\xdc\xe8\x92\x06\x70\x81\x99\xa1\x01\x7c\x27\x5d\xf2\x6b\x13\x14\xc9\xb2\xe5\x7a\xc9\xd7\x95\x3e\x93\x6a\xae\x89\xdc\xb0\xdb\xa3\x7d\xbc\xda\xfb\x3f\xba\x26\xee\x23\x10\x57\xd1\xaa\xc9\x7a\x3c\x5b\x7b\x4e\x01\x17\x87\xb9\x56\x8e\xab\xfa\x3c\x4e\x54\x7e\xb2\x7a\x6e\x69\x35\x3c\xbe\x44\x7c\x44\xb9\x80\x55\xb0\xf5\xd5\xf3\xb6\x47\x18\xcb\xf5\x37\x26\x5a\x19\xb3\x7c\x63\x6a\x77\xe6\x4c\xdc\x13\x2c\xcb\x6c\x1f\xda\xc7\x94\xa0\xbb\x79\xe8\xb1\xb0\x1a\xf5\x62\xed\xba\x05\x09\x4a\xf7\x5a\x64\x68\x52\x66\xad\x64\x8f\x0c\x11\xc4\x52\xa5\xb4\x9b\x23\x8e\x7d\xc4\xc7\xb1\xc8\x84\x5d\x70\x4c\x4f\x29\x51\x72\x92\x09\x77\x39\x6a\x4f\x16\x79\xa1\xb4\x45\x69\xbd\x1b\x6b\x9a\xd2\x23\x08\x0b\x39\x97\xbe\x64\x38\x71\x1c\xa5\xd2\x9c\x9e\x9e\xbd\xbe\x2e\xc7\xa9\xca\x51\xc8\x8b\xdc\x9e\x1c\xbf\x3b\xfa\xa9\xc4\x8c\x23\x66\xfa\x0d\xe6\x74\x91\xdb\xe3\x3d\x8a\x83\xd3\x37\x4f\xfa\xe1\xd1\xad\xf7\xb6\xbb\xa3\xdb\xa8\xfa\xeb\x65\xdd\x75\xfc\xee\x68\x14\x77\x8e\x1f\xbf\x64\xd6\xd6\x7c\xf8\xee\x36\x5a\x39\x70\x7c\xf7\xf2\xf8\xdd\xda\xd8\xf1\x33\xdd\xb9\xf9\xfa\xef\x5b\xd4\x50\x5e\x37\x4e\xab\x0a\xb6\xc6\x31\x9f\x5c\x1a\x87\xbc\xea\x1b\x87\x5a\xae\x4d\x1d\x4f\x58\xdd\x6f\x35\xbb\xef\x34\x39\x16\xd1\x3d\x2d\x1a\xe2\x58\xcb\xee\x6d\x4f\x3d\x39\x16\x4d\x2f\x79\xd7\x2d\x51\xb2\x43\x1f\x5d\xcf\x68\x5d\xcb\x34\xd1\xe7\x78\x44\xc9\xd4\x54\x24\x98\xfd\x39\x53\xc9\xfd\xb5\xf8\xb9\x21\xc0\x3d\x9f\x76\xae\x52\xca\xbe\x29\xf3\x31\xe9\x83\xce\xda\xfd\xde\xd7\xfa\xb4\xb3\xc7\xbb\xe8\xbe\x76\xd3\xf1\xbe\xd7\xf5\xb6\xd7\xc1\x01\x87\x41\x0e\x3c\x07\x2d\x2a\x50\x5b\xe7\x94\xdf\x34\x65\xc5\x2e\xd1\x17\x68\x67\x87\x6d\x35\x5b\x98\xcf\x66\x08\x5a\x29\x7b\x55\x9f\xe5\x20\xb6\x0c\x69\x81\xcf\xb1\x21\xab\x0a\x95\xa9\x69\x83\xaf\x7c\xee\x67\x76\xab\x2c\x66\xff\x7d\x57\x6d\x7b\xc2\x65\x4d\x3f\xfd\x70\xbb\xbb\x3a\x5a\xc2\x28\x6b\x5d\x5c\xd3\xf7\x5a\x09\xf9\x2b\xdd\x10\xac\x2e\x7d\xe4\x34\x56\x69\x9c\xd2\x10\x26\x5c\x78\x6d\xc0\x9e\x63\xb2\x01\xf5\x5c\xb6\x80\x7a\x06\xd4\x33\xa0\x9e\x01\xf5\xdc\x5d\x19\x50\xcf\xba\xfd\x8e\x50\xcf\x24\x21\x63\x6e\xc4\x81\x25\x4b\x00\x4b\x03\x58\x0a\x01\x2c\x0d\x60\xa9\x6f\x01\x2c\x0d\x60\x69\x00\x4b\x03\x58\xba\x4d\x39\x80\xa5\x10\xc0\xd2\x00\x96\x06\xb0\x34\x80\xa5\x01\x2c\x0d\x60\x69\x00\x4b\x03\x58\x1a\xc0\xd2\x00\x96\x06\xb0\x34\x80\xa5\xcb\xf6\x1b\x04\x4b\xcf\xfc\xa4\x00\x96\x06\xb0\x34\x80\xa5\x01\x2c\x0d\x60\x69\xc3\xca\x00\x96\xd6\x2d\x80\xa5\x01\x2c\x0d\x60\x69\x00\x4b\x03\x58\xea\x5b\x00\x4b\x03\x58\x1a\xc0\xd2\x00\x96\x6e\x53\x0e\x60\x29\x04\xb0\x34\x80\xa5\x01\x2c\x0d\x60\x69\x00\x4b\x03\x58\x1a\xc0\xd2\x00\x96\xb6\x2f\xfb\xee\xbb\xcb\x0f\xbf\x7f\x9c\x15\x7f\x54\xba\x0d\x23\x5b\x23\xfb\xfa\xec\x30\xb2\x42\x7e\x16\xb2\x01\x15\x5e\xb6\xff\x39\x2a\x5c\xad\x3c\xd8\x2d\x02\x9e\x1c\xf0\xe4\x5f\x1c\x4f\x7e\xed\x27\x05\x3c\x39\xe0\xc9\x01\x4f\x0e\x78\x72\xc0\x93\x1b\x56\x06\x3c\xb9\x6e\x01\x4f\x0e\x78\x72\xc0\x93\x03\x9e\x1c\xf0\x64\xdf\x02\x9e\x1c\xf0\xe4\x80\x27\x07\x3c\x79\x9b\x72\xc0\x93\x21\xe0\xc9\x01\x4f\x0e\x78\x72\xc0\x93\x03\x9e\x1c\xf0\xe4\x80\x27\xff\x16\xf0\xe4\xfc\x60\xd4\x2c\x3d\x1c\x0b\x0e\xa8\xf5\x6f\x11\xb5\x46\x63\x0f\x45\x96\xd3\x83\x05\x1e\xb0\xf1\x5f\x2f\x36\x7e\xc3\x89\xf4\xa6\xb1\x5e\xd8\x67\xe5\x33\xb0\xf1\x5f\x02\x8f\xaf\x56\x36\xe5\x95\xae\xe7\xed\x5f\x19\x90\x4f\x98\x7e\x92\x59\x43\xcc\xea\x3a\xc3\x2f\x00\xff\x9b\x07\x2c\x3e\xb5\xee\xd5\xcc\xe6\xef\xee\x5f\x06\x00\x4a\x9a\x93\xb4\x17\xd7\x07\xdb\xab\x5f\x78\xed\xc4\x7f\xd0\xc2\x39\xc9\x54\x1d\xa6\xab\xb9\xd0\xb6\x6c\xdf\xa6\x59\x59\x0f\x0f\xa2\xd5\x93\x1a\x76\xf9\x55\xff\xf7\x84\xeb\x59\xdd\x21\xfd\xfb\xa4\x2f\xbd\x37\x7e\x59\xba\xef\x6f\x6b\xf5\x8f\x45\xbb\xcf\x35\x5c\x07\x6e\xef\x7a\x9e\x2a\xa5\xdf\xd7\x3f\x04\xcd\x9d\xff\x0e\x00\x00\xff\xff\x6f\xe3\x4a\xba\x9d\x7b\x00\x00")

func home_bala_golang_gopath_src_github_com_minio_direct_csi_config_crd_direct_csi_min_io_directcsidrives_yaml() ([]byte, error) {
	return bindata_read(
		_home_bala_golang_gopath_src_github_com_minio_direct_csi_config_crd_direct_csi_min_io_directcsidrives_yaml,
		"home/bala/golang/gopath/src/github.com/minio/direct-csi/config/crd/direct.csi.min.io_directcsidrives.yaml",
	)
}

var _home_bala_golang_gopath_src_github_com_minio_direct_csi_config_crd_direct_csi_min_io_directcsivolumes_yaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x5c\x5d\x8f\xdb\xba\xd1\xbe\xf7\xaf\x18\xf8\x7d\x81\xec\xa6\x96\x1c\x27\x45\x7a\x8e\x81\x20\x08\x36\x4d\x11\xe4\xa4\x08\xb2\xdb\x5c\x74\xbd\xed\x19\x49\x63\x99\x67\x25\x52\x21\x29\x67\x7d\x8a\xfe\xf7\x62\x48\xc9\x92\xd7\x92\xb3\x49\x9b\x3b\xf2\x6a\x4d\x52\xc3\xe1\x7c\x3c\x33\x7c\x2e\x76\x12\x45\xd1\x04\x2b\xf1\x89\xb4\x11\x4a\x2e\x01\x2b\x41\x77\x96\x24\xff\x32\xf1\xed\x4f\x26\x16\x6a\xbe\x5d\x4c\x6e\x85\xcc\x96\x70\x51\x1b\xab\xca\x8f\x64\x54\xad\x53\x7a\x4d\x6b\x21\x85\x15\x4a\x4e\x4a\xb2\x98\xa1\xc5\xe5\x04\x00\xa5\x54\x16\x79\xda\xf0\x4f\x80\x54\x49\xab\x55\x51\x90\x8e\x72\x92\xf1\x6d\x9d\x50\x52\x8b\x22\x23\xed\x84\xb7\x47\x6f\x9f\xc4\xcf\xe3\xc5\x04\x20\xd5\xe4\x3e\xbf\x12\x25\x19\x8b\x65\xb5\x04\x59\x17\xc5\x04\x40\x62\x49\x4b\xc8\x84\xa6\xd4\xa6\x46\x6c\x55\x51\x97\x64\x62\x3f\x11\xa7\x46\xc4\xa5\x90\xb1\x50\x13\x53\x51\xca\x87\xe7\x5a\xd5\x55\xfb\x45\x7f\x83\x97\xd5\x28\xe8\x2f\xf7\xda\x6d\xba\xb8\x7c\xfb\xc9\x89\x75\x2b\x85\x30\xf6\xdd\xd0\xea\x2f\xc2\x58\xb7\xa3\x2a\x6a\x8d\xc5\xb1\x52\x6e\xd1\x08\x99\xd7\x05\xea\xa3\xe5\x09\x80\x49\x55\x45\x4b\xb8\x28\x6a\x63\x49\x4f\x00\x1a\x43\x38\x9d\xa2\xe6\xaa\xdb\x05\x16\xd5\x06\x17\x5e\x5a\xba\xa1\x12\xbd\xca\x00\xaa\x22\xf9\xea\xc3\xdb\x4f\xcf\x2e\x0f\xa6\x01\x2a\xad\x2a\xd2\x56\xb4\xb7\xf3\xa3\xe7\xe4\xde\x2c\x40\x46\x26\xd5\xa2\xb2\xce\x05\x8f\x58\xa0\xdf\x05\x19\x7b\x97\x0c\xd8\x0d\xb5\xaa\x51\xd6\xe8\x00\x6a\x0d\x76\x23\x0c\x68\xaa\x34\x19\x92\xde\xdf\x07\x82\x81\x37\xa1\x04\x95\xfc\xc6\xb6\x87\x4b\xd2\x2c\x06\xcc\x46\xd5\x45\xc6\x41\xb1\x25\x6d\x41\x53\xaa\x72\x29\x7e\xdf\xcb\x36\x60\x95\x3b\xb4\x40\x4b\x8d\x91\xbb\x21\xa4\x25\x2d\xb1\x80\x2d\x16\x35\xcd\x00\x65\x06\x25\xee\x40\x13\x9f\x02\xb5\xec\xc9\x73\x5b\x4c\x0c\xef\x95\x26\x10\x72\xad\x96\xb0\xb1\xb6\x32\xcb\xf9\x3c\x17\xb6\x0d\xee\x54\x95\x65\x2d\x85\xdd\xcd\x5d\x9c\x8a\xa4\xb6\x4a\x9b\x79\x46\x5b\x2a\xe6\x46\xe4\x11\xea\x74\x23\x2c\xa5\xb6\xd6\x34\xc7\x4a\x44\x4e\x75\xe9\x02\x3c\x2e\xb3\xff\xd3\x4d\x3a\x98\x47\x07\xba\xda\x1d\xbb\xd7\x58\x2d\x64\xde\x5b\x70\xb1\x76\xc2\x03\x1c\x6d\x20\x0c\x60\xf3\xa9\xbf\x45\x67\x68\x9e\x62\xeb\x7c\xfc\xf3\xe5\x15\xb4\x47\x3b\x67\xdc\xb7\xbe\xb3\x7b\xf7\xa1\xe9\x5c\xc0\x06\x13\x72\x4d\xda\x3b\x71\xad\x55\xe9\x64\x92\xcc\x2a\x25\xa4\x75\x3f\xd2\x42\x90\xbc\x6f\x7e\x53\x27\xa5\xb0\xec\xf7\xcf\x35\x19\xcb\xbe\x8a\xe1\xc2\x65\x3c\x24\x04\x75\x95\xa1\xa5\x2c\x86\xb7\x12\x2e\xb0\xa4\xe2\x02\x0d\xfd\x70\x07\xb0\xa5\x4d\xc4\x86\x7d\x98\x0b\xfa\x60\x75\x7f\xb3\xb7\x5a\x6f\xc1\x58\xb4\xb5\x39\xdc\x3a\x9c\x61\x3c\x70\x8b\xa2\xc0\xa4\xa0\x0b\xac\x30\x15\x76\x77\x7f\x03\xc0\x5a\xe9\x12\xed\x92\x23\xf9\xf9\x1f\x8f\x56\xbd\x16\x1c\xe5\xb9\x03\x85\xfe\x48\x95\xcc\x44\x0f\x57\xfb\x43\x58\x2a\x07\xa6\xef\x45\xd7\xf4\xa2\x15\xe1\x40\x19\x85\x34\x90\x91\x45\x51\x18\xd6\x0b\x94\x24\x40\xc6\x4e\xeb\x33\x9c\x20\xad\xb5\x3e\x0e\x83\xce\x34\xb4\x87\x82\x57\x1f\xde\x42\x5b\x19\x62\x88\xa2\x08\xae\x78\xda\x58\x5d\xa7\x96\x23\x9a\x2f\x25\x33\xca\xdc\x49\x1e\x0f\x07\xc5\xd6\x86\x95\x60\xe8\x40\xad\x71\x07\xe8\xe3\x71\x2d\xa8\xc8\xa0\x42\xbb\x81\xd8\x3b\x25\xee\x0c\x12\x03\xbc\x51\x1a\xe8\x0e\xcb\xaa\xa0\xd9\xa0\x5c\x36\x2d\xbc\x51\xea\xd2\x7d\xdc\x28\xf6\x2f\xb7\x34\x9f\xc3\xc7\x7d\x9e\xb8\xd3\x54\x62\x48\x6f\x7d\x15\x73\x40\x36\x28\x72\xad\xd4\x23\xd3\xda\xc8\xdb\x23\x6e\x05\xbe\x93\xea\x8b\x1c\x52\xd5\xe9\x81\x9a\x86\xbc\x05\xb0\x9a\xbe\x6a\x63\x68\x35\x9d\xc1\x6a\xfa\x41\xab\x5c\x93\xe1\x52\xc2\x13\x0c\x78\xab\xe9\x6b\xca\x35\x66\x94\xad\xa6\xed\x71\x7f\xa8\xd0\xa6\x9b\xf7\xa4\x73\x7a\x47\xbb\x17\x7c\xc8\xb0\xfc\x83\xfd\x97\x56\xa3\xa5\x7c\xf7\xa2\xe4\x0f\xf7\xb2\xb8\xec\x5d\xed\x2a\x7a\x51\x62\x75\x30\xf9\x1e\xab\xaf\x4b\xdf\x07\x99\x81\xeb\x1b\x4e\xb6\xed\x22\xee\x02\xef\xd7\xdf\x8c\x92\xcb\xd5\xb4\xb3\xc8\x4c\x95\x1c\xbe\x95\xdd\xad\xa6\x83\x52\x0f\x54\x5d\xae\xa6\x4e\xd9\xd5\x14\x0e\xae\xbc\x5c\x4d\x59\x2d\x9e\xd6\xca\xaa\xa4\x5e\x2f\x57\xd3\x64\x67\xc9\xcc\x16\x33\x4d\xd5\x8c\x2b\xea\x8b\xee\xd4\xd5\xf4\xd7\xe1\x2b\xc8\xf6\xc6\xca\x6e\x48\xfb\xb8\x33\xf0\xef\x21\xd5\xc6\x81\xc0\x8f\x02\x8d\xbd\xd2\x28\x8d\x68\xfb\x99\xe1\x7d\xf7\xd2\xf4\xf8\x33\xce\x1f\x5f\x13\x8d\x05\xcb\x13\x2e\x39\xdb\xcb\x8c\x08\x05\xb0\x7b\x29\x9c\x77\x8c\xf3\x9c\xe2\x3e\x26\xb9\xce\xa2\x74\x97\x8c\x9b\x5c\xf5\xa5\x39\x21\xf8\xb2\xa1\x13\x42\x37\x04\xb5\xcc\x48\x17\x3b\xae\x46\x69\x87\x29\x1b\x94\x39\xc3\x3f\xbc\x65\x50\x40\x97\xf6\x5c\x1a\x6e\x39\x17\x66\xfc\xe1\xb8\xd4\xda\xb4\xa5\xcd\xdd\x8f\x35\x70\xbf\x18\x57\x7c\xee\x37\xe2\x5d\x75\x4c\x53\xaa\x2c\x27\x49\x3c\x22\xb0\x85\x59\x2e\x48\x11\x4b\x1c\xd9\x37\x52\x23\xba\x51\x92\x31\x98\x3f\xcc\x71\xcd\x5e\x5f\xbf\x37\x75\x89\x12\x34\x61\xc6\x7a\x76\x6b\x32\x13\x29\xda\xb1\xe3\xbc\x4c\x0f\xc9\x98\xa8\xda\x83\x5f\xe7\xc7\xc6\x55\x5c\xc2\x13\x62\x90\x74\x89\xd3\x5c\x60\xcc\x18\x25\xde\xfd\x42\x32\xb7\x9b\x25\x3c\x7b\xfa\xa7\xe7\x3f\x7d\xaf\x2d\x3c\x2a\x52\xf6\x17\x92\xa4\x1d\x38\x3e\xc8\x2c\xc7\x9f\xf5\xda\x12\x77\xbf\xb8\xad\xc9\x71\xbe\xdf\x73\x22\xfe\x9a\x92\xd0\x45\xde\x17\x34\x60\xc8\x42\x82\x86\x32\xa8\x2b\xb6\x13\x17\x04\x21\x8d\x45\x99\xd2\x0c\xc4\xfa\xdb\x0e\x11\x7b\x5c\x2f\x76\xb0\x78\x3a\x83\xa4\x71\xc5\x31\xa2\x5f\xdf\xdd\xc4\xc7\x57\x3c\x25\xf9\xe7\xd9\x3d\xfd\x85\x01\x76\xb5\x5a\xbb\x78\x85\x2f\xc2\x6e\xb8\xb9\x73\x95\xb8\x69\x87\x4f\x55\x62\x38\xac\xc6\xb4\xbf\xf7\xd7\xb2\x63\xb8\x09\xf1\xa3\x14\x52\x94\x75\xb9\x84\x27\x27\xc3\x65\xb8\x57\xf1\x43\x13\x9a\x07\xc6\x88\xdf\xda\xb5\x25\xc8\xe0\x9a\x6b\x2c\x4b\xb4\x22\x05\x91\x71\xc3\xb7\x16\xa4\x1f\x92\x40\x6c\x82\x46\x20\x37\x1b\x07\xb6\x7e\x64\x1a\x14\xed\xa5\xd4\x07\xad\xb2\x3a\x25\x7d\xbf\x87\xee\x86\x5a\x03\x7b\x43\xac\x45\xda\x73\x9b\xeb\x3c\x5d\x2e\xfa\xd7\x12\xd0\x1d\xbb\x6c\xff\xf6\xe0\x6a\x3d\x2a\xb2\x24\x94\x42\xe6\xa6\x51\x91\x1b\x71\x86\x39\x5f\xe2\xbf\x6c\xc8\x55\x1f\xf7\xfa\x6a\x64\x69\x77\x0b\x23\x32\xd2\x34\x2e\x16\x21\xaf\x51\xa3\xb4\x44\x19\x83\x27\x03\x46\x23\xa3\x07\xf0\xd8\xf5\xe7\x5f\xc1\x0e\xf0\x80\xe3\x21\x98\xaf\xda\xf4\xfa\x0e\x77\x1e\x00\x38\x8b\x27\x4f\x4f\x44\xd8\x7e\xd7\xc8\x96\x0a\x2d\x3f\xf8\x96\xf0\x8f\xeb\x57\xd1\xdf\x31\xfa\xfd\xe6\xac\xf9\xe3\x49\xf4\xf3\x3f\x67\xcb\x9b\xc7\xbd\x9f\x37\xe7\x2f\xff\xff\x7b\xa1\x6d\xa8\xcf\xef\xc6\x41\xa8\x36\xe5\xb3\xed\x90\xdb\x68\x98\xb9\xda\xaa\xd6\x70\xa5\xf9\x65\xfa\x06\x0b\x43\x33\xf8\x9b\x74\xc5\x6f\xcc\x50\x24\xeb\x72\xec\xd0\x08\xa6\x2c\x6a\xb8\x27\x72\xcb\xee\x8c\xf1\xf5\xe6\xec\xef\x35\x89\xdb\xf0\x10\x83\xb8\x8e\x56\xad\xfb\x78\xd6\x7b\xff\x81\xc3\x61\xee\x95\xe3\xa6\x3f\x8f\x53\x55\xce\xbb\xf7\xe1\x68\xe0\xf1\x23\xe2\x3d\xca\x1d\x74\x60\xeb\xbb\xe7\xfb\x19\x61\x2c\xf7\xdf\x98\x6a\x65\xcc\xfe\x51\x3c\x9e\xcc\x85\xb8\x25\xd8\xb7\xd9\x1e\xda\x13\x4a\xd1\xbd\x3c\x74\x22\xac\x46\xbd\xeb\x3d\xb7\x20\x45\xe9\x9e\xb7\x86\xd6\x75\x31\x2a\xf6\xcc\x10\x41\x2c\x55\x46\xc7\x35\xe2\xdc\x23\x3e\x26\xa2\x10\x76\xc7\x98\x9e\x51\xaa\xe4\xba\x10\xee\x71\x34\x5e\x2c\xca\x4a\x69\x8b\xd2\xfa\x34\xd6\x94\xd3\x1d\x08\x0b\x25\xb7\xbe\x64\xb8\x70\x9c\x65\xd2\x2c\x16\x4f\x9f\x5d\xd6\x49\xa6\x4a\x14\xf2\x4d\x69\xe7\xe7\x2f\xcf\x3e\xd7\x58\x30\x62\x66\x7f\xc5\x92\xde\x94\xf6\xfc\x01\xcd\xc1\xe2\xf9\x57\xf3\xf0\xec\xda\x67\xdb\xcd\xd9\x75\xd4\xfc\xf5\xb8\x9d\x3a\x7f\x79\xb6\x8a\x4f\xae\x9f\x3f\x66\xd5\x7a\x39\x7c\x73\x1d\x75\x09\x1c\xdf\x3c\x3e\x7f\xd9\x5b\x3b\xff\xce\x74\xd6\xf4\xb9\x16\x9a\xb2\xa1\xe8\x8d\x06\xda\xeb\xc1\x6d\x4d\xc3\x36\xb8\xe6\x8b\xcb\xe0\x92\x77\xfd\xe0\xd2\xc8\xb3\x69\x84\x79\xe8\x2f\xba\x97\xf0\xd1\xda\x5d\x74\x5b\x27\xa4\x25\x59\x32\x11\x3f\xcf\xa2\x12\xab\xe8\x96\x76\x03\x38\x36\x72\xfa\xb1\x08\x7f\x60\x89\xd5\x31\xfb\xc0\x95\x99\xf4\x07\xb4\x9b\x63\xf9\x27\x3c\x92\x69\xb1\x1d\x00\x92\x13\x5f\x6c\x94\xb1\xdf\x7c\x0c\x27\x1e\x87\xfa\x37\x7d\x64\x2c\xe6\x42\xe6\xdf\x7c\x98\x55\x16\x8b\x1f\x41\xf2\xd4\x86\xb2\xff\xbd\xdc\xc1\x10\x3b\xce\x92\x68\xcf\x8d\x4d\x46\xbf\xf4\x7d\xee\x12\xac\xae\x7d\x38\x19\xab\x34\x3f\x90\x60\xcd\xd5\xe8\x80\xbc\x4e\xc8\x06\xee\x7a\x3f\x02\x77\x1d\xb8\xeb\xc0\x5d\x07\xee\x3a\x70\xd7\x81\xbb\x0e\xdc\x75\xe0\xae\x03\x77\x1d\xb8\xeb\xc0\x5d\x43\xe0\xae\x0f\x35\x0b\xdc\x75\xe0\xae\x0f\x46\xe0\xae\x8f\x46\xe0\xae\x03\x77\x0d\x81\xbb\x0e\xdc\x75\xe0\xae\x03\x77\xed\xc7\x8f\xe0\xae\x9f\x06\xee\xba\x19\x81\xbb\x0e\xdc\x75\xe0\xae\x03\x77\x1d\xb8\xeb\xc0\x5d\x07\xee\x3a\x70\xd7\x81\xbb\x0e\xdc\x75\xe0\xae\x21\x70\xd7\x87\x9a\x05\xee\x3a\x70\xd7\x07\x23\x70\xd7\x47\x23\x70\xd7\x81\xbb\x86\xc0\x5d\x07\xee\x3a\x70\xd7\x81\xbb\xf6\xe3\x47\x70\xd7\xcf\x02\x77\xdd\x8c\xc0\x5d\x07\xee\x3a\x70\xd7\x81\xbb\x0e\xdc\x75\xe0\xae\x03\x77\x1d\xb8\xeb\xc0\x5d\x07\xee\x3a\x70\xd7\x10\xb8\xeb\x43\xcd\x02\x77\x1d\xb8\xeb\x83\x11\xb8\xeb\xa3\x11\xb8\xeb\xc0\x5d\x43\xe0\xae\x03\x77\x1d\xb8\xeb\xc0\x5d\xfb\xf1\xdf\x71\xd7\x6e\xa6\xab\xa3\xfe\x8d\xe6\xe1\xe7\xe0\x9f\x73\x4f\x7d\xc5\x6a\xff\xdb\xb6\xfb\xd9\xe3\xb6\xe0\xfa\x66\xe2\xa5\x52\xf6\xa9\xfd\x3f\xda\x3c\xf9\x9f\x00\x00\x00\xff\xff\x93\x4a\x11\x4c\xe1\x5c\x00\x00")

func home_bala_golang_gopath_src_github_com_minio_direct_csi_config_crd_direct_csi_min_io_directcsivolumes_yaml() ([]byte, error) {
	return bindata_read(
		_home_bala_golang_gopath_src_github_com_minio_direct_csi_config_crd_direct_csi_min_io_directcsivolumes_yaml,
		"home/bala/golang/gopath/src/github.com/minio/direct-csi/config/crd/direct.csi.min.io_directcsivolumes.yaml",
	)
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		return f()
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() ([]byte, error){
	"home/bala/golang/gopath/src/github.com/minio/direct-csi/config/crd/direct.csi.min.io_directcsidrives.yaml":  home_bala_golang_gopath_src_github_com_minio_direct_csi_config_crd_direct_csi_min_io_directcsidrives_yaml,
	"home/bala/golang/gopath/src/github.com/minio/direct-csi/config/crd/direct.csi.min.io_directcsivolumes.yaml": home_bala_golang_gopath_src_github_com_minio_direct_csi_config_crd_direct_csi_min_io_directcsivolumes_yaml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for name := range node.Children {
		rv = append(rv, name)
	}
	return rv, nil
}

type _bintree_t struct {
	Func     func() ([]byte, error)
	Children map[string]*_bintree_t
}

var _bintree = &_bintree_t{nil, map[string]*_bintree_t{
	"home": {nil, map[string]*_bintree_t{
		"bala": {nil, map[string]*_bintree_t{
			"golang": {nil, map[string]*_bintree_t{
				"gopath": {nil, map[string]*_bintree_t{
					"src": {nil, map[string]*_bintree_t{
						"github.com": {nil, map[string]*_bintree_t{
							"minio": {nil, map[string]*_bintree_t{
								"direct-csi": {nil, map[string]*_bintree_t{
									"config": {nil, map[string]*_bintree_t{
										"crd": {nil, map[string]*_bintree_t{
											"direct.csi.min.io_directcsidrives.yaml":  {home_bala_golang_gopath_src_github_com_minio_direct_csi_config_crd_direct_csi_min_io_directcsidrives_yaml, map[string]*_bintree_t{}},
											"direct.csi.min.io_directcsivolumes.yaml": {home_bala_golang_gopath_src_github_com_minio_direct_csi_config_crd_direct_csi_min_io_directcsivolumes_yaml, map[string]*_bintree_t{}},
										}},
									}},
								}},
							}},
						}},
					}},
				}},
			}},
		}},
	}},
}}
