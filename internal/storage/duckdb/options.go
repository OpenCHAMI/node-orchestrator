package duckdb

import (
	"os"
	"time"
)

type DuckDBStorageOption interface {
	apply(*DuckDBStorage) error
}

type snapshotFrequencyOption time.Duration

func (s snapshotFrequencyOption) apply(d *DuckDBStorage) error {
	d.snapshotFrequency = time.Duration(s)
	return nil
}

func WithSnapshotFrequency(frequency time.Duration) DuckDBStorageOption {
	return snapshotFrequencyOption(frequency)
}

type snapshotPathOption string

func (s snapshotPathOption) apply(d *DuckDBStorage) error {
	d.snapshotPath = string(s)
	return nil
}

func WithSnapshotPath(path string) DuckDBStorageOption {
	return snapshotPathOption(path)
}

type restoreOption string

func (r restoreOption) apply(d *DuckDBStorage) error {
	d.restoreFirst = true
	d.snapshotPath = string(r)
	return d.restore(d.snapshotPath)
}

func WithRestore(path string) DuckDBStorageOption {
	return restoreOption(path)
}

type createSnapshotDirOption bool

func (c createSnapshotDirOption) apply(d *DuckDBStorage) error {
	if bool(c) {
		return os.MkdirAll(d.snapshotPath, 0755)
	}
	return nil
}

func WithCreateSnapshotDir(create bool) DuckDBStorageOption {
	return createSnapshotDirOption(create)
}

type initTablesOption bool

func (i initTablesOption) apply(d *DuckDBStorage) error {
	if bool(i) {
		return d.initializeDatabase()
	}
	return nil
}

func WithInitTables(init bool) DuckDBStorageOption {
	return initTablesOption(init)
}
