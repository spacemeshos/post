package shared

import (
	"github.com/tzdybal/go-disk-usage/du"
	"os"
)

var (
	// OwnerReadWriteExec is a standard owner read / write / exec file permission.
	OwnerReadWriteExec = os.FileMode(0700)

	// OwnerReadWrite is a standard owner read / write file permission.
	OwnerReadWrite = os.FileMode(0600)
)

func AvailableSpace(path string) uint64 {
	usage := du.NewDiskUsage(path)
	return usage.Available()
}
