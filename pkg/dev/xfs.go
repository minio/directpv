package dev

import (
	"errors"
)

const FSTypeXFS FSType = "xfs"

var XFSMagicNum uint32 = 0x58465342
var ErrNotXFS = errors.New("Not a xfs partition")

type XFSSuperBlock struct {
	MagicNumber         uint32
	BlockSize           uint32
	TotalBlocks         uint64
	RBlocks             uint64
	RExtents            uint64
	UUID                [16]byte
	FirstBlock          uint64
	RootInode           uint64
	ExtentsBitmapInode  uint64
	BitmapSummaryInode  uint64
	ExtentSize          uint32
	AGSize              uint32
	AGCount             uint32
	BitmapBlocks        uint32
	JournalBlocks       uint32
	FilesystemVersion   uint16
	SectorSize          uint16
	InodeSize           uint16
	Inodes              uint16
	FilesystemName      [12]byte
	LogBlockSize        uint8
	LogSectorSize       uint8
	LogInodeSize        uint8
	LogInodeOrBlockSize uint8
	LogAGSize           uint8
	LogExtents          uint8
	InProgress          uint8
	MaxInodePercentage  uint8
	AllocatedInodes     uint64
	FreeInodes          uint64
	FreeBlocks          uint64
	FreeExtents         uint64
	// Ignoring the rest
}

func (x XFSSuperBlock) Is() bool {
	return x.MagicNumber == XFSMagicNum
}
