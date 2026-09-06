package httpx

import (
	"context"
	"net/http"
	"strings"
)

type ClientInfo struct {
	IPAddress string
	UserAgent string
}

func ClientInfoFromRequest(r *http.Request) ClientInfo {
	ip := r.Header.Get("CF-Connecting-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
		if idx := strings.Index(ip, ","); idx >= 0 {
			ip = strings.TrimSpace(ip[:idx])
		}
	}
	if ip == "" {
		ip = strings.TrimSpace(r.RemoteAddr)
	}
	return ClientInfo{
		IPAddress: ip,
		UserAgent: r.Header.Get("User-Agent"),
	}
}

func ClientInfoFrom(ctx context.Context) ClientInfo {
	if r, ok := RequestFrom(ctx); ok {
		return ClientInfoFromRequest(r)
	}
	return ClientInfo{}
}

func WithClientInfo(ctx context.Context, info ClientInfo) context.Context {
	return context.WithValue(ctx, clientInfoKey, info)
}

type clientInfoKeyType struct{}

var clientInfoKey = clientInfoKeyType{}

func ClientInfoFromContext(ctx context.Context) (ClientInfo, bool) {
	info, ok := ctx.Value(clientInfoKey).(ClientInfo)
	return info, ok
}
