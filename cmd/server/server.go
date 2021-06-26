package server

// NOTE: PoST RPC server is currently disabled.

/*
import (
	"context"
	"fmt"
	proxy "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spacemeshos/post/rpc"
	"github.com/spacemeshos/post/rpc/api"
	"github.com/spacemeshos/post/shared"
	"github.com/spacemeshos/smutil/log"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"net"
	"net/http"
	"os"
	"runtime"
)

var Cmd = &cobra.Command{
	Use:   "server",
	Short: "start server",
	Run: func(cmd *cobra.Command, args []string) {
		s := NewPostServer()

		logger, err := s.Initialize(cmd, args)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "server initialization failure:", err)
			return
		}

		logger.Info("Version: %s, NumLabels: %v, LabelSize: %v, DataDir: %v, NumCPU: %v",
			shared.Version(), s.cfg.PostCfg.NumLabels, s.cfg.PostCfg.LabelSize, s.cfg.PostCfg.DataDir, runtime.NumCPU())

		err = s.Start(cmd, args, logger)
		if err != nil {
			logger.Error("server start failure: %v", err)
			return
		}
	},
}

func init() {
	setFlags(Cmd, defaultConfig())
}

type PostServer struct {
	cfg *Config
}

func NewPostServer() *PostServer {
	return &PostServer{cfg: defaultConfig()}
}

func (s *PostServer) Initialize(cmd *cobra.Command, args []string) (*log.Log, error) {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}

	s.cfg = cfg

	log.DebugMode(cfg.ServerCfg.LogDebug)
	log.InitSpacemeshLoggingSystem(cfg.ServerCfg.LogDir, "post.log")

	return &log.AppLog, nil
}

func (s *PostServer) Start(cmd *cobra.Command, args []string, logger shared.Logger) error {
	signal := shared.NewSignal(log.AppLog)
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	rpcServer, err := rpc.NewRPCServer(signal, s.cfg.PostCfg, logger)
	if err != nil {
		return err
	}

	// Initialize and register the implementation of gRPC interface
	var grpcServer *grpc.Server
	options := []grpc.ServerOption{
		grpc.UnaryInterceptor(loggerInterceptor(logger)),
	}

	grpcServer = grpc.NewServer(options...)
	api.RegisterPostServer(grpcServer, rpcServer)

	// Resolve the RPC listener
	rpcListener, err := net.ResolveTCPAddr("tcp", s.cfg.ServerCfg.RPCListener)
	if err != nil {
		return err
	}

	// Resolve the REST listener.
	restListener, err := net.ResolveTCPAddr("tcp", s.cfg.ServerCfg.RESTListener)
	if err != nil {
		return err
	}

	// Start the gRPC server listening for HTTP/2 connections.
	lis, err := net.Listen(rpcListener.Network(), rpcListener.String())
	if err != nil {
		return fmt.Errorf("failed to listen: %v\n", err)
	}
	defer lis.Close()

	go func() {
		logger.Info("RPC server listening on %s", lis.Addr())
		_ = grpcServer.Serve(lis)
	}()

	// Start the REST proxy for the gRPC server above.
	mux := proxy.NewServeMux()
	err = api.RegisterPostHandlerFromEndpoint(ctx, mux, rpcListener.String(), []grpc.DialOption{grpc.WithInsecure()})
	if err != nil {
		return err
	}

	go func() {
		logger.Info("REST proxy start listening on %s", restListener.String())
		err := http.ListenAndServe(restListener.String(), mux)
		logger.Error("REST proxy failed listening: %s\n", err)
	}()

	// Wait for shutdown signal from either a graceful server stop or from
	// the interrupt handler.
	<-signal.ShutdownChannel()
	return nil
}

// loggerInterceptor returns UnaryServerInterceptor handler to log all RPC server incoming requests.
func loggerInterceptor(logger shared.Logger) func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		peer, _ := peer.FromContext(ctx)
		maxDispLen := 50
		reqStr := fmt.Sprintf("%v", req)

		var reqDispStr string
		if len(reqStr) > maxDispLen {
			reqDispStr = reqStr[:maxDispLen] + "..."
		} else {
			reqDispStr = reqStr
		}
		logger.Info("%v: %v %v", peer.Addr.String(), info.FullMethod, reqDispStr)

		resp, err := handler(ctx, req)

		if err != nil {
			logger.Info("%v: FAILURE %v %s", peer.Addr.String(), info.FullMethod, err)
		}
		return resp, err
	}
}
*/
