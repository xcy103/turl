package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCache is a minimal Interface whose Get result is configurable.
type fakeCache struct {
	val []byte
	err error
}

func (f *fakeCache) Set(context.Context, string, []byte, time.Duration) error { return nil }
func (f *fakeCache) Get(context.Context, string) ([]byte, error)              { return f.val, f.err }
func (f *fakeCache) Del(context.Context, string) error                       { return nil }
func (f *fakeCache) Ping(context.Context) error                              { return nil }
func (f *fakeCache) Close() error                                            { return nil }

func TestMetricsCache_Get_PassesThrough(t *testing.T) {
	tests := []struct {
		name    string
		val     []byte
		err     error
		wantErr error
	}{
		{name: "hit", val: []byte("https://example.com"), err: nil},
		{name: "miss", val: nil, err: ErrCacheMiss, wantErr: ErrCacheMiss},
		{name: "backend error", val: nil, err: errors.New("conn refused"), wantErr: errors.New("conn refused")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newMetricsCache("local", &fakeCache{val: tt.val, err: tt.err})

			got, err := c.Get(context.Background(), "k")

			assert.Equal(t, tt.val, got)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
