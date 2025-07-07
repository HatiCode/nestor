package dynamodb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/HatiCode/nestor/shared/pkg/logging"
)

type Client struct {
	client    *dynamodb.Client
	config    *Config
	logger    logging.Logger
	tableName string
}

func NewClient(cfg *Config, logger logging.Logger) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	logger.Info("creating DynamoDB client",
		"region", cfg.Region,
		"table", cfg.GetTableName(),
		"endpoint", cfg.Endpoint,
		"is_local", cfg.IsLocal())

	awsConfig, err := loadAwsConfig(cfg)
	if err != nil {
		logger.Error("failed to load AWS config", "error", err, "region", cfg.Region)
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := dynamodb.NewFromConfig(awsConfig)
	clientLogger := logger.With("component", "dynamodb", "table", cfg.GetTableName())
	clientLogger.Info("DynamoDB client created successfully")

	return &Client{
		client:    client,
		config:    cfg,
		logger:    clientLogger,
		tableName: cfg.GetTableName(),
	}, nil
}

func loadAwsConfig(cfg *Config) (aws.Config, error) {
	awsConfig, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.Region),
		config.WithRetryMaxAttempts(cfg.MaxRetries),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	if cfg.IsLocal() {
		awsConfig.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...any) (aws.Endpoint, error) {
				if service == dynamodb.ServiceID {
					return aws.Endpoint{
						URL: cfg.Endpoint,
					}, nil
				}
				return aws.Endpoint{}, fmt.Errorf("unknown endpoint for service %s", service)
			})
	}

	return awsConfig, nil
}

func (c *Client) GetClient() *dynamodb.Client {
	return c.client
}

func (c *Client) GetTableName() string {
	return c.tableName
}

func (c *Client) Ping(ctx context.Context) error {
	c.logger.Debug("pinging DynamoDB", "operation", "DescribeTable")

	_, err := c.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(c.tableName),
	})

	if err != nil {
		if _, ok := err.(*types.ResourceNotFoundException); ok {
			c.logger.Debug("ping successful, table does not exist yet")
			return nil
		}
		c.logger.Error("ping failed", "error", err)
		return fmt.Errorf("failed to ping DynamoDB: %w", err)
	}

	c.logger.Debug("ping successful, table exists")
	return nil
}

func (c *Client) TableExists(ctx context.Context) (bool, error) {
	_, err := c.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(c.tableName),
	})
	if err != nil {
		if _, ok := err.(*types.ResourceNotFoundException); ok {
			return false, nil
		}
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}

	return true, nil
}

func (c *Client) GetTableDescription(ctx context.Context) (*types.TableDescription, error) {
	output, err := c.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(c.tableName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe table: %w", err)
	}

	return output.Table, nil
}

func (c *Client) WaitForTable(ctx context.Context) error {
	c.logger.Info("waiting for table to be active")
	waiter := dynamodb.NewTableExistsWaiter(c.client)

	err := waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(c.tableName),
	}, 5*time.Minute)
	if err != nil {
		c.logger.Error("table did not become active", "error", err, "timeout", "5m")
		return fmt.Errorf("table did not become active: %w", err)
	}

	c.logger.Info("table is now active")
	return nil
}

func (c *Client) GetItem(ctx context.Context, input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	if input.TableName == nil {
		input.TableName = aws.String(c.tableName)
	}

	if input.ConsistentRead == nil {
		input.ConsistentRead = aws.Bool(c.config.ConsistentReads)
	}

	c.logger.DebugContext(ctx, "executing GetItem", "operation", "GetItem", "consistent_read", *input.ConsistentRead)

	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.QueryTimeout)
	defer cancel()

	result, err := c.client.GetItem(timeoutCtx, input)
	if err != nil {
		c.logger.ErrorContext(ctx, "GetItem failed", "error", err, "operation", "GetItem")
		return nil, err
	}

	c.logger.DebugContext(ctx, "GetItem completed",
		"operation", "GetItem",
		"item_found", result.Item != nil)

	return result, nil
}

func (c *Client) PutItem(ctx context.Context, input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	if input.TableName == nil {
		input.TableName = aws.String(c.tableName)
	}

	c.logger.DebugContext(ctx, "executing PutItem", "operation", "PutItem")

	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.QueryTimeout)
	defer cancel()

	result, err := c.client.PutItem(timeoutCtx, input)
	if err != nil {
		c.logger.ErrorContext(ctx, "PutItem failed", "error", err, "operation", "PutItem")
	}

	return result, nil
}

func (c *Client) Query(ctx context.Context, input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	if input.TableName == nil {
		input.TableName = aws.String(c.tableName)
	}

	if input.ConsistentRead == nil {
		input.ConsistentRead = aws.Bool(c.config.ConsistentReads)
	}

	c.logger.DebugContext(ctx, "executing Query",
		"operation", "Query",
		"consistent_read", *input.ConsistentRead,
		"index_name", aws.ToString(input.IndexName))

	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.QueryTimeout)
	defer cancel()

	result, err := c.client.Query(timeoutCtx, input)
	if err != nil {
		c.logger.ErrorContext(ctx, "Query failed", "error", err, "operation", "Query")
		return nil, err
	}

	c.logger.DebugContext(ctx, "Query completed",
		"operation", "Query",
		"count", result.Count,
		"scanned_count", result.ScannedCount)

	return result, nil
}

func (c *Client) Scan(ctx context.Context, input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error) {
	if input.TableName == nil {
		input.TableName = aws.String(c.tableName)
	}

	if input.ConsistentRead == nil {
		input.ConsistentRead = aws.Bool(c.config.ConsistentReads)
	}

	c.logger.DebugContext(ctx, "executing Scan",
		"operation", "Scan",
		"consistent_read", *input.ConsistentRead)

	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.QueryTimeout)
	defer cancel()

	result, err := c.client.Scan(timeoutCtx, input)
	if err != nil {
		c.logger.ErrorContext(ctx, "Scan failed", "error", err, "operation", "Scan")
		return nil, err
	}

	c.logger.DebugContext(ctx, "Scan completed",
		"operation", "Scan",
		"count", result.Count,
		"scanned_count", result.ScannedCount)

	return result, nil
}

func (c *Client) BatchGetItem(ctx context.Context, input *dynamodb.BatchGetItemInput) (*dynamodb.BatchGetItemOutput, error) {
	c.logger.DebugContext(ctx, "executing BatchGetItem", "operation", "BatchGetItem")

	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.QueryTimeout)
	defer cancel()

	result, err := c.client.BatchGetItem(timeoutCtx, input)
	if err != nil {
		c.logger.ErrorContext(ctx, "BatchGetItem failed", "error", err, "operation", "BatchGetItem")
		return nil, err
	}

	c.logger.DebugContext(ctx, "BatchGetItem completed", "operation", "BatchGetItem")
	return result, nil
}

func (c *Client) BatchWriteItem(ctx context.Context, input *dynamodb.BatchWriteItemInput) (*dynamodb.BatchWriteItemOutput, error) {
	c.logger.DebugContext(ctx, "executing BatchWriteItem", "operation", "BatchWriteItem")

	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.QueryTimeout)
	defer cancel()

	result, err := c.client.BatchWriteItem(timeoutCtx, input)
	if err != nil {
		c.logger.ErrorContext(ctx, "BatchWriteItem failed", "error", err, "operation", "BatchWriteItem")
		return nil, err
	}

	c.logger.DebugContext(ctx, "BatchWriteItem completed", "operation", "BatchWriteItem")
	return result, nil
}

func (c *Client) UpdateItem(ctx context.Context, input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error) {
	if input.TableName == nil {
		input.TableName = aws.String(c.tableName)
	}

	c.logger.DebugContext(ctx, "executing UpdateItem", "operation", "UpdateTime")

	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.QueryTimeout)
	defer cancel()

	result, err := c.client.UpdateItem(timeoutCtx, input)
	if err != nil {
		c.logger.ErrorContext(ctx, "UpdateItem failed", "error", err, "operation", "UpdateItem")
		return nil, err
	}

	c.logger.DebugContext(ctx, "UpdateItem completed", "operation", "UpdateItem")
	return result, nil
}

func (c *Client) DeleteItem(ctx context.Context, input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {

	if input.TableName == nil {
		input.TableName = aws.String(c.tableName)
	}

	c.logger.DebugContext(ctx, "executing DeleteItem", "operation", "DeleteItem")

	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.QueryTimeout)
	defer cancel()

	result, err := c.client.DeleteItem(timeoutCtx, input)
	if err != nil {
		c.logger.ErrorContext(ctx, "DeleteItem failed", "error", err, "operation", "DeleteItem")
		return nil, err
	}

	c.logger.DebugContext(ctx, "DeleteItem completed", "operation", "DeleteItem")
	return result, nil
}

func (c *Client) CreateTable(ctx context.Context, input *dynamodb.CreateTableInput) (*dynamodb.CreateTableOutput, error) {
	if input.TableName == nil {
		input.TableName = aws.String(c.tableName)
	}

	c.logger.InfoContext(ctx, "creating table", "operation", "CreateTable")

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	result, err := c.client.CreateTable(timeoutCtx, input)
	if err != nil {
		c.logger.ErrorContext(ctx, "CreateTable failed", "error", err, "operation", "CreateTable")
		return nil, err
	}

	c.logger.InfoContext(ctx, "CreateTable completed", "operation", "CreateTable")
	return result, nil
}

func (c *Client) Close() error {
	c.logger.Debug("DynamoDB client closed")
	return nil
}

func (c *Client) GetMaxBatchSize() int {
	return c.config.MaxBatchSize
}

func (c *Client) IsConsistentReads() bool {
	return c.config.ConsistentReads
}
