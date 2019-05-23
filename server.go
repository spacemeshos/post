package main

import (
	"fmt"
	proxy "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spacemeshos/post/rpc"
	"github.com/spacemeshos/post/rpc/api"
	"github.com/spacemeshos/post/signal"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"net"
	"net/http"
)

// startServer starts the RPC server.
func startServer() error {
	signal := signal.NewSignal()
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Initialize and register the implementation of gRPC interface
	var grpcServer *grpc.Server
	options := []grpc.ServerOption{
		grpc.UnaryInterceptor(loggerInterceptor()),
	}

	rpcServer := rpc.NewRPCServer(signal, cfg.Params, cfg.DataDir, cfg.LabelsLogRate)
	grpcServer = grpc.NewServer(options...)

	api.RegisterPostServer(grpcServer, rpcServer)

	// Start the gRPC server listening for HTTP/2 connections.
	lis, err := net.Listen(cfg.RPCListener.Network(), cfg.RPCListener.String())
	if err != nil {
		return fmt.Errorf("failed to listen: %v\n", err)
	}
	defer lis.Close()

	go func() {
		rpcServerLog.Infof("RPC server listening on %s", lis.Addr())
		_ = grpcServer.Serve(lis)
	}()

	// Start the REST proxy for the gRPC server above.
	mux := proxy.NewServeMux()
	err = api.RegisterPostHandlerFromEndpoint(ctx, mux, cfg.RPCListener.String(), []grpc.DialOption{grpc.WithInsecure()})
	if err != nil {
		return err
	}

	go func() {
		rpcServerLog.Infof("REST proxy start listening on %s", cfg.RESTListener.String())
		err := http.ListenAndServe(cfg.RESTListener.String(), mux)
		rpcServerLog.Errorf("REST proxy failed listening: %s\n", err)
	}()

	// Wait for shutdown signal from either a graceful server stop or from
	// the interrupt handler.
	<-signal.ShutdownChannel()
	return nil
}

// loggerInterceptor returns UnaryServerInterceptor handler to log all RPC server incoming requests.
func loggerInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
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
		rpcServerLog.Debugf("%v: %v %v", peer.Addr.String(), info.FullMethod, reqDispStr)

		resp, err := handler(ctx, req)

		if err != nil {
			rpcServerLog.Debugf("%v: FAILURE %v %s", peer.Addr.String(), info.FullMethod, err)
		}
		return resp, err
	}
}
