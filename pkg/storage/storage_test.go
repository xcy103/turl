package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/beihai0xff/turl/internal/tests"
	"github.com/beihai0xff/turl/pkg/db/mysql"
)

func Test_storage_DeleteExpired(t *testing.T) {
	db, _ := mysql.New(tests.GlobalConfig.MySQL)
	s, ctx := newStorage(db), context.Background()
	t.Cleanup(func() { s.Close() })

	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	_, err := s.Insert(ctx, uint64(70001), []byte("https://expired-1.com"), &past)
	require.NoError(t, err)
	_, err = s.Insert(ctx, uint64(70002), []byte("https://expired-2.com"), &past)
	require.NoError(t, err)
	_, err = s.Insert(ctx, uint64(70003), []byte("https://alive.com"), &future)
	require.NoError(t, err)

	n, err := s.DeleteExpired(ctx, time.Now(), 100)
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, int64(2))

	// The unexpired link is still retrievable...
	_, err = s.GetByShortID(ctx, uint64(70003))
	require.NoError(t, err)

	// ...while the expired ones are gone.
	_, err = s.GetByShortID(ctx, uint64(70001))
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestMain(m *testing.M) {
	tests.CreateTable(&TinyURL{})

	code := m.Run()
	tests.DropTable(&TinyURL{})

	os.Exit(code)
}

func TestNew(t *testing.T) {
	db, _ := mysql.New(tests.GlobalConfig.MySQL)

	s := New(db)
	t.Cleanup(func() {
		s.Close()
	})

	require.NotNil(t, s)
}

func Test_newStorage(t *testing.T) {
	db, _ := mysql.New(tests.GlobalConfig.MySQL)

	s := newStorage(db)
	t.Cleanup(func() {
		s.Close()
	})

	require.NotNil(t, s)
}

func Test_storage_Insert(t *testing.T) {
	db, _ := mysql.New(tests.GlobalConfig.MySQL)

	long := []byte("www.Insert.com")
	s, ctx := newStorage(db), context.Background()
	t.Cleanup(func() { s.Close() })

	t.Run("Insert", func(t *testing.T) {
		got, err := s.Insert(ctx, uint64(10000), long, nil)
		require.NoError(t, err)
		require.Equal(t, long, got.LongURL)
		require.Equal(t, 10000, int(got.Short))
		require.Greater(t, got.CreatedAt.Unix(), int64(0))
	})

	t.Run("InsertDuplicateURL", func(t *testing.T) {
		got, err := s.Insert(ctx, uint64(30000), long, nil)
		require.ErrorIs(t, err, gorm.ErrDuplicatedKey)
		require.Nil(t, got)
	})

	t.Run("InsertDuplicateShort", func(t *testing.T) {
		got, err := s.Insert(ctx, uint64(10000), []byte("www.InsertDuplicateShort.com"), nil)
		require.ErrorIs(t, err, gorm.ErrDuplicatedKey)
		require.Nil(t, got)
	})
}

func Test_storage_GetTinyURLByID(t *testing.T) {
	db, _ := mysql.New(tests.GlobalConfig.MySQL)

	short, long := uint64(40000), []byte("www.GetByShortID.com")
	s, ctx := newStorage(db), context.Background()
	t.Cleanup(func() { s.Close() })

	got, err := s.Insert(ctx, short, long, nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	got, err = s.GetByShortID(ctx, short)
	require.NoError(t, err)
	require.Equal(t, long, got.LongURL)

	got, err = s.GetByShortID(ctx, 100)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func Test_storage_GetByLongURL(t *testing.T) {
	db, _ := mysql.New(tests.GlobalConfig.MySQL)

	long := []byte("www.GetByLongURL.com")
	s, ctx := newStorage(db), context.Background()
	t.Cleanup(func() { s.Close() })

	t.Run("GetByLongURL", func(t *testing.T) {
		_, err := s.Insert(ctx, uint64(50000), long, nil)
		require.NoError(t, err)

		got, err := s.GetByLongURL(ctx, long)
		require.NoError(t, err)
		require.Equal(t, long, got.LongURL)
		require.Equal(t, uint64(50000), got.Short)
	})

	t.Run("GetByLongURLNotFound", func(t *testing.T) {
		got, err := s.GetByLongURL(ctx, []byte("www.GetByLongURLNotFound.com"))
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		require.Nil(t, got)
	})
}

func Test_storage_Delete(t *testing.T) {
	db, _ := mysql.New(tests.GlobalConfig.MySQL)

	long := []byte("www.storage_Delete.com")
	s, ctx := newStorage(db), context.Background()
	t.Cleanup(func() { s.Close() })

	t.Run("Delete", func(t *testing.T) {
		_, err := s.Insert(ctx, uint64(60000), long, nil)
		require.NoError(t, err)

		err = s.Delete(ctx, uint64(60000))
		require.NoError(t, err)

		got, err := s.GetByShortID(ctx, uint64(60000))
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		require.Nil(t, got)
	})

	t.Run("DeleteNotFound", func(t *testing.T) {
		err := s.Delete(ctx, uint64(60000))
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}
