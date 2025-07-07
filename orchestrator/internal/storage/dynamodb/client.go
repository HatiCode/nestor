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

	awsConfig, err := loadAwsConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := dynamodb.NewFromConfig(awsConfig)

	return &Client{
		client:    client,
		config:    cfg,
		logger:    logger,
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
	_, err := c.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(c.tableName)})
	if err != nil {
		if _, ok := err.(*types.ResourceNotFoundException); ok {
			c.logger.Debug("ping successful, table does not exist yet", "table", c.tableName)
			return nil
		}
		return fmt.Errorf("failed to ping DynamoDB: %w", err)
	}

	c.logger.Debug("ping successful", "table", c.tableName)
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
	c.logger.Info("waiting for table to be active", "table", c.tableName)
	waiter := dynamodb.NewTableExistsWaiter(c.client)

	err := waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(c.tableName),
	}, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("table did not become active: %w", err)
	}

	c.logger.Info("table is now active", "table", c.tableName)
	return nil
}

func (c *Client) GetItem(ctx context.Context, input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	if input.TableName == nil {
		input.TableName = aws.String(c.tableName)
	}

	if input.ConsistentRead == nil {
		input.ConsistentRead = aws.Bool(c.config.ConsistentReads)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.QueryTimeout)
	defer cancel()

	return c.client.GetItem(timeoutCtx, input)
}

func (c *Client) PutItem(ctx context.Context, input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	if input.TableName == nil {
		input.TableName = aws.String(c.tableName)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.QueryTimeout)
	defer cancel()

	return c.client.PutItem(timeoutCtx, input)
}
