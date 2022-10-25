package shared

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
)

func InitFileName(index int) string {
	return fmt.Sprintf("postdata_%d.bin", index)
}

func IsInitFile(file os.FileInfo) bool {
	if file.IsDir() {
		return false
	}

	re := regexp.MustCompile("postdata_(.*).bin")
	matches := re.FindStringSubmatch(file.Name())
	if len(matches) != 2 {
		return false
	}
	if _, err := strconv.Atoi(matches[1]); err != nil {
		return false
	}

	return true
}
