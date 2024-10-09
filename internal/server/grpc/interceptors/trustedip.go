package interceptors

import (
	"context"
	"net"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/realip"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TrustedSubnetInterceptor(subnet *net.IPNet) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if subnet == nil {
			return handler(ctx, req)
		}

		var remoteAddr string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			values := md.Get(realip.XRealIp)
			if len(values) > 0 {
				remoteAddr = values[0]
			}
		}

		ip := net.ParseIP(remoteAddr)
		if ip == nil || !subnet.Contains(ip) {
			return nil, status.Error(codes.PermissionDenied, "The request from this ip-address was rejected")
		}

		return handler(ctx, req)
	}
}
