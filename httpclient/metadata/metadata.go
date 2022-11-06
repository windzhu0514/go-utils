package metadata

import (
	"context"
	"fmt"
	"strings"
)

// Metadata is our way of representing request headers internally.
// They're used at the RPC level and translate back and forth
// from Transport headers.
type Metadata map[string]string

// New creates an MD from a given key-values map.
func New(mds ...map[string]string) Metadata {
	md := Metadata{}
	for _, m := range mds {
		for k, v := range m {
			md.Set(k, v)
		}
	}
	return md
}

// Get returns the value associated with the passed key.
func (m Metadata) Get(key string) string {
	return m[strings.ToLower(key)]
}

// Set stores the key-value pair.
func (m Metadata) Set(key string, value string) {
	if key == "" || value == "" {
		return
	}
	m[strings.ToLower(key)] = value
}

// Range iterate over element in metadata.
func (m Metadata) Range(f func(k, v string) bool) {
	for k, v := range m {
		if !f(k, v) {
			break
		}
	}
}

// Clone returns a deep copy of Metadata
func (m Metadata) Clone() Metadata {
	md := Metadata{}
	for k, v := range m {
		md[k] = v
	}
	return md
}

type requestMetadataKey struct{}

// NewRequestContext creates a new context with request md attached.
func NewRequestContext(ctx context.Context, md Metadata) context.Context {
	return context.WithValue(ctx, requestMetadataKey{}, md)
}

// FromRequestContext returns the request metadata in ctx if it exists.
func FromRequestContext(ctx context.Context) (Metadata, bool) {
	md, ok := ctx.Value(requestMetadataKey{}).(Metadata)
	return md, ok
}

// AppendToRequestContext returns a new context with the provided kv merged
// with any existing metadata in the context.
func AppendToRequestContext(ctx context.Context, kv ...string) context.Context {
	if len(kv)%2 == 1 {
		panic(fmt.Sprintf("metadata: AppendToRequestContext got an odd number of input pairs for metadata: %d", len(kv)))
	}
	md, _ := FromRequestContext(ctx)
	md = md.Clone()
	for i := 0; i < len(kv); i += 2 {
		md.Set(kv[i], kv[i+1])
	}
	return NewRequestContext(ctx, md)
}

// MergeToRequestContext merge new metadata into ctx.
func MergeToRequestContext(ctx context.Context, cmd Metadata) context.Context {
	md, _ := FromRequestContext(ctx)
	md = md.Clone()
	for k, v := range cmd {
		md[k] = v
	}
	return NewRequestContext(ctx, md)
}
