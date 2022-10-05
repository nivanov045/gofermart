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
	productsTableName    = "products"
	ordersTableName      = "orders"
	ordersQueueTableName = "orders_queue"
)

const (
	ErrCodeDuplicateKeyViolatesUniqueConstraint = "23505"
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

	var accrual float64
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

func (s *dbStorage) MatchProducts(ctx context.Context, description string) ([]models.Product, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT matchText, reward, reward_type FROM `+productsTableName+` WHERE $1 LIKE concat('%',matchText,'%');`,
		description)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]models.Product, 0)
	for rows.Next() {
		var matchText string
		var reward float64
		var rewardType models.RewardType
		err = rows.Scan(&matchText, &reward, &rewardType)
		products = append(products, models.Product{Match: matchText, Reward: reward, RewardType: rewardType})

		if err != nil {
			return nil, err
		}
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return products, nil
}

func (s *dbStorage) RegisterProduct(ctx context.Context, product models.Product) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO `+productsTableName+` (matchText, reward, reward_type) VALUES ($1, $2, $3);`,
		product.Match, product.Reward, product.RewardType)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == ErrCodeDuplicateKeyViolatesUniqueConstraint {
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
			matchText varchar(255) NOT NULL UNIQUE,
			reward float8,
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
			accrual float8,
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
