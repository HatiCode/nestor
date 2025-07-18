package dynamodb

import (
	"fmt"
	"strings"
	"time"
)

const DefaultCatalogTableName = "nestor-catalog"

type Config struct {
	TableName         string        `yaml:"table_name" json:"table_name"`
	Region            string        `yaml:"region" json:"region" validate:"required"`
	Endpoint          string        `yaml:"endpoint" json:"endpoint"`
	ConsistentReads   bool          `yaml:"consistent_reads" json:"consistent_reads" default:"false"`
	QueryTimeout      time.Duration `yaml:"query_timeout" json:"query_timeout" default:"30s"`
	MaxRetries        int           `yaml:"max_retries" json:"max_retries" default:"3"`
	MaxBatchSize      int           `yaml:"max_batch_size" json:"max_batch_size" default:"25"`
	AutoCreateTable   bool          `yaml:"auto_create_table" json:"auto_create_table" default:"false"`
	VerifyTableSchema bool          `yaml:"verify_table_schema" json:"verify_table_schema" default:"true"`
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if c.Region == "" {
		return fmt.Errorf("region is required")
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}

	if c.MaxBatchSize < 1 || c.MaxBatchSize > 25 {
		return fmt.Errorf("max_batch_size must be between 1 and 25 (DynamoDB limit)")
	}

	if c.QueryTimeout <= 0 {
		return fmt.Errorf("query_timeout must be positive")
	}

	if c.TableName != "" && len(c.TableName) < 3 {
		return fmt.Errorf("table_name must be at least 3 characters long")
	}

	if c.Endpoint != "" && !strings.HasPrefix(c.Endpoint, "http") {
		return fmt.Errorf("endpoint must be a valid URL starting with http:// or https://")
	}

	return nil
}

func (c *Config) IsLocal() bool {
	return c.Endpoint != ""
}

func (c *Config) GetTableName() string {
	if c.TableName == "" {
		return DefaultCatalogTableName
	}
	return c.TableName
}

func DefaultConfig() *Config {
	return &Config{
		Region:            "eu-central-1",
		ConsistentReads:   false,
		QueryTimeout:      30 * time.Second,
		MaxRetries:        1,
		MaxBatchSize:      25,
		AutoCreateTable:   false,
		VerifyTableSchema: true,
	}
}

func LocalConfig() *Config {
	return &Config{
		TableName:         "nestor-catalog-local",
		Region:            "eu-central-1",
		Endpoint:          "http://localhost:8000",
		ConsistentReads:   true,
		QueryTimeout:      10 * time.Second,
		MaxRetries:        2,
		MaxBatchSize:      10,
		AutoCreateTable:   true,
		VerifyTableSchema: true,
	}
}
