package repo

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
)

type Transaction struct {
	Value       int        `json:"valor"`
	Typ         string     `json:"tipo"`
	Description string     `json:"descricao"`
	Created_at  *time.Time `json:"realizada_em"`
}

type RespT struct {
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
type Client struct {
	Id      int
	Limit   int
	Balance int
}

var cache = make(map[int]*Client)

func Gc(id int, dbConn *pgxpool.Pool) (*Client, error) {
	cC, ok := cache[id]
	if ok {
		return cC, nil
	}

	query := `SELECT id,limite,saldo FROM clientes WHERE id = $1`
	rows, _ := dbConn.Query(context.Background(), query, id)
	client, err := pgx.CollectOneRow(rows, pgx.RowToStructByPos[Client])

	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		cache[client.Id] = nil
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	cache[client.Id] = &client

	return &client, err
}

func Tb(t *Transaction, client_id int, dbConn *pgxpool.Pool) (*RespT, error) {
	tx, err := dbConn.Begin(context.Background())
	if err != nil {
		return &RespT{}, err
	}

	defer tx.Rollback(context.Background())

	query := `SELECT limite,saldo FROM clientes WHERE id = $1 FOR UPDATE`

	var limit, balance int

	err = tx.QueryRow(context.Background(), query, client_id).Scan(&limit, &balance)
	if err != nil {
		return nil, err
	}

	var newBalance int

	if t.Typ == "d" {
		newBalance = balance - t.Value
	} else {
		newBalance = balance + t.Value
	}

	if (limit + newBalance) < 0 {
		return nil, fmt.Errorf("le")
	}

	query2 := `INSERT INTO transacoes (cliente_id, valor, tipo, descricao, realizada_em) VALUES($1, $2, $3, $4,$5)`

	log.Print(client_id)
	batch := &pgx.Batch{}
	batch.Queue(query2, client_id, t.Value, t.Typ, t.Description, t.Created_at)
	batch.Queue("UPDATE clientes SET saldo=$1 WHERE id=$2", newBalance, client_id)
	br := tx.SendBatch(context.Background(), batch)
	_, err = br.Exec()
	if err != nil {
		return &RespT{}, err
	}

	err = br.Close()
	if err != nil {
		return &RespT{}, err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return &RespT{}, err
	}
	r := RespT{
		Limit:   limit,
		Balance: newBalance,
	}

	log.Print(r.Limit, r.Balance)

	return &r, nil
}

func Ex(id int, dbConn *pgxpool.Pool) (*Ext, error) {
	query := `SELECT  saldo, now(),limite FROM clientes WHERE id = $1`

	rows, _ := dbConn.Query(context.Background(), query, id)
	bE, err := pgx.CollectOneRow(rows, pgx.RowToStructByPos[BalanceE])
	if err != nil {
		return nil, err
	}

	query2 := `SELECT valor, tipo, descricao, realizada_em FROM transacoes WHERE cliente_id=$1 ORDER BY realizada_em DESC LIMIT 10`

	log.Print(id)

	rows, _ = dbConn.Query(context.Background(), query2, id)
	lastTransactions, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Transaction])
	log.Print(lastTransactions)
	if err != nil {
		return nil, err
	}
	e := Ext{
		Balance:         bE,
		LastTransaction: []Transaction{},
	}
	e.LastTransaction = append(e.LastTransaction, lastTransactions...)
	return &e, nil
}
