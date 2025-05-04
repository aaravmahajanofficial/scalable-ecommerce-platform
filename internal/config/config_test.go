package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Creates a temporary YAML config file in a temporary directory.
func createTempConfigFile(t *testing.T, content string) (string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	err := os.WriteFile(configPath, []byte(content), 0o600)
	require.NoError(t, err, "Failed to write temporary config file")

	return configPath, func() {}
}

func createTempDefaultConfigFile(t *testing.T, content string) func() {
	t.Helper()

	configDir := "./config"

	err := os.MkdirAll(configDir, 0o755)
	if err != nil && !os.IsExist(err) {
		require.NoError(t, err, "Failed to create ./config directory")
	}

	defaultConfigPath := filepath.Join(configDir, "local.yaml")
	err = os.WriteFile(defaultConfigPath, []byte(content), 0o600)
	require.NoError(t, err, "Failed to write temporary default config file")

	return func() {
		os.Remove(defaultConfigPath)
		_ = os.Remove(configDir)
	}
}

func TestMustLoad(t *testing.T) {
	validYAML := `
env: "test"
http_server:
  address: ":8081"
database:
  PG_HOST: "dbhost"
  PG_PORT: "5433"
  PG_USER: "testuser"
  PG_PASSWORD: "testpassword"
  PG_DBNAME: "testdb"
  PG_SSLMODE: "disable"
  MAX_OPEN_CONNS: 10
  MAX_IDLE_CONNS: 5
  CONN_MAX_LIFETIME: "10m"
  CONN_MAX_IDLE_TIME: "2m"
redis:
  REDIS_HOST: "redishost"
  REDIS_USER: "redisuser"
  REDIS_PASSWORD: "redispassword"
  REDIS_DB: 1
  REDIS_PORT: "6380"
rateConfig:
  MAX_ATTEMPTS: 10
  WINDOW_SIZE: "30s"
stripe:
  STRIPE_API_KEY: "sk_test_123"
  STRIPE_WEBHOOK_SECRET: "whsec_test_123"
  STRIPE_PAYMENT_METHODS: ["card"]
  STRIPE_SUPPORTED_CURRENCIES: ["usd"]
sendgrid:
  API_KEY: "sg_test_123"
  FROM_EMAIL: "test@example.com"
  FROM_NAME: "Test Service"
  SMSENABLED: true
security:
  JWT_KEY: "testjwtkey"
  JWT_EXPIRY_HOURS: 48
otel:
  SERVICE_NAME: "test-service"
  EXPORTER_ENDPOINT: "http://otel:4318/v1/traces"
  SAMPLER_RATIO: 0.5
cache:
  default_ttl: "10m"
`
	resetEnvAndArgs := func() {
		originalArgs := os.Args

		t.Cleanup(func() { os.Args = originalArgs })
		os.Unsetenv("CONFIG_PATH")
		os.Unsetenv("ENV")
		os.Unsetenv("PG_HOST")
		os.Unsetenv("REDIS_HOST")
	}

	// Verifies values from YAML are loaded correctly
	t.Run("Load from CONFIG_PATH env var", func(t *testing.T) {
		resetEnvAndArgs()

		configPath, _ := createTempConfigFile(t, validYAML)
		t.Setenv("CONFIG_PATH", configPath)

		cfg, err := LoadConfigFromPath(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "test", cfg.Env)
		assert.Equal(t, ":8081", cfg.HTTPServer.Addr)
		assert.Equal(t, "dbhost", cfg.Database.Host)
		assert.Equal(t, "redisuser", cfg.RedisConnect.Username)
		assert.Equal(t, 48, cfg.Security.JWTExpiryHours)
		assert.Equal(t, 10*time.Minute, cfg.Cache.DefaultTTL)
	})

	// Simulates passing CLI argument -config path/to/config
	t.Run("Load from -config flag", func(t *testing.T) {
		resetEnvAndArgs()

		configPath, _ := createTempConfigFile(t, validYAML)

		os.Args = []string{"cmd", "-config", configPath}

		cfg, err := LoadConfigFromPath(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "test", cfg.Env)
		assert.Equal(t, "dbhost", cfg.Database.Host)
	})

	// Uses default config path when no CONFIG_PATH or CLI flag is given
	t.Run("Load from default ./config/local.yaml", func(t *testing.T) {
		resetEnvAndArgs()

		configPath, _ := createTempConfigFile(t, validYAML)
		os.Args = []string{"cmd"}
		cleanupDefault := createTempDefaultConfigFile(t, validYAML)
		t.Cleanup(cleanupDefault)

		cfg, err := LoadConfigFromPath(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "test", cfg.Env)
		assert.Equal(t, "dbhost", cfg.Database.Host)
	})

	// Verifies envs override the YAML values
	t.Run("Environment variable override", func(t *testing.T) {
		resetEnvAndArgs()

		configPath, _ := createTempConfigFile(t, validYAML)
		t.Setenv("CONFIG_PATH", configPath)

		t.Setenv("ENV", "production")
		t.Setenv("PG_HOST", "prod-db")
		t.Setenv("REDIS_HOST", "prod-redis")
		t.Setenv("JWT_KEY", "prodjwtkey")
		t.Setenv("PG_USER", "produser")
		t.Setenv("PG_PASSWORD", "prodpass")
		t.Setenv("PG_DBNAME", "proddb")
		t.Setenv("REDIS_USER", "prodredisuser")
		t.Setenv("REDIS_PASSWORD", "prodredispass")

		cfg, err := LoadConfigFromPath(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "production", cfg.Env)
		assert.Equal(t, "prod-db", cfg.Database.Host)
		assert.Equal(t, "prod-redis", cfg.RedisConnect.Host)
		assert.Equal(t, "prodpass", cfg.Database.Password)
		assert.Equal(t, "prodredispass", cfg.RedisConnect.Password)
		assert.Equal(t, "prodjwtkey", cfg.Security.JWTKey)
	})
}

func TestDatabaseGetDSN(t *testing.T) {
	dbConfig := Database{
		Host:     "localhost",
		Port:     "5432",
		User:     "user",
		Password: "password",
		Name:     "dbname",
		SSLMode:  "disable",
	}

	expectedBaseDSN := "postgresql://user:password@localhost:5432/dbname?sslmode=disable"

	t.Run("DSN from struct values", func(t *testing.T) {
		// clear any related environment variables to prevent interference
		os.Unsetenv("PG_HOST")
		os.Unsetenv("PG_PORT")
		os.Unsetenv("PG_USER")
		os.Unsetenv("PG_PASSWORD")
		os.Unsetenv("PG_DBNAME")
		os.Unsetenv("PG_SSLMODE")

		dsn := dbConfig.GetDSN()
		assert.Equal(t, expectedBaseDSN, dsn)
	})

	createMinimalValidConfig := func(t *testing.T, _, _ map[string]string) (string, func()) {
		t.Helper()

		content := `
env: "test-dsn"
http_server: {address: ":9999"}
database:
  PG_HOST: "filehost"
  PG_PORT: "5000"
  PG_USER: "fileuser"
  PG_PASSWORD: "filepassword"
  PG_DBNAME: "filedb"
  PG_SSLMODE: "prefer"
redis:
  REDIS_HOST: "fileredishost"
  REDIS_PORT: "6000"
  REDIS_USER: "fileredisuser"
  REDIS_PASSWORD: "fileredispassword"
security: {JWT_KEY: "filekey"} # Required field
`

		return createTempConfigFile(t, content)
	}

	t.Run("DSN with environment variable overrides", func(t *testing.T) {
		configPath, cleanup := createMinimalValidConfig(t, nil, nil)
		t.Cleanup(cleanup)

		t.Setenv("PG_HOST", "envhost")
		t.Setenv("PG_PORT", "5433")
		t.Setenv("PG_USER", "envuser")
		t.Setenv("PG_PASSWORD", "envpass")
		t.Setenv("PG_DBNAME", "envdb")
		t.Setenv("PG_SSLMODE", "require")

		t.Cleanup(func() {
			os.Unsetenv("PG_HOST")
			os.Unsetenv("PG_PORT")
			os.Unsetenv("PG_USER")
			os.Unsetenv("PG_PASSWORD")
			os.Unsetenv("PG_DBNAME")
			os.Unsetenv("PG_SSLMODE")
			os.Unsetenv("REDIS_USER")
			os.Unsetenv("REDIS_PASSWORD")
			os.Unsetenv("JWT_KEY")
		})

		loadedCfg, err := LoadConfigFromPath(configPath)
		require.NoError(t, err)
		require.NotNil(t, loadedCfg)

		expectedEnvDSN := "postgresql://envuser:envpass@envhost:5433/envdb?sslmode=require"
		dsn := loadedCfg.Database.GetDSN()
		assert.Equal(t, expectedEnvDSN, dsn)
	})

	t.Run("DSN with partial environment variable overrides", func(t *testing.T) {
		configPath, cleanup := createMinimalValidConfig(t, nil, nil)
		t.Cleanup(cleanup)

		t.Setenv("PG_HOST", "envhost2")
		t.Setenv("PG_PASSWORD", "envpass2")

		t.Cleanup(func() {
			os.Unsetenv("PG_HOST")
			os.Unsetenv("PG_PASSWORD")
			os.Unsetenv("REDIS_USER")
			os.Unsetenv("REDIS_PASSWORD")
			os.Unsetenv("JWT_KEY")
		})

		loadedCfg, err := LoadConfigFromPath(configPath)
		require.NoError(t, err)
		require.NotNil(t, loadedCfg)

		expectedPartialEnvDSN := "postgresql://fileuser:envpass2@envhost2:5000/filedb?sslmode=prefer"
		dsn := loadedCfg.Database.GetDSN()
		assert.Equal(t, expectedPartialEnvDSN, dsn)
	})
}

func TestRedisConnectGetDSN(t *testing.T) {
	redisConfig := RedisConnect{
		Host:     "localhost",
		Username: "user",
		Password: "password",
		Port:     "6379",
		DB:       0,
	}

	expectedBaseDSN := "redis://user:password@localhost:6379"

	createMinimalValidConfig := func(t *testing.T) (string, func()) {
		t.Helper()

		content := `
env: "test-dsn-redis"
http_server: {address: ":9998"}
database: # Required fields
  PG_USER: "fileuser"
  PG_PASSWORD: "filepassword"
  PG_DBNAME: "filedb"
redis:
  REDIS_HOST: "fileredishost"
  REDIS_PORT: "6000"
  REDIS_USER: "fileredisuser"
  REDIS_PASSWORD: "fileredispassword"
security: {JWT_KEY: "filekey"} # Required field
`

		return createTempConfigFile(t, content)
	}

	t.Run("DSN from struct values", func(t *testing.T) {
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_USER")
		os.Unsetenv("REDIS_PASSWORD")
		os.Unsetenv("REDIS_PORT")

		dsn := redisConfig.GetDSN()
		assert.Equal(t, expectedBaseDSN, dsn)
	})

	t.Run("DSN with environment variable overrides", func(t *testing.T) {
		configPath, cleanup := createMinimalValidConfig(t)
		t.Cleanup(cleanup)

		t.Setenv("REDIS_HOST", "envredishost")
		t.Setenv("REDIS_USER", "envredisuser")
		t.Setenv("REDIS_PASSWORD", "envredispass")
		t.Setenv("REDIS_PORT", "16379")

		t.Cleanup(func() {
			os.Unsetenv("REDIS_HOST")
			os.Unsetenv("REDIS_USER")
			os.Unsetenv("REDIS_PASSWORD")
			os.Unsetenv("REDIS_PORT")
			os.Unsetenv("PG_USER")
			os.Unsetenv("PG_PASSWORD")
			os.Unsetenv("PG_DBNAME")
			os.Unsetenv("JWT_KEY")
		})

		loadedCfg, err := LoadConfigFromPath(configPath)
		require.NoError(t, err)
		require.NotNil(t, loadedCfg)

		expectedEnvDSN := "redis://envredisuser:envredispass@envredishost:16379"
		dsn := loadedCfg.RedisConnect.GetDSN()
		assert.Equal(t, expectedEnvDSN, dsn)
	})

	t.Run("DSN with partial environment variable overrides", func(t *testing.T) {
		configPath, cleanup := createMinimalValidConfig(t)
		t.Cleanup(cleanup)

		t.Setenv("REDIS_HOST", "envredishost2")
		t.Setenv("REDIS_PASSWORD", "envredispass2")

		t.Cleanup(func() {
			os.Unsetenv("REDIS_HOST")
			os.Unsetenv("REDIS_PASSWORD")
			os.Unsetenv("PG_USER")
			os.Unsetenv("PG_PASSWORD")
			os.Unsetenv("PG_DBNAME")
			os.Unsetenv("JWT_KEY")
		})

		loadedCfg, err := LoadConfigFromPath(configPath)
		require.NoError(t, err)
		require.NotNil(t, loadedCfg)

		expectedPartialEnvDSN := "redis://fileredisuser:envredispass2@envredishost2:6000"
		dsn := loadedCfg.RedisConnect.GetDSN()
		assert.Equal(t, expectedPartialEnvDSN, dsn)
	})

	t.Run("DSN with empty username from struct", func(t *testing.T) {
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_USER")
		os.Unsetenv("REDIS_PASSWORD")
		os.Unsetenv("REDIS_PORT")

		configWithEmptyUser := RedisConnect{
			Host:     "localhost",
			Username: "",
			Password: "password",
			Port:     "6379",
		}
		expectedDSN := "redis://:password@localhost:6379"
		dsn := configWithEmptyUser.GetDSN()
		assert.Equal(t, expectedDSN, dsn)
	})

	t.Run("DSN with empty username and password from struct", func(t *testing.T) {
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_USER")
		os.Unsetenv("REDIS_PASSWORD")
		os.Unsetenv("REDIS_PORT")

		configWithEmptyCreds := RedisConnect{
			Host:     "localhost",
			Username: "",
			Password: "",
			Port:     "6379",
		}
		expectedDSN := "redis://:@localhost:6379"
		dsn := configWithEmptyCreds.GetDSN()
		assert.Equal(t, expectedDSN, dsn)
	})
}

func TestMustLoad_SpecificFieldCheck(t *testing.T) {
	resetEnvAndArgs := func() {
		originalArgs := os.Args

		t.Cleanup(func() { os.Args = originalArgs })
		os.Unsetenv("CONFIG_PATH")
		os.Unsetenv("CACHE_DEFAULT_TTL")
		t.Setenv("ENV", "test")
		t.Setenv("PG_USER", "test")
		t.Setenv("PG_PASSWORD", "test")
		t.Setenv("PG_DBNAME", "test")
		t.Setenv("REDIS_USER", "test")
		t.Setenv("REDIS_PASSWORD", "test")
		t.Setenv("JWT_KEY", "test")
	}

	t.Run("Cache TTL from file", func(t *testing.T) {
		resetEnvAndArgs()

		yamlContent := `
env: "test-cache"
cache:
  default_ttl: "15m"
http_server: {address: ":1111"}
database: {PG_USER: u, PG_PASSWORD: p, PG_DBNAME: d}
redis: {REDIS_USER: u, REDIS_PASSWORD: p}
security: {JWT_KEY: k}
`
		configPath, _ := createTempConfigFile(t, yamlContent)
		t.Setenv("CONFIG_PATH", configPath)

		cfg, err := LoadConfigFromPath(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, 15*time.Minute, cfg.Cache.DefaultTTL)
	})

	t.Run("Cache TTL overridden by environment", func(t *testing.T) {
		resetEnvAndArgs()

		yamlContent := `
env: "test-cache-env"
cache:
  default_ttl: "15m"
# Add other required fields
http_server: {address: ":1111"}
database: {PG_USER: u, PG_PASSWORD: p, PG_DBNAME: d}
redis: {REDIS_USER: u, REDIS_PASSWORD: p}
security: {JWT_KEY: k}
`
		configPath, _ := createTempConfigFile(t, yamlContent)
		t.Setenv("CONFIG_PATH", configPath)
		t.Setenv("CACHE_DEFAULT_TTL", "30m")

		cfg, err := LoadConfigFromPath(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, 30*time.Minute, cfg.Cache.DefaultTTL)
	})

	t.Run("Cache TTL default value", func(t *testing.T) {
		resetEnvAndArgs()

		yamlContent := `
env: "test-cache-default"
# Cache section omitted to test default
http_server: {address: ":1111"}
database: {PG_USER: u, PG_PASSWORD: p, PG_DBNAME: d}
redis: {REDIS_USER: u, REDIS_PASSWORD: p}
security: {JWT_KEY: k}
`
		configPath, _ := createTempConfigFile(t, yamlContent)
		t.Setenv("CONFIG_PATH", configPath)
		os.Unsetenv("CACHE_DEFAULT_TTL")

		cfg, err := LoadConfigFromPath(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, 5*time.Minute, cfg.Cache.DefaultTTL)
	})
}
