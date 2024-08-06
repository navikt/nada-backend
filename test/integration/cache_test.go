package integration

import (
	"os"
	"testing"
	"time"

	"github.com/navikt/nada-backend/pkg/cache"
	"github.com/navikt/nada-backend/pkg/database"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type CacheTest struct {
	Field1 string
	Field2 string
}

func TestCache(t *testing.T) {
	log := zerolog.New(os.Stdout)

	c := NewContainers(t, log)
	defer c.Cleanup()

	pgCfg := c.RunPostgres(NewPostgresConfig())

	repo, err := database.New(
		pgCfg.ConnectionURL(),
		10,
		10,
	)
	assert.NoError(t, err)

	t.Run("Test set and get cache", func(t *testing.T) {
		val := &CacheTest{
			Field1: "bob",
			Field2: "the builder",
		}

		c := cache.New(time.Second*10, repo.GetDB(), zerolog.New(os.Stdout))

		c.Set("myTestKey", val)
		got := &CacheTest{}
		isInCache := c.Get("myTestKey", got)

		assert.True(t, isInCache)
		assert.Equal(t, val, got)
	})

	t.Run("Test cache expiration", func(t *testing.T) {
		val := &CacheTest{
			Field1: "bob",
			Field2: "the builder",
		}

		c := cache.New(time.Second*1, repo.GetDB(), zerolog.New(os.Stdout))

		c.Set("myTestKey", val)

		time.Sleep(time.Second * 2)

		got := &CacheTest{}

		isInCache := c.Get("myTestKey", got)
		assert.False(t, isInCache)
		assert.NotEqual(t, val, got)
	})

	t.Run("Test cache update", func(t *testing.T) {
		val := &CacheTest{
			Field1: "bob",
			Field2: "the builder",
		}

		c := cache.New(time.Second*10, repo.GetDB(), zerolog.New(os.Stdout))

		c.Set("myTestKey", val)

		newVal := &CacheTest{
			Field1: "bob",
			Field2: "the swimmer",
		}

		c.Set("myTestKey", newVal)

		got := &CacheTest{}
		isInCache := c.Get("myTestKey", got)

		assert.True(t, isInCache)
		assert.Equal(t, newVal, got)
	})
}
