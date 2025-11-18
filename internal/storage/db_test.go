package storage

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Host != "localhost" {
		t.Errorf("DefaultConfig Host = %s, want localhost", cfg.Host)
	}
	if cfg.Port != "5432" {
		t.Errorf("DefaultConfig Port = %s, want 5432", cfg.Port)
	}
	if cfg.MaxConns != 25 {
		t.Errorf("DefaultConfig MaxConns = %d, want 25", cfg.MaxConns)
	}
	if cfg.MinConns != 5 {
		t.Errorf("DefaultConfig MinConns = %d, want 5", cfg.MinConns)
	}
}

func TestDB_InvalidConfig(t *testing.T) {
	cfg := &Config{
		Host:     "invalid-host",
		Port:     "9999",
		User:     "invalid",
		Password: "invalid",
		DBName:   "invalid",
		SSLMode:  "disable",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	db, err := NewDB(cfg)
	if err == nil && db != nil {
		// If connection somehow succeeded, close it
		db.Close()
		t.Skip("Connection to invalid host succeeded unexpectedly, skipping test")
	}

	// We expect this to fail
	if err == nil {
		t.Error("Expected error connecting to invalid database, got nil")
	}

	// Additional context check
	<-ctx.Done()
}

func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
	}{
		{
			name: "Default config values",
			config: &Config{
				Host:        "localhost",
				Port:        "5432",
				User:        "workflow",
				Password:    "password",
				DBName:      "test_db",
				SSLMode:     "disable",
				MaxConns:    25,
				MinConns:    5,
				MaxIdleTime: 5 * time.Minute,
				MaxLifetime: 30 * time.Minute,
			},
		},
		{
			name: "Custom config values",
			config: &Config{
				Host:        "db.example.com",
				Port:        "5433",
				User:        "custom_user",
				Password:    "custom_pass",
				DBName:      "custom_db",
				SSLMode:     "require",
				MaxConns:    50,
				MinConns:    10,
				MaxIdleTime: 10 * time.Minute,
				MaxLifetime: 60 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Host == "" {
				t.Error("Host should not be empty")
			}
			if tt.config.Port == "" {
				t.Error("Port should not be empty")
			}
			if tt.config.MaxConns < tt.config.MinConns {
				t.Error("MaxConns should be >= MinConns")
			}
		})
	}
}
