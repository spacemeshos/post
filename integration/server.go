package integration

import (
	"bytes"
	"fmt"
	"github.com/spacemeshos/post/config"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// serverConfig contains all the args and data required to launch a post server
// instance  and connect to it via rpc client.
type serverConfig struct {
	config.Config
	rpcListen string
	baseDir   string
	dataDir   string
	exe       string
}

// newConfig returns a newConfig with all default values.
func newConfig(cfg *config.Config) (*serverConfig, error) {
	baseDir, err := baseDir()
	if err != nil {
		return nil, err
	}

	postPath, err := postExecutablePath(filepath.Join(os.TempDir(), "post-build"))

	if err != nil {
		return nil, err
	}

	return &serverConfig{
		Config:    *cfg,
		baseDir:   baseDir,
		rpcListen: "127.0.0.1:18558",
		exe:       postPath,
	}, nil
}

// genArgs generates a slice of command line arguments from serverConfig instance.
func (cfg *serverConfig) genArgs() []string {
	var args []string

	args = append(args, fmt.Sprintf("--homedir=%v", cfg.baseDir))
	args = append(args, fmt.Sprintf("--rpclisten=%v", cfg.rpcListen))

	args = append(args, fmt.Sprintf("--post-numfiles=%v", cfg.NumFiles))

	args = append(args, fmt.Sprintf("--post-numlabels=%v", cfg.NumLabels))
	args = append(args, fmt.Sprintf("--post-labelsize=%v", cfg.LabelSize))
	args = append(args, fmt.Sprintf("--post-k1=%v", cfg.K1))
	args = append(args, fmt.Sprintf("--post-k2=%v", cfg.K2))

	args = append(args, fmt.Sprintf("--post-parallel-files=%v", cfg.MaxWriteFilesParallelism))
	args = append(args, fmt.Sprintf("--post-parallel-infile=%v", cfg.MaxWriteInFileParallelism))
	args = append(args, fmt.Sprintf("--post-parallel-read=%v", cfg.MaxReadFilesParallelism))

	// Disabling disk space availability checks because datadir is a temp dir,
	// and so stats might not be reliable.
	args = append(args, fmt.Sprintf("--post-disable-space-checks"))

	return args
}

// server houses the necessary state required to configure, launch,
// and manage post server process.
type server struct {
	cfg *serverConfig
	cmd *exec.Cmd

	// processExit is a channel that's closed once it's detected that the
	// process this instance is bound to has exited.
	processExit chan struct{}

	quit chan struct{}
	wg   sync.WaitGroup

	errChan chan error
}

// newNode creates a new post server instance according to the passed cfg.
func newServer(serverCfg *serverConfig) (*server, error) {
	return &server{
		cfg:     serverCfg,
		errChan: make(chan error),
	}, nil
}

// start launches a new running process of post server.
func (s *server) start() error {
	s.quit = make(chan struct{})

	args := s.cfg.genArgs()
	s.cmd = exec.Command(s.cfg.exe, args...)

	// Redirect stderr output to buffer
	var errb bytes.Buffer
	s.cmd.Stderr = &errb

	var outb bytes.Buffer
	s.cmd.Stdout = &outb

	if err := s.cmd.Start(); err != nil {
		return err
	}

	// Launch a new goroutine which that bubbles up any potential fatal
	// process errors to errChan.
	s.processExit = make(chan struct{})
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		err := s.cmd.Wait()

		if err != nil {
			// Don't propagate 'signal: killed' error,
			// since it's an expected behavior.
			if !strings.Contains(err.Error(), "signal: killed") {
				s.errChan <- fmt.Errorf("%v\n%v\n%v\n", err, errb.String(), outb.String())
			}
		}

		// Signal any onlookers that this process has exited.
		close(s.processExit)
	}()

	return nil
}

// shutdown terminates the running post server process, and cleans up
// all files/directories created by it.
func (s *server) shutdown(cleanup bool) error {
	if err := s.stop(); err != nil {
		return err
	}

	if cleanup {
		if err := s.cleanup(); err != nil {
			return err
		}
	}

	return nil
}

// stop kills the server running process, since it doesn't support
// RPC-driven stop functionality.
func (s *server) stop() error {
	// Do nothing if the process is not running.
	if s.processExit == nil {
		return nil
	}

	if err := s.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process: %v", err)
	}

	close(s.quit)
	s.wg.Wait()

	s.quit = nil
	s.processExit = nil
	return nil
}

// cleanup cleans up the temporary files/directories created by the server process.
func (s *server) cleanup() error {
	return os.RemoveAll(s.cfg.baseDir)
}
