package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplaceEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "简单替换",
			content:  "api_key: ${API_KEY}",
			envVars:  map[string]string{"API_KEY": "test-key"},
			expected: "api_key: test-key",
		},
		{
			name:     "带默认值",
			content:  "port: ${PORT:8080}",
			envVars:  map[string]string{},
			expected: "port: 8080",
		},
		{
			name:     "环境变量覆盖默认值",
			content:  "port: ${PORT:8080}",
			envVars:  map[string]string{"PORT": "9090"},
			expected: "port: 9090",
		},
		{
			name:     "多个变量",
			content:  "host: ${HOST:localhost}\nport: ${PORT:8080}",
			envVars:  map[string]string{"HOST": "0.0.0.0"},
			expected: "host: 0.0.0.0\nport: 8080",
		},
		{
			name:     "空默认值",
			content:  "password: ${PASSWORD:}",
			envVars:  map[string]string{},
			expected: "password: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置环境变量
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			result := replaceEnvVars(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name         string
		durationStr  string
		defaultValue time.Duration
		expected     time.Duration
		expectError  bool
	}{
		{
			name:         "有效的时间间隔",
			durationStr:  "5m",
			defaultValue: 1 * time.Minute,
			expected:     5 * time.Minute,
			expectError:  false,
		},
		{
			name:         "空字符串使用默认值",
			durationStr:  "",
			defaultValue: 10 * time.Minute,
			expected:     10 * time.Minute,
			expectError:  false,
		},
		{
			name:         "无效的时间间隔",
			durationStr:  "invalid",
			defaultValue: 1 * time.Minute,
			expected:     1 * time.Minute,
			expectError:  true,
		},
		{
			name:         "复杂的时间间隔",
			durationStr:  "1h30m",
			defaultValue: 1 * time.Minute,
			expected:     90 * time.Minute,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDuration(tt.durationStr, tt.defaultValue)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	tests := []struct {
		name       string
		env        string
		configFile string
		configDir  string
		expected   string
	}{
		{
			name:     "开发环境",
			env:      "development",
			expected: "config/dev.yaml",
		},
		{
			name:     "生产环境",
			env:      "production",
			expected: "config/prod.yaml",
		},
		{
			name:     "测试环境",
			env:      "test",
			expected: "config/test.yaml",
		},
		{
			name:     "默认环境",
			env:      "unknown",
			expected: "config/config.yaml",
		},
		{
			name:       "自定义配置文件",
			env:        "production",
			configFile: "/custom/path/config.yaml",
			expected:   "/custom/path/config.yaml",
		},
		{
			name:      "自定义配置目录",
			env:       "production",
			configDir: "/custom/config",
			expected:  "/custom/config/prod.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置环境变量
			if tt.configFile != "" {
				os.Setenv("CONFIG_FILE", tt.configFile)
				defer os.Unsetenv("CONFIG_FILE")
			}
			if tt.configDir != "" {
				os.Setenv("CONFIG_DIR", tt.configDir)
				defer os.Unsetenv("CONFIG_DIR")
			}

			result := getConfigPath(tt.env)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name     string
		appEnv   string
		goEnv    string
		expected string
	}{
		{
			name:     "使用APP_ENV",
			appEnv:   "production",
			expected: "production",
		},
		{
			name:     "使用GO_ENV",
			goEnv:    "development",
			expected: "development",
		},
		{
			name:     "APP_ENV优先于GO_ENV",
			appEnv:   "production",
			goEnv:    "development",
			expected: "production",
		},
		{
			name:     "默认为development",
			expected: "development",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清理环境变量
			os.Unsetenv("APP_ENV")
			os.Unsetenv("GO_ENV")

			// 设置环境变量
			if tt.appEnv != "" {
				os.Setenv("APP_ENV", tt.appEnv)
				defer os.Unsetenv("APP_ENV")
			}
			if tt.goEnv != "" {
				os.Setenv("GO_ENV", tt.goEnv)
				defer os.Unsetenv("GO_ENV")
			}

			result := GetEnv()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadFromYAML(t *testing.T) {
	// 创建临时配置文件
	tempFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// 写入测试配置
	configContent := `
server:
  port: "8080"
  host: "localhost"
  mode: debug

genkit:
  provider: google
  api_key: "test-api-key"
  model: "gemini-2.5-flash"
  default_temperature: 0.7
  default_max_tokens: 2000
  log_level: debug
  timeout: "30s"

database:
  host: "localhost"
  port: 5432
  database: "test_db"
  user: "postgres"
  password: "postgres"
  ssl_mode: "disable"
  max_connections: 10
  max_idle_conns: 5
  conn_max_lifetime: "5m"
  log_level: "info"

redis:
  host: "localhost"
  port: 6379
  password: ""
  database: 0
  enabled: true

log:
  level: "debug"
  format: "json"
  enable_file: true
  log_dir: "logs"
  enable_console: true

session:
  timeout: "30m"
  cleanup_interval: "5m"
  summary_threshold: 50
  default_page_size: 20
  max_page_size: 100
  max_title_length: 255

models:
  dir: "./models"

auth:
  jwt_secret: "test-secret-key-for-testing-only-min-32-chars"
  jwt_issuer: "genkit-test"
  jwt_audience: "genkit-test-api"
  access_token_ttl: "60m"
  refresh_token_ttl: "720h"
  bcrypt_cost: 10
  max_login_attempts: 5
  login_attempt_window: "15m"
  password_min_length: 8
  enable_refresh_rotation: true
  tenant_identify_strategy: "header"
  token_cleanup_interval: "1h"
  enable_token_blacklist: true

bootstrap:
  admin_email: "admin@test.local"
  admin_password: "Test1234"
  admin_display_name: "Test Admin"
  tenant_name: "Test Tenant"
  tenant_domain: "test.local"

monitoring:
  prometheus_port: 9090
  jaeger_endpoint: ""
  enable_tracing: false
  enable_metrics: false
  metrics_path: "/metrics"
  tracing_sampling: 0.0

cache:
  namespace: "genkit:test"
  default_ttl: "5m"
  enable_warmup: false
  warmup_interval: "10m"
  context_ttl: "5m"
  vector_search_ttl: "30m"
  summary_ttl: "1h"
  session_list_ttl: "10m"
  token_usage_ttl: "5m"

vector:
  provider: "google"
  embedding_model: "text-embedding-004"
  dimension: 768
  batch_size: 10
  timeout: "30s"
`

	_, err = tempFile.WriteString(configContent)
	require.NoError(t, err)
	tempFile.Close()

	// 加载配置
	config, err := LoadFromYAML(tempFile.Name())
	require.NoError(t, err)
	require.NotNil(t, config)

	// 验证配置
	assert.Equal(t, "8080", config.Server.Port)
	assert.Equal(t, "localhost", config.Server.Host)
	assert.Equal(t, "test-api-key", config.Genkit.APIKey)
	assert.Equal(t, "gemini-2.5-flash", config.Genkit.Model)
	assert.Equal(t, 0.7, config.Genkit.DefaultTemperature)
	assert.Equal(t, 2000, config.Genkit.DefaultMaxTokens)
	assert.Equal(t, "localhost", config.Database.Host)
	assert.Equal(t, "5432", config.Database.Port)
	assert.Equal(t, "test_db", config.Database.DBName)
	assert.Equal(t, 10, config.Database.MaxOpenConns)
	assert.Equal(t, 5, config.Database.MaxIdleConns)
	assert.Equal(t, 30*time.Minute, config.Session.Timeout)
	assert.Equal(t, 5*time.Minute, config.Session.CleanupInterval)
	assert.Equal(t, 60*time.Minute, config.Auth.AccessTokenTTL)
	assert.Equal(t, 720*time.Hour, config.Auth.RefreshTokenTTL)
}

func TestLoadFromYAMLWithEnvVars(t *testing.T) {
	// 设置环境变量
	os.Setenv("TEST_API_KEY", "env-api-key")
	os.Setenv("TEST_DB_PASSWORD", "env-password")
	defer os.Unsetenv("TEST_API_KEY")
	defer os.Unsetenv("TEST_DB_PASSWORD")

	// 创建临时配置文件
	tempFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// 写入包含环境变量的配置
	configContent := `
server:
  port: "${SERVER_PORT:8080}"
  host: "localhost"
  mode: debug

genkit:
  provider: google
  api_key: "${TEST_API_KEY}"
  model: "gemini-2.5-flash"
  default_temperature: 0.7
  default_max_tokens: 2000
  log_level: debug
  timeout: "30s"

database:
  host: "localhost"
  port: 5432
  database: "test_db"
  user: "postgres"
  password: "${TEST_DB_PASSWORD}"
  ssl_mode: "disable"
  max_connections: 10
  max_idle_conns: 5
  conn_max_lifetime: "5m"
  log_level: "info"

redis:
  host: "localhost"
  port: 6379
  password: ""
  database: 0
  enabled: true

log:
  level: "debug"
  format: "json"
  enable_file: true
  log_dir: "logs"
  enable_console: true

session:
  timeout: "30m"
  cleanup_interval: "5m"
  summary_threshold: 50
  default_page_size: 20
  max_page_size: 100
  max_title_length: 255

models:
  dir: "./models"

auth:
  jwt_secret: "test-secret-key-for-testing-only-min-32-chars"
  jwt_issuer: "genkit-test"
  jwt_audience: "genkit-test-api"
  access_token_ttl: "60m"
  refresh_token_ttl: "720h"
  bcrypt_cost: 10
  max_login_attempts: 5
  login_attempt_window: "15m"
  password_min_length: 8
  enable_refresh_rotation: true
  tenant_identify_strategy: "header"
  token_cleanup_interval: "1h"
  enable_token_blacklist: true

bootstrap:
  admin_email: "admin@test.local"
  admin_password: "Test1234"
  admin_display_name: "Test Admin"
  tenant_name: "Test Tenant"
  tenant_domain: "test.local"

monitoring:
  prometheus_port: 9090
  jaeger_endpoint: ""
  enable_tracing: false
  enable_metrics: false
  metrics_path: "/metrics"
  tracing_sampling: 0.0

cache:
  namespace: "genkit:test"
  default_ttl: "5m"
  enable_warmup: false
  warmup_interval: "10m"
  context_ttl: "5m"
  vector_search_ttl: "30m"
  summary_ttl: "1h"
  session_list_ttl: "10m"
  token_usage_ttl: "5m"

vector:
  provider: "google"
  embedding_model: "text-embedding-004"
  dimension: 768
  batch_size: 10
  timeout: "30s"
`

	_, err = tempFile.WriteString(configContent)
	require.NoError(t, err)
	tempFile.Close()

	// 加载配置
	config, err := LoadFromYAML(tempFile.Name())
	require.NoError(t, err)
	require.NotNil(t, config)

	// 验证环境变量替换
	assert.Equal(t, "8080", config.Server.Port) // 使用默认值
	assert.Equal(t, "env-api-key", config.Genkit.APIKey) // 使用环境变量
	assert.Equal(t, "env-password", config.Database.Password) // 使用环境变量
}
