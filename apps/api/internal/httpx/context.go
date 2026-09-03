package httpx

import (
	"context"
	"net/http"
)

type ctxKey int

const (
	RequestKey ctxKey = iota
	ResponseWriterKey
	SessionKey
)

func WithRequest(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, RequestKey, r)
}

func RequestFrom(ctx context.Context) (*http.Request, bool) {
	r, ok := ctx.Value(RequestKey).(*http.Request)
	return r, ok
}

func WithResponseWriter(ctx context.Context, w http.ResponseWriter) context.Context {
	return context.WithValue(ctx, ResponseWriterKey, w)
}

func ResponseWriterFrom(ctx context.Context) (http.ResponseWriter, bool) {
	w, ok := ctx.Value(ResponseWriterKey).(http.ResponseWriter)
	return w, ok
}

type SessionContext struct {
	Token          string
	UserID         string
	OrganizationID string
}

func WithSession(ctx context.Context, session SessionContext) context.Context {
	return context.WithValue(ctx, SessionKey, session)
}

func SessionFrom(ctx context.Context) (SessionContext, bool) {
	s, ok := ctx.Value(SessionKey).(SessionContext)
	return s, ok
}
