package db_test

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/boar-d-white-foundation/drone/db"
	"github.com/boar-d-white-foundation/drone/lang"
	"github.com/stretchr/testify/require"
)

func TestBadger(t *testing.T) {
	t.Parallel()

	bdb := db.NewBadgerDB(":memory:")
	restoreBDB := db.NewBadgerDB(":memory:")
	testDB(t, "badger", bdb, restoreBDB)
}

func testDB(t *testing.T, name string, database db.DB, restoreDB db.DB) {
	ctx := context.Background()
	err := database.Start(ctx)
	require.NoError(t, err)
	defer database.Stop()
	err = restoreDB.Start(ctx)
	require.NoError(t, err)
	defer restoreDB.Stop()

	t.Run(name+" get set", func(t *testing.T) {
		key := "key1"
		err := database.Do(ctx, func(tx db.Tx) error {
			_, err := db.GetJson[int](tx, key)
			require.ErrorIs(t, err, db.ErrKeyNotFound)

			err = db.SetJson(tx, key, 55)
			require.NoError(t, err)

			val, err := db.GetJson[int](tx, key)
			require.NoError(t, err)
			require.Equal(t, 55, val)
			return nil
		})
		require.NoError(t, err)
	})

	type S1 struct {
		A string `json:"a,omitempty"`
		B int    `json:"b,omitempty"`
	}

	type S2 struct {
		Struct S1                  `json:"struct,omitempty"`
		Slice  []int               `json:"slice,omitempty"`
		String string              `json:"string"`
		Int    int                 `json:"int"`
		Float  float64             `json:"float"`
		True   bool                `json:"true"`
		False  bool                `json:"false"`
		Null   *int                `json:"null"`
		Map    map[string]int      `json:"map"`
		Set    map[string]struct{} `json:"set"`
	}

	testSet := S2{
		Struct: S1{
			A: "s1_a",
			B: 33,
		},
		Slice:  []int{444, 13, 44, -1, 0, 44},
		String: "string sg",
		Int:    535533535,
		Float:  -666.44,
		True:   true,
		False:  false,
		Null:   nil,
		Map:    map[string]int{"k1": 1, "k2": 2, "k3": 3},
		Set:    map[string]struct{}{"k1": {}, "k2": {}, "k3": {}},
	}

	t.Run(name+" json", func(t *testing.T) {
		key := "key2"
		err := database.Do(ctx, func(tx db.Tx) error {
			err := db.SetJson[*int](tx, key, nil)
			require.NoError(t, err)
			intPtrRes, err := db.GetJson[*int](tx, key)
			require.NoError(t, err)
			require.Equal(t, (*int)(nil), intPtrRes)

			err = db.SetJson(tx, key, lang.NewPtr(2))
			require.NoError(t, err)
			intPtrRes, err = db.GetJson[*int](tx, key)
			require.NoError(t, err)
			require.Equal(t, lang.NewPtr(2), intPtrRes)

			err = db.SetJson(tx, key, 1)
			require.NoError(t, err)
			intRes, err := db.GetJson[int](tx, key)
			require.NoError(t, err)
			require.Equal(t, 1, intRes)

			err = db.SetJson(tx, key, 1.0)
			require.NoError(t, err)
			floatRes, err := db.GetJson[float64](tx, key)
			require.NoError(t, err)
			require.InEpsilon(t, 1.0, floatRes, 1e-7)

			err = db.SetJson(tx, key, "string")
			require.NoError(t, err)
			strRes, err := db.GetJson[string](tx, key)
			require.NoError(t, err)
			require.Equal(t, "string", strRes)

			err = db.SetJson(tx, key, true)
			require.NoError(t, err)
			trueRes, err := db.GetJson[bool](tx, key)
			require.NoError(t, err)
			require.True(t, trueRes)

			err = db.SetJson(tx, key, false)
			require.NoError(t, err)
			falseRes, err := db.GetJson[bool](tx, key)
			require.NoError(t, err)
			require.False(t, falseRes)

			err = db.SetJson(tx, key, []int{4, 8, 9, 33})
			require.NoError(t, err)
			sliceRes, err := db.GetJson[[]int](tx, key)
			require.NoError(t, err)
			require.Equal(t, []int{4, 8, 9, 33}, sliceRes)

			tm := time.Now()
			err = db.SetJson(tx, key, tm)
			require.NoError(t, err)
			timeRes, err := db.GetJson[time.Time](tx, key)
			require.NoError(t, err)
			require.True(t, tm.Equal(timeRes))

			err = db.SetJson(tx, key, testSet)
			require.NoError(t, err)
			structRes, err := db.GetJson[S2](tx, key)
			require.NoError(t, err)
			require.Equal(t, testSet, structRes)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run(name+" json backup", func(t *testing.T) {
		keys := []string{"key3.0", "key3.1", "key3.2"}
		err := database.Do(ctx, func(tx db.Tx) error {
			for _, key := range keys {
				err = db.SetJson(tx, key, testSet)
				require.NoError(t, err)
			}
			return nil
		})
		require.NoError(t, err)

		var buf bytes.Buffer
		require.Empty(t, buf)
		err = db.DumpJson(ctx, database, &buf)
		require.NoError(t, err)
		require.NotEmpty(t, buf)

		err = db.RestoreJson(ctx, restoreDB, &buf)
		require.NoError(t, err)
		err = restoreDB.Do(ctx, func(tx db.Tx) error {
			for _, key := range keys {
				restored, err := db.GetJson[S2](tx, key)
				require.NoError(t, err)
				require.Equal(t, testSet, restored)
			}
			return nil
		})
		require.NoError(t, err)
	})

	t.Run(name+" transaction", func(t *testing.T) {
		key := "key4"
		err := database.Do(ctx, func(tx db.Tx) error {
			err = db.SetJson(tx, key, 80)
			require.NoError(t, err)
			return nil
		})
		require.NoError(t, err)

		err = database.Do(ctx, func(tx db.Tx) error {
			err = db.SetJson(tx, key, 90)
			require.NoError(t, err)
			return errors.New("err after set")
		})
		require.Error(t, err)

		err = database.Do(ctx, func(tx db.Tx) error {
			val, err := db.GetJson[int](tx, key)
			require.NoError(t, err)
			require.Equal(t, 80, val)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run(name+" concurrent transactions", func(t *testing.T) {
		key := "key5"

		var wg sync.WaitGroup
		wg.Add(10000)
		for i := 0; i < 10000; i++ {
			//nolint:testifylint
			go func() {
				err := database.Do(ctx, func(tx db.Tx) error {
					val, err := db.GetJsonDefault[int](tx, key, 0)
					require.NoError(t, err)

					err = db.SetJson(tx, key, val+1)
					require.NoError(t, err)
					return nil
				})
				require.NoError(t, err)
				wg.Done()
			}()
		}
		wg.Wait()

		err = database.Do(ctx, func(tx db.Tx) error {
			val, err := db.GetJson[int](tx, key)
			require.NoError(t, err)
			require.Equal(t, 10000, val)
			return nil
		})
		require.NoError(t, err)
	})
}
