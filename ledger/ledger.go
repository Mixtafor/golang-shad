//go:build !solution

package ledger

import (
	"context"
	"errors"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4/pgxpool"
)

const checkErr = "23514"

type ledger struct {
	conn *pgxpool.Pool
}

func (l *ledger) CreateAccount(ctx context.Context, id ID) error {
	_, err := l.conn.Exec(ctx, `
		insert into ledger(id)
		values ($1)
`, id)
	return err
}

func (l *ledger) GetBalance(ctx context.Context, id ID) (Money, error) {
	mn := Money(0)

	err := l.conn.QueryRow(ctx, `
		select money
		from ledger
		where id = $1
`, id).Scan(&mn)

	return mn, err

}

func (l *ledger) Deposit(ctx context.Context, id ID, amount Money) error {
	if amount < 0 {
		return errors.New("amount must be >= 0")
	}

	_, err := l.conn.Exec(ctx, `
		update ledger
		set money = money + $2
		where id = $1
	`, id, amount)

	return err
}

func (l *ledger) Withdraw(ctx context.Context, id ID, amount Money) error {
	if amount < 0 {
		return errors.New("amount must be >= 0")
	}

	_, err := l.conn.Exec(ctx, `
		update ledger
		set money = money - $2
		where id = $1
	`, id, amount)

	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) {
		if pgerr.Code == checkErr {
			return ErrNoMoney
		}
	}
	return err
}

func (l *ledger) Transfer(ctx context.Context, from, to ID, amount Money) error {
	if amount < 0 {
		return errors.New("amount must be >= 0")
	}

	trx, err := l.conn.Begin(ctx)
	defer trx.Rollback(ctx)
	if err != nil {
		return err
	}

	_, err = trx.Exec(ctx, `
		select *
		from ledger
		where id = $1 or id = $2
		order by id
		for no key update
	`, from, to)

	_, err = trx.Exec(ctx, `
		update ledger
		set money = money - $2
		where id = $1
	`, from, amount)

	if err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) {
			if pgerr.Code == checkErr {
				return ErrNoMoney
			}
		}
		return err
	}

	_, err = trx.Exec(ctx, `
		update ledger
		set money = money + $2
		where id = $1
	`, to, amount)

	if err != nil {
		return err
	}

	return trx.Commit(ctx)
}

func (l *ledger) Close() error {
	l.conn.Close()
	return nil
}

func New(ctx context.Context, dsn string) (Ledger, error) {
	conn, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	_, err = conn.Exec(ctx, `
		CREATE TABLE if not exists ledger (
		    id text PRIMARY KEY,
			money bigint not null default 0,
			constraint money_not_neg check(money >= 0)
		)	
`)
	if err != nil {
		return nil, err
	}

	return &ledger{conn: conn}, nil
}
