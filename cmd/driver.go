package cmd

import (
	"github.com/minio/jbod-csi-driver/pkg/identity"
)

func driver(args []string) error {
	if err := identity.Run(identity); err != nil {
		return err
	}

	return nil
}
