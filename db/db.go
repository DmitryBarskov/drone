package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
)

var (
	ErrKeyNotFound = errors.New("db: not found")
)

type KV struct {
	Key []byte `json:"key"`
	Val []byte `json:"val"`
}

type DB interface {
	Start(context.Context) error
	Stop()
	Do(context.Context, func(Tx) error) error
	Dump(context.Context) ([]KV, error)
}

type Tx interface {
	Get(key []byte) ([]byte, error)
	Set(key []byte, val []byte) error
}

func DumpJson(ctx context.Context, db DB, writer io.Writer) error {
	dump, err := db.Dump(ctx)
	if err != nil {
		return fmt.Errorf("dump db: %w", err)
	}

	out := make(map[string]json.RawMessage, len(dump))
	for _, kv := range dump {
		out[string(kv.Key)] = kv.Val
	}

	data, err := json.Marshal(out)
	if err != nil {
		return fmt.Errorf("marshall dump: %w", err)
	}

	_, err = writer.Write(data)
	if err != nil {
		return fmt.Errorf("write dump: %w", err)
	}

	return nil
}

func RestoreJson(ctx context.Context, db DB, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("read dump: %w", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshall dump: %w", err)
	}

	dump := make([]KV, 0, len(raw))
	for k, v := range raw {
		dump = append(dump, KV{Key: []byte(k), Val: v})
	}
	return db.Do(ctx, func(tx Tx) error {
		for _, kv := range dump {
			if err := tx.Set(kv.Key, kv.Val); err != nil {
				return err
			}
			slog.Info("set key", slog.String("key", string(kv.Key)))
		}

		return nil
	})
}

func GetJson[T any](tx Tx, key string) (T, error) {
	data, err := tx.Get([]byte(key))
	if err != nil {
		return *new(T), fmt.Errorf("get key: %w", err)
	}

	result := *new(T)
	err = json.Unmarshal(data, &result)
	if err != nil {
		return *new(T), fmt.Errorf("unmarshall key val: %w", err)
	}

	return result, nil
}

func GetJsonDefault[T any](tx Tx, key string, defaultVal T) (T, error) {
	result, err := GetJson[T](tx, key)
	if errors.Is(err, ErrKeyNotFound) {
		return defaultVal, nil
	}

	return result, err
}

func SetJson[T any](tx Tx, key string, val T) error {
	data, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("marshall key val: %w", err)
	}

	err = tx.Set([]byte(key), data)
	if err != nil {
		return fmt.Errorf("set key: %w", err)
	}

	return nil
}
