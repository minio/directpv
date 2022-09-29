package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	directpv "github.com/minio/directpv/pkg/apis/directpv.min.io/v1beta1"
	directcsi "github.com/minio/directpv/pkg/legacy/apis/direct.csi.min.io/v1beta5"
)

func TestMigrate(t *testing.T) {
	oldDriveList := []directcsi.DirectCSIDrive{}
	newDriveList1 := []directpv.DirectPVDrive{}
	newDriveList2, therror := migrateDriveCRD(oldDriveList)
	fmt.Println("message: ", newDriveList2, therror)
	// Checking the conversion is taking place from old to new resource definition
	assert.Equal(t, newDriveList1, newDriveList2, "The two arrays should be same")
}
