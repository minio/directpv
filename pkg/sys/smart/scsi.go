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

package smart

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	// INQ_REPLY_LEN = 96
	INQ_REPLY_LEN = 20

	SGInfoOk     = 0x0 //no sense, host nor driver "noise" or error
	SGInfoOkMask = 0x1 //indicates whether some error or status field is non-zero

	SG_IO             = 0x2285 //scsi generic ioctl command
	SCSI_INQUIRY      = 0x12   // inquiry command
	SG_DXFER_FROM_DEV = -3
)

var (
	SERIALINQUIRY = []byte{0x12, 0x01, 0x80, 0x00, 0x60, 0x00}
)

type CDB6 [6]byte

type SCSIDevice struct {
	Name string
	fd   int
}

func NewSCSIDevice(devName string) *SCSIDevice {
	return &SCSIDevice{devName, -1}
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
	interface_id    int32   // 'S' for SCSI generic (required)
	dxfer_direction int32   // data transfer direction
	cmd_len         uint8   // SCSI command length (<= 16 bytes)
	mx_sb_len       uint8   // max length to write to sbp
	iovec_count     uint16  // 0 implies no scatter gather
	dxfer_len       uint32  // byte count of data transfer
	dxferp          uintptr // points to data transfer memory or scatter gather list
	cmdp            uintptr // points to command to perform
	sbp             uintptr // points to sense_buffer memory
	timeout         uint32  // MAX_UINT -> no timeout (unit: millisec)
	flags           uint32  // 0 -> default, see SG_FLAG...
	pack_id         int32   // unused internally (normally)
	usr_ptr         uintptr // unused internally
	status          uint8   // SCSI status
	masked_status   uint8   // shifted, masked scsi status
	msg_status      uint8   // messaging level data (optional)
	sb_len_wr       uint8   // byte count actually written to sbp
	host_status     uint16  // errors from host adapter
	driver_status   uint16  // errors from software driver
	resid           int32   // dxfer_len - actual_transferred
	duration        uint32  // time taken by cmd (unit: millisec)
	info            uint32  // auxiliary information
}

type InquiryResponse [20]byte

func (d *SCSIDevice) serialInquiry() (InquiryResponse, error) {
	var resp InquiryResponse

	respBuf := make([]byte, INQ_REPLY_LEN)

	cdb := SERIALINQUIRY
	binary.BigEndian.PutUint16(cdb[3:], uint16(len(respBuf)))

	if err := d.sendCDB(cdb[:], &respBuf); err != nil {
		return resp, err
	}

	binary.Read(bytes.NewBuffer(respBuf), NativeEndian, &resp)

	return resp, nil
}

func (d *SCSIDevice) execGenericIO(hdr *sgIoHdr) error {
	if err := Ioctl(uintptr(d.fd), SG_IO, uintptr(unsafe.Pointer(hdr))); err != nil {
		return err
	}

	// See http://www.t10.org/lists/2status.htm for SCSI status codes
	if hdr.info&SGInfoOkMask != SGInfoOk {
		err := sgIOErr{
			scsiStatus:   hdr.status,
			hostStatus:   hdr.host_status,
			driverStatus: hdr.driver_status,
		}
		return err
	}

	return nil
}

func (d *SCSIDevice) sendCDB(cdb []byte, respBuf *[]byte) error {
	senseBuf := make([]byte, 32)

	// Populate required fields of "sg_io_hdr_t" struct
	hdr := sgIoHdr{
		interface_id:    'S',
		dxfer_direction: SG_DXFER_FROM_DEV,
		// timeout:         DEFAULT_TIMEOUT,
		cmd_len:   uint8(len(cdb)),
		mx_sb_len: uint8(len(senseBuf)),
		dxfer_len: uint32(len(*respBuf)),
		dxferp:    uintptr(unsafe.Pointer(&(*respBuf)[0])),
		cmdp:      uintptr(unsafe.Pointer(&cdb[0])),
		sbp:       uintptr(unsafe.Pointer(&senseBuf[0])),
	}

	return d.execGenericIO(&hdr)
}

func (d *SCSIDevice) open() (err error) {
	d.fd, err = unix.Open(d.Name, unix.O_RDWR, 0600)
	return err
}

func (d *SCSIDevice) close() error {
	return unix.Close(d.fd)
}

func (d *SCSIDevice) SerialNumber() (string, error) {
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
