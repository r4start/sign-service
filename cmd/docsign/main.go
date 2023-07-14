package main

import (
	"context"
	"net"
	"net/http"

	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/ratelimit"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"
	"golang.org/x/crypto/ed25519"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/r4start/sign-service/cmd/docsign/internal"
	pb "github.com/r4start/sign-service/pkg/proto"
)

const (
	_addr     = "[::]:10116"
	_httpAddr = "[::]:8080"
	_rpsLimit = 120
)

func main() {
	creds := insecure.NewCredentials()
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return
	}

	service, _ := internal.NewSignServer(privateKey, publicKey)
	authFunc := internal.BuildAuthorizationInterceptor()

	server := grpc.NewServer(grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(
			ratelimit.UnaryServerInterceptor(internal.NewLimiter(_rpsLimit)),
			selector.UnaryServerInterceptor(
				grpcauth.UnaryServerInterceptor(authFunc),
				selector.MatchFunc(internal.AllButReflection)),
		),
		grpc.ChainStreamInterceptor(
			ratelimit.StreamServerInterceptor(internal.NewLimiter(_rpsLimit)),
			selector.StreamServerInterceptor(
				grpcauth.StreamServerInterceptor(authFunc),
				selector.MatchFunc(internal.AllButReflection)),
		))
	pb.RegisterSignServiceServer(server, service)

	listener, err := net.Listen("tcp", _addr)
	if err != nil {
		return
	}

	reflection.Register(server)

	go func() {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		mux := runtime.NewServeMux()
		opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		err := pb.RegisterSignServiceHandlerFromEndpoint(ctx, mux, _addr, opts)
		if err != nil {
			return
		}

		// Start HTTP server (and proxy calls to gRPC server endpoint)
		http.ListenAndServe(_httpAddr, mux)
	}()

	if err := server.Serve(listener); err != nil {
		return
	}
}
