package middleware

import (
	"context"
	ctxHelper "main/pkg/jwt/context"
	"main/pkg/jwt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var publicMethods = map[string]struct{}{
	"/auth.AuthService/Login":    {},
	"/auth.AuthService/Register": {},
}

func AuthInterceptor(jwtManager *jwt.Manager) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {

		if req == nil {
			return nil, status.Errorf(codes.Unauthenticated, "request is nil")
		}

		if _, ok := publicMethods[info.FullMethod]; ok {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, "metadata is missing")
		}

		values := md.Get("authorization")
		if len(values) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "authorization token is not provided")
		}

		accessToken := strings.TrimPrefix(values[0], "Bearer ")
		claims, err := jwtManager.Parse(accessToken)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "access token is invalid: %v", err)
		}
		newCtx := ctxHelper.ToContext(ctx, claims)

		return handler(newCtx, req)
	}
}
