package storages

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgconn"

	"github.com/nivanov045/gofermart/internal/accrual/models"
)

const (
	productsTableName    = "models"
	ordersTableName      = "orders"
	ordersQueueTableName = "orders_queue"
)

type dbStorage struct {
	db *sql.DB
}

func NewDBStorage(ctx context.Context, db *sql.DB) (Storage, OrderQueue, error) {
	if db == nil {
		return nil, nil, errors.New("storage creation error: db is nil")
	}

	s := dbStorage{db: db}

	err := s.createProductsTable(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("storage error: %v", err)
	}
	err = s.createOrdersTable(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("storage error: %v", err)
	}
	err = s.createOrdersQueueTable(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("storage error: %v", err)
	}

	return &s, &s, nil
}

func (s *dbStorage) GetOrderStatus(ctx context.Context, id string) (models.OrderStatus, error) {
	row := s.db.QueryRowContext(ctx, `SELECT accrual, status FROM `+ordersTableName+` WHERE id = $1`, id)
	err := row.Err()
	if err != nil {
		return models.OrderStatus{}, err
	}

	var accrual int
	var status models.OrderStatusCode
	err = row.Scan(&accrual, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return models.OrderStatus{Accrual: 0, Status: models.OrderStatusInvalid}, ErrOrderNotFound
	}
	if err != nil {
		return models.OrderStatus{}, err
	}

	return models.OrderStatus{Accrual: accrual, Status: status}, nil
}

func (s *dbStorage) UpdateOrderStatus(ctx context.Context, id string, orderStatus models.OrderStatus) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO `+ordersTableName+` (id, accrual, status) VALUES ($1, $2, $3)
										   ON CONFLICT (id)
										   DO UPDATE SET (accrual, status) = ($2, $3);`, id, orderStatus.Accrual, orderStatus.Status)
	if err != nil {
		return err
	}

	return nil
}

func (s *dbStorage) GetProduct(ctx context.Context, name string) (*models.Product, error) {
	row := s.db.QueryRowContext(ctx, `SELECT reward, reward_type  FROM `+productsTableName+` WHERE name = $1`, name)
	err := row.Err()
	if err != nil {
		return nil, err
	}

	var reward int
	var rewardType models.RewardType
	err = row.Scan(&reward, &rewardType)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductNotFound
	}
	if err != nil {
		return nil, err
	}

	return &models.Product{Match: name, Reward: reward, RewardType: rewardType}, nil
}

func (s *dbStorage) RegisterProduct(ctx context.Context, name string, reward int, rewardType models.RewardType) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO `+productsTableName+` (name, reward, reward_type) VALUES ($1, $2, $3);`, name, reward, rewardType)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrProductAlreadyRegistered
		}
		return err
	}

	return nil
}

func (s *dbStorage) GetOrder(ctx context.Context) ([]byte, error) {
	row := s.db.QueryRowContext(ctx, `SELECT info FROM `+ordersTableName+` LIMIT 1;`)
	err := row.Err()
	if err != nil {
		return nil, err
	}

	var orderInfo []byte
	err = row.Scan(&orderInfo)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}

	return orderInfo, nil
}

func (s *dbStorage) RemoveOrder(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM `+ordersQueueTableName+` WHERE id=$1`, id)
	if err != nil {
		return err
	}

	return nil
}

func (s *dbStorage) RegisterOrder(ctx context.Context, id string, orderInfo []byte) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO `+ordersQueueTableName+` (id, info) VALUES ($1, $2);`, id, orderInfo)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `INSERT INTO `+ordersTableName+` (id, accrual, status) VALUES ($1, $2, $3);`, id, 0, models.OrderStatusRegistered)
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
			accrual bigint,
			status int
	);`)

	if err != nil {
		return err
	}
	return nil
}

func (s *dbStorage) createOrdersQueueTable(ctx context.Context) error {
	isExists, err := s.isTableExists(ctx, ordersQueueTableName)
	if err != nil {
		return err
	}
	if isExists {
		return nil
	}

	_, err = s.db.ExecContext(ctx, `
		CREATE TABLE `+ordersQueueTableName+` (
			id varchar(255) NOT NULL UNIQUE,
			info bytea
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
