package repo

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

type Transaction struct {
	Value       int        `json:"valor"`
	Typ         string     `json:"tipo"`
	Description string     `json:"descricao"`
	Created_at  *time.Time `json:"realizada_em"`
}

type respT struct {
	Limit   int `json:"limite"`
	Balance int `json:"saldo"`
}
type Ext struct {
	Balance         BalanceE      `json:"saldo"`
	LastTransaction []Transaction `json:"ultimas_transacoes"`
}

type BalanceE struct {
	Total      int        `json:"total"`
	Created_at *time.Time `json:"data_extrato"`
	Limit      int        `json:"limite"`
}

var mutex sync.Mutex

func OpenConn() *sql.DB {
	du := os.Getenv("DATABASE_URL")
	conn, err := sql.Open("postgres", du)
	if err != nil {
		log.Fatal(err)
	}

	return conn
}

func Tb(t *Transaction, cliente_id int) (*respT, error) {
	dbConn := OpenConn()
	defer dbConn.Close()

	mutex.Lock()
	defer mutex.Unlock()

	tx, err := dbConn.Begin()
	if err != nil {
		return &respT{}, err
	}

	query := `SELECT clientes.limite AS client_limit, saldos.valor AS balance 
    FROM clientes JOIN saldos ON saldos.cliente_id = clientes.id 
    WHERE clientes.id = $1
    `
	var accountBalance int
	var accountLimit int

	err = tx.QueryRow(query, cliente_id).Scan(&accountLimit, &accountBalance)
	if err != nil {
		return &respT{}, fmt.Errorf("cnf")
	}

	var newBalance int
	var newLimit int

	switch typ := t.Typ; typ {
	case "d":
		newBalance = accountBalance - t.Value
	case "c":
		newLimit = accountLimit + t.Value
	}

	if (accountBalance + newLimit) < 0 {
		return nil, fmt.Errorf("le")
	}

	query2 := `INSERT INTO transacoes (cliente_id, valor, tipo, descricao, realizada_em) VALUES($1, $2, $3, $4, $5)`

	_, err = tx.Exec(query2, cliente_id, t.Value, t.Typ, t.Description, t.Created_at)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	_, err = tx.Exec("UPDATE clientes SET limite=$1 WHERE id=$2", newLimit, cliente_id)
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec("Update saldos SET valor=$1 WHERE cliente_id=$2", newBalance, cliente_id)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	r := respT{
		Limit:   newLimit,
		Balance: newBalance,
	}

	return &r, nil
}

func Ex(id int) (*Ext, error) {
	dbConn := OpenConn()
	defer dbConn.Close()

	mutex.Lock()
	defer mutex.Unlock()

	var limit int
	var balance int

	query := `SELECT clientes.limite, saldos.valor
    FROM clientes JOIN saldos ON saldos.cliente_id = clientes.id
    WHERE clientes.id = $1
    `

	err := dbConn.QueryRow(query, id).Scan(&limit, &balance)
	if err != nil {
		return nil, err
	}

	query2 := `
    SELECT valor, tipo, descricao, realizada_em FROM transacoes WHERE cliente_id=$1 
    ORDER BY realizada_em DESC LIMIT 10
    `

	rows, err := dbConn.Query(query2, id)
	if err != nil {
		return nil, err
	}
	e := new(Ext)

	if rows != nil {
		for rows.Next() {
			var trans Transaction
			err := rows.Scan(&trans.Value, &trans.Typ, &trans.Description, &trans.Created_at)
			if err != nil {
				return nil, err
			}
			e.LastTransaction = append(e.LastTransaction, trans)
		}
	}
	now := time.Now()

	e.Balance.Limit = limit
	e.Balance.Total = balance
	e.Balance.Created_at = &now

	return e, nil
}
