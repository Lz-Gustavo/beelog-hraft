package ycsb

import (
	"context"

	"beelog-hraft/client"

	"github.com/magiconair/properties"
	"github.com/pingcap/go-ycsb/pkg/ycsb"
)

type beelogKV struct {
	client client.Info
}

// Close closes the database layer.
func (bk *beelogKV) Close() error

// InitThread initializes the state associated to the goroutine worker.
// The Returned context will be passed to the following usage.
func (bk *beelogKV) InitThread(ctx context.Context, threadID int, threadCount int) context.Context

// CleanupThread cleans up the state when the worker finished.
func (bk *beelogKV) CleanupThread(ctx context.Context)

// Read reads a record from the database and returns a map of each field/value pair.
func (bk *beelogKV) Read(ctx context.Context, table string, key string, fields []string) (map[string][]byte, error)

// Scan scans records from the database.
func (bk *beelogKV) Scan(ctx context.Context, table string, startKey string, count int, fields []string) ([]map[string][]byte, error)

// Update updates a record in the database. Any field/value pairs will be written into the
// database or overwritten the existing values with the same field name.
func (bk *beelogKV) Update(ctx context.Context, table string, key string, values map[string][]byte) error

// Insert inserts a record in the database. Any field/value pairs will be written into the
// database.
func (bk *beelogKV) Insert(ctx context.Context, table string, key string, values map[string][]byte) error

// Delete deletes a record from the database.
func (bk *beelogKV) Delete(ctx context.Context, table string, key string) error

type beelogKVCreator struct {
}

func (bc beelogKVCreator) Create(p *properties.Properties) (ycsb.DB, error) {
	return &beelogKV{}, nil
}
