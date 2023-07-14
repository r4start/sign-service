package internal

import (
	"context"

	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"

	"google.golang.org/grpc/codes"
	reflection "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
)

const (
	_expectedScheme = "bearer"
)

func AllButReflection(ctx context.Context, callMeta interceptors.CallMeta) bool {
	return reflection.ServerReflection_ServiceDesc.ServiceName != callMeta.Service
}

func checkAuthorization(string) (bool, error) {
	return true, nil
}

func BuildAuthorizationInterceptor() grpcauth.AuthFunc {
	return func(ctx context.Context) (context.Context, error) {
		token, err := grpcauth.AuthFromMD(ctx, _expectedScheme)
		if err != nil {
			return ctx, status.Error(codes.Unauthenticated, "unauthorized")
		}

		if authorized, err := checkAuthorization(token); err != nil || !authorized {
			return ctx, status.Error(codes.Unauthenticated, "unauthorized")
		}

		return ctx, nil
	}
}
