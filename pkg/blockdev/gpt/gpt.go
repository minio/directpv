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

package gpt

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/minio/directpv/pkg/blockdev/parttable"
)

func isUUIDZero(uuid [16]byte) bool {
	for i := range uuid {
		if uuid[i] != 0 {
			return false
		}
	}

	return true
}

// UUID2String converts UUID to string.
func UUID2String(uuid [16]byte) string {
	return fmt.Sprintf(
		"%08x-%04x-%04x-%x-%x",
		binary.LittleEndian.Uint32(uuid[0:4]),
		binary.LittleEndian.Uint16(uuid[4:6]),
		binary.LittleEndian.Uint16(uuid[6:8]),
		uuid[8:10],
		uuid[10:],
	)
}

// Header contains GPT header in LBA 1 as per specification in
// https://en.wikipedia.org/wiki/GUID_Partition_Table#Partition_table_header_(LBA_1)
type Header struct {
	Signature              [8]byte
	Revision               [4]byte
	HeaderSize             uint32 // 4 bytes.
	CRC32                  uint32 // 4 bytes.
	_                      uint32 // 4 bytes. Reserved: must be zero.
	CurrentLBA             uint64 // 8 bytes.
	BackupLBA              uint64 // 8 bytes.
	FirstUsableLBA         uint64 // 8 bytes.
	LastUsableLBA          uint64 // 8 bytes.
	DiskGUID               [16]byte
	PartitionEntryStartLBA uint64    // 8 bytes.
	NumPartitionEntries    uint32    // 4 bytes.
	PartitionEntrySize     uint32    // 4 bytes.
	PartitionArrayCRC32    uint32    // 4 bytes.
	_                      [420]byte // Reserved: must be zero.
}

// Entry contains partition entries (LBA 2-33) as per specification in
// https://en.wikipedia.org/wiki/GUID_Partition_Table#Partition_entries_(LBA_2%E2%80%9333)
type Entry struct {
	TypeGUID      [16]byte
	GUID          [16]byte
	FirstLBA      uint64 // 8 bytes.
	LastLBA       uint64 // 8 bytes.
	AttributeFlag uint64 // 8 bytes.
	Name          [72]byte
}

// Table denotes GPT partition table.
type Table struct {
	Header  Header
	Entries []Entry
}

// Read reads GPT partition table from given reader.
func Read(reader io.Reader) (*Table, error) {
	var header Header
	if err := binary.Read(reader, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	if string(header.Signature[:]) != "EFI PART" {
		return nil, parttable.ErrPartTableNotFound
	}

	// TODO: validate Header.CRC
	// TODO: validate Header.PartitionArrayCRC32

	entrySize := int(header.PartitionEntrySize)
	data := make([]byte, entrySize)
	var entries []Entry
	for i := 0; i < int(header.NumPartitionEntries); i++ {
		if _, err := io.ReadFull(reader, data); err != nil {
			return nil, err
		}
		var entry Entry
		if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, &entry); err != nil {
			return nil, err
		}
		if isUUIDZero(entry.TypeGUID) {
			break
		}
		entries = append(entries, entry)
	}

	return &Table{
		Header:  header,
		Entries: entries,
	}, nil
}

// GPT is interface compatible partition table information.
type GPT struct {
	uuid       string
	partitions map[int]*parttable.Partition
}

// Type returns "gpt"
func (gpt *GPT) Type() string {
	return "gpt"
}

// UUID returns partition table UUID.
func (gpt *GPT) UUID() string {
	return gpt.uuid
}

// Partitions returns list of partitions.
func (gpt *GPT) Partitions() map[int]*parttable.Partition {
	return gpt.partitions
}

// Probe reads and returns GPT partition table.
func Probe(reader io.Reader) (*GPT, error) {
	table, err := Read(reader)
	if err != nil {
		return nil, err
	}
	partitionMap := map[int]*parttable.Partition{}
	for i, entry := range table.Entries {
		partitionMap[i+1] = &parttable.Partition{
			Number: i + 1,
			UUID:   UUID2String(entry.GUID),
			Type:   parttable.Primary,
		}
	}

	return &GPT{
		uuid:       UUID2String(table.Header.DiskGUID),
		partitions: partitionMap,
	}, nil
}
