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

func ParseFileIndex(fileName string) (int, error) {
	re := regexp.MustCompile(`^postdata_(\d*).bin$`)
	matches := re.FindStringSubmatch(fileName)
	if len(matches) != 2 {
		return 0, fmt.Errorf("invalid file name: %s", fileName)
	}
	return strconv.Atoi(matches[1])
}

func IsInitFile(file os.FileInfo) bool {
	if file.IsDir() {
		return false
	}

	_, err := ParseFileIndex(file.Name())
	return err == nil
}
