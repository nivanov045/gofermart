package storage

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"runtime"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/nivanov045/gofermart/internal/order"
	"github.com/nivanov045/gofermart/internal/withdraw"
)

type storage struct {
	databasePath string
	db           *sql.DB
	tables       []table
}

/*
Tables:
- orders: order_num|user_login|created_at|status|accrual
- withdraws: user_login|created_at|sum|order_num
- users: user_login|password_hash
- sessions: user_login|session_token|valid_until

Can be added for better user experience:
- user_login|refresh_token|valid_until

Can be added for speedup:
- user_login|orders|withdraws|current_balance|withdraws_balance
*/

type table struct {
	name    string
	columns []column
}

type column struct {
	name       string
	attributes string
}

func (c *column) toString() string {
	return c.name + " " + c.attributes
}

func New(databasePath string) (*storage, error) {
	log.Println("storage::New::info: started")
	var resultStorage = &storage{
		databasePath: databasePath,
		tables: []table{
			{
				name: "orders",
				columns: []column{
					{"order_num", "TEXT UNIQUE"},
					{"user_login", "TEXT"},
					{"created_at", "TIMESTAMP"},
					{"status", "TEXT"},
					{"accrual", "BIGINT"},
				},
			},
			{
				name: "withdraws",
				columns: []column{
					{"user_login", "TEXT"},
					{"order_num", "TEXT"},
					{"created_at", "TIMESTAMP"},
					{"sum", "BIGINT"},
				},
			},
			{
				name: "users",
				columns: []column{
					{"user_login", "TEXT UNIQUE"},
					{"password_hash", "TEXT"},
				},
			},
			{
				name: "sessions",
				columns: []column{
					{"user_login", "TEXT UNIQUE"},
					{"session_token", "TEXT"},
					{"valid_until", "TIMESTAMP"},
				},
			},
		},
	}

	var err error
	resultStorage.db, err = sql.Open("postgres", databasePath)
	if err != nil {
		log.Println("storage::New::error: in db open:", err)
		return nil, errors.New(`can't create database'`)
	}
	runtime.SetFinalizer(resultStorage, func(s *storage) {
		log.Println("storage::New::info: finalizer started")
		defer s.db.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, table := range resultStorage.tables {
		var isTableExists bool
		row := resultStorage.db.QueryRowContext(ctx,
			`SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE  table_name = $1);`, table.name)
		err = row.Scan(&isTableExists)
		if err != nil {
			log.Println("storage::New::error: in table check:", err)
			return nil, errors.New(`can't create database'`)
		}
		if !isTableExists {
			_, err = resultStorage.db.Exec(constructMakeTableQuery(table))
			if err != nil {
				log.Println("storage::New::error: in table creation:", err)
				return nil, errors.New(`can't create database'`)
			}
		} else {
			tableIsOk := true
			for _, c := range table.columns {
				var isColumnInTable bool
				row := resultStorage.db.QueryRowContext(ctx,
					`SELECT EXISTS (
    				SELECT column_name FROM information_schema.columns
    				WHERE table_name=$1 and column_name=$2);`, table.name, c.name)
				err = row.Scan(&isColumnInTable)
				if err != nil {
					log.Println("storage::New::error: in columns check:", err)
					return nil, errors.New(`can't create database'`)
				}
				if !isColumnInTable {
					tableIsOk = false
					break
				}
			}

			if !tableIsOk {
				log.Println("storage::New::info: table is wrong, clone, drop and create new")
				_, err = resultStorage.db.Exec(`CREATE TABLE dupe_$1_$2 AS (SELECT * FROM $1);`, table.name,
					time.Now().String())
				if err != nil {
					log.Println("storage::New::error: in table duplication:", err)
					return nil, errors.New(`can't create database'`)
				}
				_, err = resultStorage.db.Exec(`DROP TABLE $1;`, table.name)
				if err != nil {
					log.Println("storage::New::error: in table drop:", err)
					return nil, errors.New(`can't create database'`)
				}
				_, err = resultStorage.db.Exec(constructMakeTableQuery(table))
				if err != nil {
					log.Println("storage::New::error: in table creation:", err)
					return nil, errors.New(`can't create database'`)
				}
			} else {
				log.Println("storage::New::info: existing table", table.name, "is OK")
			}
		}
	}
	return resultStorage, nil
}

func constructMakeTableQuery(t table) string {
	var query strings.Builder
	query.WriteString(`CREATE TABLE ` + t.name)
	if len(t.columns) == 0 {
		query.WriteString(`;`)
		return query.String()
	}
	query.WriteString(` (` + t.columns[0].toString())
	for i := 1; i < len(t.columns); i++ {
		query.WriteString(`, ` + t.columns[i].toString())
	}
	query.WriteString(`);`)
	return query.String()
}

func (s *storage) FindOrderByUser(login string, number string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var isExists bool
	row := s.db.QueryRowContext(ctx,
		`SELECT EXISTS (
    	SELECT FROM orders WHERE order_num=$1 AND user_login=$2);`, number, login)
	err := row.Scan(&isExists)
	if err != nil {
		log.Println("storage::FindOrderByUser::info: in QueryRowContext:", err)
		return false, err
	}
	return isExists, nil
}

func (s *storage) FindOrder(number string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var isExists bool
	row := s.db.QueryRowContext(ctx,
		`SELECT EXISTS (
    	SELECT FROM orders WHERE order_num=$1);`, number)
	err := row.Scan(&isExists)
	if err != nil {
		log.Println("storage::FindOrder::info: in QueryRowContext:", err)
		return false, err
	}
	return isExists, nil
}

func (s *storage) AddOrder(login string, number string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	log.Println("storage::AddOrder::info:", login, number)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO orders(order_num, user_login, created_at, status)
		VALUES ($1, $2, $3, $4);`, number, login, time.Now(), order.ProcessingTypeNew)
	return err
}

func (s *storage) GetOrders(login string) ([]order.Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var resultOrders []order.Order

	rows, err := s.db.QueryContext(ctx,
		`SELECT order_num, created_at, status, accrual FROM orders WHERE user_login=$1;`, login)
	if err != nil {
		log.Println("storage::GetOrders::info: in QueryContext:", err)
		return resultOrders, err
	}
	if rows.Err() != nil {
		log.Println("storage::GetOrders::error: in rows:", err)
		return resultOrders, err
	}
	for rows.Next() {
		var orderNum, status string
		var creationTime time.Time
		var accrual sql.NullInt64
		err := rows.Scan(&orderNum, &creationTime, &status, &accrual)
		if err != nil {
			log.Println("storage::GetOrders::info: in Scan:", err)
			continue
		}
		log.Println("storage::GetOrders::info:", orderNum, status, creationTime, accrual)
		val := order.Order{
			Number:     orderNum,
			Status:     status,
			UploadedAt: creationTime,
		}
		if status == order.ProcessingTypeProcessed {
			val.Accrual = accrual.Int64
		}
		resultOrders = append(resultOrders, val)
	}
	return resultOrders, nil
}

func (s *storage) UpdateOrder(orderData order.Order) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.db.ExecContext(ctx,
		`UPDATE orders SET status = $1, accrual = $2 WHERE order_num = $3;`, orderData.Status, orderData.Accrual,
		orderData.Number)
	return err
}

func (s *storage) MakeWithdraw(login string, order string, sum int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO withdraws(user_login, created_at, sum, order_num)
		VALUES ($1, $2, $3, $4);`, login, time.Now(), sum, order)
	if err != nil {
		log.Println("storage::MakeWithdraw::error: in ExecContext:", err)
		return err
	}
	return nil
}

func (s *storage) GetWithdraws(login string) ([]withdraw.Withdraw, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var resultWithdraws []withdraw.Withdraw

	rows, err := s.db.QueryContext(ctx,
		`SELECT created_at, sum, order_num FROM withdraws WHERE user_login=$1;`, login)
	if err != nil {
		log.Println("storage::GetWithdraws::info: in QueryContext:", err)
		return resultWithdraws, err
	}
	if rows.Err() != nil {
		log.Println("storage::GetWithdraws::error: in rows:", err)
		return resultWithdraws, err
	}
	for rows.Next() {
		var orderNum string
		var creationTime time.Time
		var sum int64
		err := rows.Scan(&creationTime, &sum, &orderNum)
		if err != nil {
			log.Println("storage::GetWithdraws::info: in Scan:", err)
			continue
		}
		resultWithdraws = append(resultWithdraws, withdraw.Withdraw{
			Order:       orderNum,
			Sum:         sum,
			ProcessedAt: creationTime,
		})
	}
	return resultWithdraws, nil
}

func (s *storage) AddUser(login string, passwordHash string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users(user_login, password_hash)
		VALUES ($1, $2);`, login, passwordHash)
	if err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint \"users_user_login_key\"" {
			return errors.New("login is already in use")
		}
		return err
	}
	return nil
}

func (s *storage) AddSession(login string, sessionToken string, expiresAt time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions(user_login, session_token, valid_until) VALUES ($1, $2, $3)
		ON CONFLICT (user_login) DO UPDATE SET session_token=$2, valid_until=$3;`, login, sessionToken, expiresAt)
	return err
}

func (s *storage) GetSessionInfo(sessionToken string) (string, time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var login string
	var expTime time.Time
	row := s.db.QueryRowContext(ctx,
		`SELECT user_login, valid_until FROM sessions WHERE session_token=$1;`, sessionToken)
	err := row.Scan(&login, &expTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", time.Time{}, errors.New("no such token")
		}
		return "", time.Time{}, err
	}
	return login, expTime, nil
}

func (s *storage) CheckPassword(login string, passwordHash string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var isPasswordHashCorrect bool
	row := s.db.QueryRowContext(ctx,
		`SELECT EXISTS (
    	SELECT FROM users WHERE user_login=$1 AND password_hash=$2);`, login, passwordHash)
	err := row.Scan(&isPasswordHashCorrect)
	if err != nil {
		return false, err
	}
	return isPasswordHashCorrect, nil
}

func (s *storage) RemoveSession(sessionToken string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE session_token = $1;`, sessionToken)
	return err
}
