package storages

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"gofermart/internal/accrual/products"
)

const (
	productsTableName = "products"
	ordersTableName   = "orders"
)

type dbStorage struct {
	db *sql.DB
}

func NewDBStorage(ctx context.Context, db *sql.DB) (Storage, error) {
	if db == nil {
		return nil, errors.New("storage creation error: db is nil")
	}

	s := dbStorage{db: db}

	err := s.createProductsTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage error: %v", err)
	}
	err = s.createOrdersTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage error: %v", err)
	}

	return &s, nil
}

func (s *dbStorage) GetOrder(ctx context.Context, id string) (int, error) {
	row := s.db.QueryRowContext(ctx, `SELECT accrual FROM `+ordersTableName+` WHERE id = $1`, id)
	err := row.Err()
	if err != nil {
		return 0, err
	}

	var accrual int
	err = row.Scan(&accrual)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrOrderNotFound
	}
	if err != nil {
		return 0, err
	}

	return accrual, nil
}

func (s *dbStorage) StoreOrder(ctx context.Context, id string, accrual int) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO `+ordersTableName+` (id, accrual) VALUES ($1, $2);`, id, accrual)
	if err != nil {
		return err
	}

	return nil
}

func (s *dbStorage) GetProduct(ctx context.Context, name string) (*products.Product, error) {
	row := s.db.QueryRowContext(ctx, `SELECT reward, reward_type  FROM `+productsTableName+` WHERE name = $1`, name)
	err := row.Err()
	if err != nil {
		return nil, err
	}

	var reward int
	var rewardType products.RewardType
	err = row.Scan(&reward, &rewardType)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductNotFound
	}
	if err != nil {
		return nil, err
	}

	return &products.Product{Match: name, Reward: reward, RewardType: rewardType}, nil
}

func (s *dbStorage) RegisterProduct(ctx context.Context, name string, reward int, rewardType products.RewardType) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO `+productsTableName+` (name, reward, reward_type) VALUES ($1, $2, $3);`, name, reward, rewardType)
	if err != nil {
		return err
	}

	return nil
}

func (s *dbStorage) createProductsTable(ctx context.Context) error {
	isExists, err := s.isTableExists(ctx, productsTableName)
	if err != nil {
		return err
	}
	if isExists {
		return nil
	}

	_, err = s.db.ExecContext(ctx, `
		CREATE TABLE `+productsTableName+` (
			name varchar(255) NOT NULL UNIQUE,
			reward bigint,
			reward_type int
	);`)

	if err != nil {
		return err
	}
	return nil
}

func (s *dbStorage) createOrdersTable(ctx context.Context) error {
	isExists, err := s.isTableExists(ctx, ordersTableName)
	if err != nil {
		return err
	}
	if isExists {
		return nil
	}

	_, err = s.db.ExecContext(ctx, `
		CREATE TABLE `+ordersTableName+` (
			id varchar(255) NOT NULL UNIQUE,
			accrual bigint
	);`)

	if err != nil {
		return err
	}
	return nil
}

func (s *dbStorage) isTableExists(ctx context.Context, tableName string) (bool, error) {
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
