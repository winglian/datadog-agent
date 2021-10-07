package conn_context

import (
	"context"
	"net"
	"net/http"
)

// ContextKey is used to define a context key in an HTTP request
type ContextKey struct {
	key string
}

// ConnContextKey is used to hold the Conn context of a request
var ConnContextKey = &ContextKey{"http-conn"}

// SaveConnInContext saves the socket Conn in the context
func SaveConnInContext(ctx context.Context, c net.Conn) context.Context {
	return context.WithValue(ctx, ConnContextKey, c)
}

// GetConn retruns the request Conn if it exists
func GetConn(r *http.Request) net.Conn {
	return r.Context().Value(ConnContextKey).(net.Conn)
}
