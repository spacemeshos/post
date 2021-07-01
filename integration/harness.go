package integration

// NOTE: PoST RPC server is currently disabled.

/*
import (
	"context"
	"fmt"
	"github.com/spacemeshos/post/config"
	"github.com/spacemeshos/post/rpc/api"
	"google.golang.org/grpc"
	"os"
	"path/filepath"
	"time"
)

// Harness fully encapsulates an active post server process to provide a unified
// platform to programmatically drive a post server instance, whether for
// creating rpc driven integration tests, or for any other usage.
type Harness struct {
	server *server
	api.PostClient
}

// NewHarness creates and initializes a new instance of Harness.
func NewHarness(cfg *config.Config) (*Harness, error) {
	serverCfg, err := newConfig(cfg)
	if err != nil {
		return nil, err
	}

	server, err := newServer(serverCfg)
	if err != nil {
		return nil, err
	}

	// Spawn a new post server process.
	if err := server.start(); err != nil {
		return nil, err
	}

	// Verify the client connectivity.
	// If failed, shutdown the server.
	conn, err := connectClient(serverCfg.rpcListen)
	if err != nil {
		_ = server.shutdown(true)
		return nil, err
	}

	h := &Harness{
		server:     server,
		PostClient: api.NewPostClient(conn),
	}

	return h, nil
}

// TearDown stops the harness running instance.
// The created process is killed, and the temporary
// directories are removed.
func (h *Harness) TearDown(cleanup bool) error {
	if err := h.server.shutdown(cleanup); err != nil {
		return err
	}

	return nil
}

// ProcessErrors returns a channel used for reporting any fatal process errors.
func (h *Harness) ProcessErrors() <-chan error {
	return h.server.errChan
}

// connectClient attempts to establish a gRPC Client connection
// to the provided target.
func connectClient(target string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}
	defer cancel()

	conn, err := grpc.DialContext(ctx, target, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to RPC server: %v", err)
	}

	return conn, nil
}

// baseDir is the directory path of the temp directory for all the harness files.
func baseDir() (string, error) {
	baseDir := filepath.Join(os.TempDir(), "post")
	err := os.MkdirAll(baseDir, 0755)
	return baseDir, err
}
*/
