package cache_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/cache"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestData struct {
	Field1 string `json:"field1"`
	Field2 int    `json:"field2"`
}

func setup(t *testing.T) (cache.Cache, redismock.ClientMock, *config.CacheConfig) {
	t.Helper()

	client, mock := redismock.NewClientMock()
	cfg := &config.CacheConfig{
		DefaultTTL: 10 * time.Minute,
	}
	redisCache := cache.NewRedisCache(client, cfg)

	return redisCache, mock, cfg
}

func TestNewRedisCache(t *testing.T) {
	redisCache, _, _ := setup(t)
	assert.NotNil(t, redisCache, "NewRedisCache should return a non-nil Cache instance")
}

func TestGet(t *testing.T) {
	ctx := t.Context()
	testKey := "test:get"
	testValue := TestData{Field1: "value1", Field2: 123}
	jsonData, err := json.Marshal(testValue)
	require.NoError(t, err)

	t.Run("Success - Key Found", func(t *testing.T) {
		// Arrange
		redisCache, mock, _ := setup(t)

		var result TestData

		mock.ExpectGet(testKey).SetVal(string(jsonData))

		// Act
		found, err := redisCache.Get(ctx, testKey, &result)

		// Assert
		require.NoError(t, err, "Get should not return an error on success")
		assert.True(t, found, "Get should return found=true when key exists")
		assert.Equal(t, testValue, result, "Get should correctly unmarshal the data")
		assert.NoError(t, mock.ExpectationsWereMet(), "Redis mock expectations not met")
	})

	t.Run("Success - Key Not Found (Cache Miss)", func(t *testing.T) {
		// Arrange
		redisCache, mock, _ := setup(t)

		var result TestData

		mock.ExpectGet(testKey).SetErr(redis.Nil)

		// Act
		found, err := redisCache.Get(ctx, testKey, &result)

		// Assert
		require.NoError(t, err, "Get should not return an error on cache miss")
		assert.False(t, found, "Get should return found=false on cache miss")
		assert.Empty(t, result, "Result should be zero value on cache miss")
		assert.NoError(t, mock.ExpectationsWereMet(), "Redis mock expectations not met")
	})

	t.Run("Failure - Redis Error", func(t *testing.T) {
		// Arrange
		redisCache, mock, _ := setup(t)

		var result TestData

		expectedErr := errors.New("redis connection error")

		mock.ExpectGet(testKey).SetErr(expectedErr)

		// Act
		found, err := redisCache.Get(ctx, testKey, &result)

		// Assert
		require.Error(t, err, "Get should return an error when Redis fails")
		assert.False(t, found, "Get should return found=false on Redis error")
		assert.ErrorIs(t, err, expectedErr, "Error should wrap the original Redis error")
		assert.Contains(t, err.Error(), fmt.Sprintf("failed to get key %s from redis", testKey), "Error message mismatch")
		assert.NoError(t, mock.ExpectationsWereMet(), "Redis mock expectations not met")
	})

	t.Run("Failure - Unmarshal Error", func(t *testing.T) {
		// Arrange
		redisCache, mock, _ := setup(t)

		var result TestData

		invalidJSON := `{"field1": "value1", "field2": "not_an_int"}`

		mock.ExpectGet(testKey).SetVal(invalidJSON)

		// Act
		found, err := redisCache.Get(ctx, testKey, &result)

		// Assert
		require.Error(t, err, "Get should return an error on unmarshal failure")
		assert.False(t, found, "Get should return found=false on unmarshal error")

		var jsonErr *json.UnmarshalTypeError

		assert.ErrorAs(t, err, &jsonErr, "Error should be a json.UnmarshalTypeError")
		assert.Contains(t, err.Error(), "failed to unmarshal cache data for key "+testKey, "Error message mismatch")
		assert.NoError(t, mock.ExpectationsWereMet(), "Redis mock expectations not met")
	})
}

func TestSet(t *testing.T) {
	ctx := t.Context()
	testKey := "test:set"
	testValue := TestData{Field1: "valueSet", Field2: 456}
	jsonData, err := json.Marshal(testValue)
	if err != nil {
		t.Fatalf("failed to marshal testValue: %v", err)
	}

	t.Run("Success - With Specific TTL", func(t *testing.T) {
		// Arrange
		redisCache, mock, _ := setup(t)
		specificTTL := 5 * time.Minute

		mock.ExpectSet(testKey, jsonData, specificTTL).SetVal("OK")

		// Act
		err := redisCache.Set(ctx, testKey, testValue, specificTTL)

		// Assert
		require.NoError(t, err, "Set should not return an error on success")
		assert.NoError(t, mock.ExpectationsWereMet(), "Redis mock expectations not met")
	})

	t.Run("Success - With Default TTL (ttl=0)", func(t *testing.T) {
		// Arrange
		redisCache, mock, cfg := setup(t)

		mock.ExpectSet(testKey, jsonData, cfg.DefaultTTL).SetVal("OK")

		// Act
		err := redisCache.Set(ctx, testKey, testValue, 0) // TTL <= 0 triggers default

		// Assert
		require.NoError(t, err, "Set should not return an error when using default TTL")
		assert.NoError(t, mock.ExpectationsWereMet(), "Redis mock expectations not met")
	})

	t.Run("Success - With Default TTL (ttl<0)", func(t *testing.T) {
		// Arrange
		redisCache, mock, cfg := setup(t)

		mock.ExpectSet(testKey, jsonData, cfg.DefaultTTL).SetVal("OK")

		// Act
		err := redisCache.Set(ctx, testKey, testValue, -1*time.Second) // TTL <= 0 triggers default

		// Assert
		require.NoError(t, err, "Set should not return an error when using default TTL")
		assert.NoError(t, mock.ExpectationsWereMet(), "Redis mock expectations not met")
	})

	t.Run("Failure - Marshal Error", func(t *testing.T) {
		// Arrange
		redisCache, mock, _ := setup(t)
		unmarshallableValue := make(chan int)

		// Act
		err := redisCache.Set(ctx, testKey, unmarshallableValue, 5*time.Minute)

		// Assert
		require.Error(t, err, "Set should return an error for unmarshallable types")
		assert.Contains(t, err.Error(), "failed to marshal value for key "+testKey, "Error message mismatch")

		var jsonErr *json.UnsupportedTypeError

		assert.ErrorAs(t, err, &jsonErr, "Error should be a json.UnsupportedTypeError")
		assert.NoError(t, mock.ExpectationsWereMet(), "Redis mock expectations not met (no calls expected)")
	})

	t.Run("Failure - Redis Error", func(t *testing.T) {
		// Arrange
		redisCache, mock, _ := setup(t)
		specificTTL := 5 * time.Minute
		expectedErr := errors.New("redis SET failed")

		mock.ExpectSet(testKey, jsonData, specificTTL).SetErr(expectedErr)

		// Act
		err := redisCache.Set(ctx, testKey, testValue, specificTTL)

		// Assert
		require.Error(t, err, "Set should return an error when Redis fails")
		assert.ErrorIs(t, err, expectedErr, "Error should wrap the original Redis error")
		assert.Contains(t, err.Error(), fmt.Sprintf("failed to set key %s in redis", testKey), "Error message mismatch")
		assert.NoError(t, mock.ExpectationsWereMet(), "Redis mock expectations not met")
	})
}

func TestDelete(t *testing.T) {
	ctx := t.Context()
	testKey := "test:delete"

	t.Run("Success", func(t *testing.T) {
		// Arrange
		redisCache, mock, _ := setup(t)

		mock.ExpectDel(testKey).SetVal(1)

		// Act
		err := redisCache.Delete(ctx, testKey)

		// Assert
		require.NoError(t, err, "Delete should not return an error on success")
		assert.NoError(t, mock.ExpectationsWereMet(), "Redis mock expectations not met")
	})

	t.Run("Failure - Redis Error", func(t *testing.T) {
		// Arrange
		redisCache, mock, _ := setup(t)
		expectedErr := errors.New("redis DEL failed")

		mock.ExpectDel(testKey).SetErr(expectedErr)

		// Act
		err := redisCache.Delete(ctx, testKey)

		// Assert
		require.Error(t, err, "Delete should return an error when Redis fails")
		assert.ErrorIs(t, err, expectedErr, "Error should wrap the original Redis error")
		assert.Contains(t, err.Error(), fmt.Sprintf("failed to delete key %s from redis", testKey), "Error message mismatch")
		assert.NoError(t, mock.ExpectationsWereMet(), "Redis mock expectations not met")
	})
}

func TestClose(t *testing.T) {
	redisCache, _, _ := setup(t)
	err := redisCache.Close()
	assert.NoError(t, err, "Close should currently return nil")
}

func TestKey(t *testing.T) {
	// Arrange
	prefix := "user"
	id := "123e4567-e89b-12d3-a456-426614174000"
	expectedKey := "user:123e4567-e89b-12d3-a456-426614174000"

	generatedKey := cache.Key(prefix, id)

	assert.Equal(t, expectedKey, generatedKey, "Key function should generate the correct format")
	assert.Equal(t, "product:abc", cache.Key("product", "abc"), "Key function failed for product prefix")
	assert.Equal(t, ":", cache.Key("", ""), "Key function failed for empty prefix and id")
	assert.Equal(t, "prefix:", cache.Key("prefix", ""), "Key function failed for empty id")
	assert.Equal(t, ":id", cache.Key("", "id"), "Key function failed for empty prefix")
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "product", cache.ProductKeyPrefix)
	assert.Equal(t, "user", cache.UserKeyPrefix)
	assert.Equal(t, "order", cache.OrderKeyPrefix)
	assert.Equal(t, "cart", cache.CartKeyPrefix)
}
