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

package smart

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	// inqReplyLen = 96
	inqReplyLen = 20

	sgInfoOk     = 0x0 // no sense, host nor driver "noise" or error
	sgInfoOkMask = 0x1 // indicates whether some error or status field is non-zero

	sgIO           = 0x2285 // scsi generic ioctl command
	sgDxferFromDev = -3
)

var serialInquiry = []byte{0x12, 0x01, 0x80, 0x00, 0x60, 0x00}

type scsiDevice struct {
	Name string
	fd   int
}

func newSCSIDevice(devName string) *scsiDevice {
	return &scsiDevice{devName, -1}
}

type sgIOErr struct {
	scsiStatus   uint8
	hostStatus   uint16
	driverStatus uint16
}

func (s sgIOErr) Error() string {
	return fmt.Sprintf("SCSI status: %#02x, host status: %#02x, driver status: %#02x",
		s.scsiStatus, s.hostStatus, s.driverStatus)
}

// SCSI generic ioctl header, defined as sg_io_hdr_t in <scsi/sg.h>
type sgIoHdr struct {
	interfaceID    int32   // interfaceID: 'S' for SCSI generic (required)
	dxferDirection int32   // dxferDirection: data transfer direction
	cmdLen         uint8   // cmdLen: SCSI command length (<= 16 bytes)
	mxSbLen        uint8   // mxSbLen: max length to write to sbp
	_              uint16  // iovecCount: 0 implies no scatter gather
	dxferLen       uint32  // dxferLen: byte count of data transfer
	dxferp         uintptr // dxferp: points to data transfer memory or scatter gather list
	cmdp           uintptr // cmdp: points to command to perform
	sbp            uintptr // sbp: points to sense_buffer memory
	_              uint32  // timeout: MAX_UINT -> no timeout (unit: millisec)
	_              uint32  // flags: 0 -> default, see SG_FLAG...
	_              int32   // packID: unused internally (normally)
	_              uintptr // usrPtr: unused internally
	status         uint8   // status: SCSI status
	_              uint8   // maskedStatus: shifted, masked scsi status
	_              uint8   // msgStatus: messaging level data (optional)
	_              uint8   // sbLenWR: byte count actually written to sbp
	hostStatus     uint16  // hostStatus: errors from host adapter
	driverStatus   uint16  // driverStatus: errors from software driver
	_              int32   // resid: dxfer_len - actual_transferred
	_              uint32  // duration: time taken by cmd (unit: millisec)
	info           uint32  // info: auxiliary information
}

type inquiryResponse [20]byte

func (d *scsiDevice) serialInquiry() (inquiryResponse, error) {
	var resp inquiryResponse

	respBuf := make([]byte, inqReplyLen)

	cdb := serialInquiry
	binary.BigEndian.PutUint16(cdb[3:], uint16(len(respBuf)))

	if err := d.sendCDB(cdb[:], &respBuf); err != nil {
		return resp, err
	}

	if err := binary.Read(bytes.NewBuffer(respBuf), nativeEndian, &resp); err != nil {
		return resp, err
	}

	return resp, nil
}

func (d *scsiDevice) execGenericIO(hdr *sgIoHdr) error {
	if err := sysIOCTL(uintptr(d.fd), sgIO, uintptr(unsafe.Pointer(hdr))); err != nil {
		return err
	}

	// See http://www.t10.org/lists/2status.htm for SCSI status codes
	if hdr.info&sgInfoOkMask != sgInfoOk {
		err := sgIOErr{
			scsiStatus:   hdr.status,
			hostStatus:   hdr.hostStatus,
			driverStatus: hdr.driverStatus,
		}
		return err
	}

	return nil
}

func (d *scsiDevice) sendCDB(cdb []byte, respBuf *[]byte) error {
	senseBuf := make([]byte, 32)

	// Populate required fields of "sg_io_hdr_t" struct
	hdr := sgIoHdr{
		interfaceID:    'S',
		dxferDirection: sgDxferFromDev,
		// timeout:         DEFAULT_TIMEOUT,
		cmdLen:   uint8(len(cdb)),
		mxSbLen:  uint8(len(senseBuf)),
		dxferLen: uint32(len(*respBuf)),
		dxferp:   uintptr(unsafe.Pointer(&(*respBuf)[0])),
		cmdp:     uintptr(unsafe.Pointer(&cdb[0])),
		sbp:      uintptr(unsafe.Pointer(&senseBuf[0])),
	}

	return d.execGenericIO(&hdr)
}

func (d *scsiDevice) open() (err error) {
	d.fd, err = unix.Open(d.Name, unix.O_RDWR, 0o600)
	return err
}

func (d *scsiDevice) close() error {
	return unix.Close(d.fd)
}

func (d *scsiDevice) SerialNumber() (string, error) {
	if err := d.open(); err != nil {
		return "", err
	}
	defer d.close()

	inquiryR, err := d.serialInquiry()
	if err != nil {
		return "", err
	}
	return string(inquiryR[4:]), nil
}
