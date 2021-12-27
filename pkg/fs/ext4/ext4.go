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

package ext4

import (
	"encoding/binary"
	"fmt"
	"io"

	fserrors "github.com/minio/directpv/pkg/fs/errors"
)

const dynamicRev = 1

// UUID2String converts UUID to string.
func UUID2String(uuid [16]byte) string {
	return fmt.Sprintf(
		"%08x-%04x-%04x-%x-%x",
		binary.BigEndian.Uint32(uuid[0:4]),
		binary.BigEndian.Uint16(uuid[4:6]),
		binary.BigEndian.Uint16(uuid[6:8]),
		uuid[8:10],
		uuid[10:],
	)
}

// SuperBlock denotes ext4 superblock.
type SuperBlock struct {
	_                         [1024]byte
	NumInodes                 uint32
	NumBlocks                 uint32
	ReservedBlocks            uint32
	FreeBlocks                uint32
	FreeInodes                uint32
	FirstDataBlock            uint32
	LogBlockSize              uint32
	LogClusterSize            uint32
	BlocksPerGroup            uint32
	ObsoleteFragmentsPerGroup uint32
	InodesPerGroup            uint32
	MountTime                 uint32
	WriteTime                 uint32
	MountCount                uint16
	MaxMountCount             uint16
	MagicNum                  uint16
	FilesystemState           uint16
	Errors                    uint16
	MinorRevLevel             uint16
	LastCheckTime             uint32
	CheckInterval             uint32
	CreatorOS                 uint32
	RevLevel                  uint32
	DefaultReserveUID         uint16
	DefaultReserveGID         uint16

	SFirstIno      uint32 /* First non-reserved inode */
	SInodeSize     uint16 /* size of inode structure */
	SBlockGroupNr  uint16 /* block group # of this superblock */
	SFeatureCompat uint32 /* compatible feature set */

	SFeatureIncompat uint32 /* incompatible feature set */
	SFeatureRoCompat uint32 /* readonly-compatible feature set */

	SUuid [16]uint8 /* 128-bit uuid for volume */

	SVolumeName [16]byte /* volume name */

	SLastMounted [64]byte /* directory where last mounted */

	SAlgorithmUsageBitmap uint32 /* For compression */

	/*
	 * Performance hints.  Directory preallocation should only
	 * happen if the EXT4_FEATURE_COMPAT_DIR_PREALLOC flag is on.
	 */
	SPreallocBlocks    uint8  /* Nr of blocks to try to preallocate*/
	SPreallocDirBlocks uint8  /* Nr to preallocate for dirs */
	SReservedGdtBlocks uint16 /* Per group desc for online growth */

	/*
	 * Journaling support valid if EXT4_FEATURE_COMPAT_HAS_JOURNAL set.
	 */
	SJournalUUID [16]uint8 /* uuid of journal superblock */

	SJournalInum    uint32    /* inode number of journal file */
	SJournalDev     uint32    /* device number of journal file */
	SLastOrphan     uint32    /* start of list of inodes to delete */
	SHashSeed       [4]uint32 /* HTREE hash seed */
	SDefHashVersion uint8     /* Default hash version to use */
	SJnlBackupType  uint8
	SDescSize       uint16 /* Size of group descriptors, in bytes, if the 64bit incompat feature flag is set. */

	SDefaultMountOpts uint32
	SFirstMetaBg      uint32     /* First metablock block group */
	SMkfsTime         uint32     /* When the filesystem was created */
	SJnlBlocks        [17]uint32 /* Backup of the journal inode */

	/* 64bit support valid if EXT4_FEATURE_COMPAT_64BIT */

	// 0x150
	SBlocksCountHi     uint32 /* Blocks count */
	SRBlocksCountHi    uint32 /* Reserved blocks count */
	SFreeBlocksCountHi uint32 /* Free blocks count */
	SMinExtraIsize     uint16 /* All inodes have at least # bytes */
	SWantExtraIsize    uint16 /* New inodes should reserve # bytes */

	SFlags            uint32 /* Miscellaneous flags */
	SRaidStride       uint16 /* RAID stride */
	SMmpInterval      uint16 /* # seconds to wait in MMP checking */
	SMmpBlock         uint64 /* Block for multi-mount protection */
	SRaidStripeWidth  uint32 /* blocks on all data disks (N*stride)*/
	SLogGroupsPerFlex uint8  /* FLEX_BG group size */
	SChecksumType     uint8  /* metadata checksum algorithm used */
	SEncryptionLevel  uint8  /* versioning level for encryption */
	SReservedPad      uint8  /* Padding to next 32bits */
	SKbytesWritten    uint64 /* nr of lifetime kilobytes written */

	SSnapshotInum         uint32 /* Inode number of active snapshot */
	SSnapshotID           uint32 /* sequential ID of active snapshot */
	SSnapshotRBlocksCount uint64 /* reserved blocks for active snapshot's future use */
	SSnapshotList         uint32 /* inode number of the head of the on-disk snapshot list */

	SErrorCount      uint32    /* number of fs errors */
	SFirstErrorTime  uint32    /* first time an error happened */
	SFirstErrorIno   uint32    /* inode involved in first error */
	SFirstErrorBlock uint64    /* block involved of first error */
	SFirstErrorFunc  [32]uint8 /* function where the error happened */
	SFirstErrorLine  uint32    /* line number where error happened */
	SLastErrorTime   uint32    /* most recent time of an error */
	SLastErrorIno    uint32    /* inode involved in last error */
	SLastErrorLine   uint32    /* line number where error happened */
	SLastErrorBlock  uint64    /* block involved of last error */
	SLastErrorFunc   [32]uint8 /* function where the error happened */

	SMountOpts        [64]uint8
	SUsrQuotaInum     uint32    /* inode for tracking user quota */
	SGrpQuotaInum     uint32    /* inode for tracking group quota */
	SOverheadClusters uint32    /* overhead blocks/clusters in fs */
	SBackupBgs        [2]uint32 /* groups with sparse_super2 SBs */
	SEncryptAlgos     [4]uint8  /* Encryption algorithms in use  */
	SEncryptPwSalt    [16]uint8 /* Salt used for string2key algorithm */
	SLpfIno           uint32    /* Location of the lost+found inode */
	SPrjQuotaInum     uint32    /* inode for tracking project quota */
	SChecksumSeed     uint32    /* crc32c(uuid) if csum_seed set */
	SWtimeHi          uint8
	SMtimeHi          uint8
	SMkfsTimeHi       uint8
	SLastcheckHi      uint8
	SFirstErrorTimeHi uint8
	SLastErrorTimeHi  uint8
	SPad              [2]uint8
	SReserved         [96]uint32 /* Padding to the end of the block */
	SChecksum         int32      /* crc32c(superblock) */
}

// ID returns filesystem UUID.
func (sb *SuperBlock) ID() string {
	if sb.RevLevel == dynamicRev {
		return UUID2String(sb.SUuid)
	}

	return UUID2String([16]byte{})
}

// Type returns "ext4".
func (sb *SuperBlock) Type() string {
	return "ext4"
}

// TotalCapacity returns total capacity of filesystem.
func (sb *SuperBlock) TotalCapacity() uint64 {
	return uint64(sb.NumBlocks) * (1 << (10 + sb.LogBlockSize))
}

// FreeCapacity returns free capacity of filesystem.
func (sb *SuperBlock) FreeCapacity() uint64 {
	return uint64(sb.FreeBlocks) * (1 << (10 + sb.LogBlockSize))
}

// Probe tries to probe ext2/3/4 superblock.
func Probe(reader io.Reader) (*SuperBlock, error) {
	var superBlock SuperBlock
	if err := binary.Read(reader, binary.LittleEndian, &superBlock); err != nil {
		return nil, err
	}

	if superBlock.MagicNum != 0xef53 {
		return nil, fserrors.ErrFSNotFound
	}

	// TODO: validate superblock CRC
	return &superBlock, nil
}
