package server

import (
	"context"
	"log/slog"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/authlimit"
	pb "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// AuthRateLimitInterceptor uses the transport peer, never client-supplied IP
// metadata. When Web is the peer, this is an aggregate limit for that Web host.
func AuthRateLimitInterceptor(limits *authlimit.Limiter, logger *slog.Logger) grpc.UnaryServerInterceptor {
	var mu sync.Mutex
	var failures, throttled int
	var nextLog time.Time
	record := func(err error) {
		code := status.Code(err)
		if logger == nil || (code != codes.Unauthenticated && code != codes.ResourceExhausted) {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if code == codes.Unauthenticated {
			failures++
		} else {
			throttled++
		}
		now := time.Now()
		if !now.Before(nextLog) {
			logger.Warn("authentication failures or throttling", "failed_attempts", failures, "throttled_attempts", throttled)
			failures, throttled = 0, 0
			nextLog = now.Add(30 * time.Second)
		}
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		switch info.FullMethod {
		case pb.AuthService_Login_FullMethodName, pb.AuthService_ChangePassword_FullMethodName, pb.AuthService_CreateUser_FullMethodName:
		default:
			return handler(ctx, req)
		}
		if retry := limits.AllowIP(authPeerIP(ctx)); retry > 0 {
			err := toStatusError(&authlimit.LimitError{RetryAfter: retry})
			record(err)
			return nil, err
		}
		resp, err := handler(ctx, req)
		record(err)
		return resp, err
	}
}

func authPeerIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok || p.Addr == nil {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return "unknown"
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return "unknown"
	}
	return ip.Unmap().WithZone("").String()
}
