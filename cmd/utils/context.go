package utils

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/streamingfast/eth-go"
)

type privateKeyContextKeyType string

const privateKeyContextKey = privateKeyContextKeyType("privateKey")

func WithPrivateKey(ctx context.Context, privateKey *eth.PrivateKey) context.Context {
	return context.WithValue(ctx, privateKeyContextKey, privateKey)
}

func GetPrivateKey(ctx context.Context) (*eth.PrivateKey, error) {
	privateKey, ok := ctx.Value(privateKeyContextKey).(*eth.PrivateKey)

	if !ok {
		return nil, fmt.Errorf("private key not found in context")
	}

	return privateKey, nil
}

func MustGetPrivateKey(ctx context.Context) *eth.PrivateKey {
	privateKey, err := GetPrivateKey(ctx)

	if err != nil {
		panic(err)
	}

	return privateKey
}

type loggerContextKeyType string

const loggerContextKey = loggerContextKeyType("logger")

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

func GetLogger(ctx context.Context) (*slog.Logger, error) {
	logger, ok := ctx.Value(loggerContextKey).(*slog.Logger)

	if !ok {
		return nil, fmt.Errorf("logger not found in context")
	}

	return logger, nil
}

func MustGetLogger(ctx context.Context) *slog.Logger {
	logger, err := GetLogger(ctx)

	if err != nil {
		panic(err)
	}

	return logger
}
