package storages

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"gofermart/internal/accrual/products"
)

const tableName = "products"

type dbStorage struct {
	db *sql.DB
}

func NewDBStorage(ctx context.Context, db *sql.DB) (Storage, error) {
	if db == nil {
		return nil, errors.New("storage creation error: db is nil")
	}

	s := dbStorage{db: db}

	err := s.createTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage creation error: %v", err)
	}

	return &s, nil
}

func (s *dbStorage) RegisterProduct(ctx context.Context, name string, reward int, rewardType products.RewardType) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO `+tableName+` (name, reward, reward_type) VALUES ($1, $2, $3);`, name, reward, rewardType)
	if err != nil {
		return err
	}

	return nil
}

func (s *dbStorage) createTable(ctx context.Context) error {
	isExists, err := s.isTableExists(ctx)
	if err != nil {
		return err
	}

	if isExists {
		return nil
	}

	_, err = s.db.ExecContext(ctx, `
		CREATE TABLE `+tableName+` (
			name varchar(255) NOT NULL UNIQUE,
			reward bigint,
			reward_type int
	);`)

	if err != nil {
		return err
	}

	return nil
}

func (s *dbStorage) isTableExists(ctx context.Context) (bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM
		information_schema.tables
		WHERE
		table_schema LIKE 'public' AND
		table_type LIKE 'BASE TABLE' AND
		table_name = '`+tableName+`'
	);`)

	err := row.Err()
	if err != nil {
		return false, err
	}
	var exists bool
	err = row.Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
