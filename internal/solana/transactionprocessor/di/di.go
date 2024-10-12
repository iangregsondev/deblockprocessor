package di

import (
	"fmt"

	"github.com/ggwhite/go-masker/v2"
	"github.com/iangregsondev/deblockprocessor/internal/adapters/kafka"
	"github.com/iangregsondev/deblockprocessor/internal/adapters/rpc"
	"github.com/iangregsondev/deblockprocessor/internal/solana/transactionprocessor/app"
	"github.com/iangregsondev/deblockprocessor/internal/solana/transactionprocessor/config"
	"github.com/iangregsondev/deblockprocessor/internal/solana/transactionprocessor/services/transaction"
	iowrapper "github.com/iangregsondev/deblockprocessor/internal/wrappers/io"
	"github.com/iangregsondev/deblockprocessor/internal/wrappers/logger"
	oswrapper "github.com/iangregsondev/deblockprocessor/internal/wrappers/os"
	"github.com/iangregsondev/deblockprocessor/pkg/blockchainproviders/solana/blockdaemon"
)

func NewDI(logger logger.Logger, mask *masker.MaskerMarshaler, ioWrapper iowrapper.IO, osWrapper oswrapper.OS, cfg *config.Config) (*app.App, error) {
	logger.Info("Loading and validating configuration...")

	loadedConfig, err := cfg.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %w", err)
	}

	// mask the configuration because it contains sensitive information
	maskedLoadedConfig, err := mask.Struct(loadedConfig)
	if err != nil {
		return nil, fmt.Errorf("error masking configuration: %w", err)
	}

	logger.Info("Configuration loaded and validated successfully", "config", maskedLoadedConfig)

	level, err := logger.ParseLogLevel(loadedConfig.Logger.Level)
	if err != nil {
		return nil, fmt.Errorf("error parsing log level: %w", err)
	}

	logger.SetLogLevel(level)

	// instantiate adapters
	rpcClient := rpc.NewRPC(
		logger, ioWrapper, loadedConfig.Connection.RPCURL, loadedConfig.Connection.APIKey, loadedConfig.HTTP.MaxRetryOnError,
		loadedConfig.HTTP.RetryDelayMilliseconds,
	)
	kafkaAdapter := kafka.NewClient(logger, loadedConfig.Kafka.Broker)

	// provider
	bcProvider := blockdaemon.NewProvider(rpcClient)

	// instantiate services
	transactionService := transaction.NewService(
		logger, kafkaAdapter, bcProvider,
		loadedConfig.Connection.RPCURL, loadedConfig.Connection.APIKey, loadedConfig.Kafka.BlockTopic, loadedConfig.Kafka.BlockConsumerGroup,
		loadedConfig.Kafka.TransactionTopic,
	)

	// return the application
	return app.NewApp(logger, mask, osWrapper, loadedConfig, transactionService), nil
}
