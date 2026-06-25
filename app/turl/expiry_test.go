package turl

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/beihai0xff/turl/internal/tests/mocks"
	"github.com/beihai0xff/turl/pkg/cache"
	"github.com/beihai0xff/turl/pkg/storage"
)

func TestCacheTTL(t *testing.T) {
	const configTTL = 10 * time.Minute

	t.Run("no expiry uses configured ttl", func(t *testing.T) {
		require.Equal(t, configTTL, cacheTTL(configTTL, nil))
	})

	t.Run("far expiry uses configured ttl", func(t *testing.T) {
		far := time.Now().Add(time.Hour)
		require.Equal(t, configTTL, cacheTTL(configTTL, &far))
	})

	t.Run("near expiry is bounded by remaining lifetime", func(t *testing.T) {
		soon := time.Now().Add(time.Minute)
		require.InDelta(t, time.Minute, cacheTTL(configTTL, &soon), float64(2*time.Second))
	})
}

func TestIsExpired(t *testing.T) {
	require.False(t, isExpired(nil))

	past := time.Now().Add(-time.Second)
	require.True(t, isExpired(&past))

	future := time.Now().Add(time.Hour)
	require.False(t, isExpired(&future))
}

// TestQueryService_Retrieve_Expired verifies an expired link returns ErrExpired
// and is never written back to the cache (the mock fails on an unexpected Set).
func TestQueryService_Retrieve_Expired(t *testing.T) {
	mockCache, mockStorage := mocks.NewMockCache(t), mocks.NewMockStorage(t)
	q := &queryService{ttl: time.Second, db: mockStorage, cache: mockCache}

	past := time.Now().Add(-time.Hour)
	mockCache.EXPECT().Get(mock.Anything, mock.Anything).Return(nil, cache.ErrCacheMiss).Times(1)
	mockStorage.EXPECT().GetByShortID(mock.Anything, uint64(38068692543)).
		Return(&storage.TinyURL{LongURL: []byte("https://expired.com"), ExpiresAt: &past}, nil).Times(1)

	got, err := q.Retrieve(context.Background(), []byte("zzzzzz"))

	require.ErrorIs(t, err, ErrExpired)
	require.Nil(t, got)
}
