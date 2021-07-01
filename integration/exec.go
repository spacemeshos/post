package integration

// NOTE: PoST RPC server is currently disabled.

/*
import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	// compileMtx guards access to the executable path so that the project is
	// only compiled once.
	compileMtx sync.Mutex

	// executablePath is the path to the compiled executable. This is an empty
	// string until the initial compilation. It should not be accessed directly;
	// use the postExecutablePath() function instead.
	executablePath string
)

// postExecutablePath returns a path to the post server executable.
// To ensure the code tests against the most up-to-date version, this method
// compiles post server the first time it is called. After that, the
// generated binary is used for subsequent requests.
func postExecutablePath(baseDir string) (string, error) {
	compileMtx.Lock()
	defer compileMtx.Unlock()

	// If post has already been compiled, just use that.
	if len(executablePath) != 0 {
		return executablePath, nil
	}

	// Build post and output an executable in a static temp path.
	outputPath := filepath.Join(baseDir, "post")
	if runtime.GOOS == "windows" {
		outputPath += ".exe"
	}

	cmd := exec.Command(
		"go", "build", "-o", outputPath, "github.com/spacemeshos/post",
	)

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to build post: %v", err)
	}

	// Save executable path so future calls do not recompile.
	executablePath = outputPath
	return executablePath, nil
}
*/
