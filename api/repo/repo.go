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
		log.Print("	tx, err := dbConn.Begin(context.Background())")
		return &RespT{}, err
	}

	defer tx.Rollback(context.Background())

	query := `SELECT limite,saldo FROM clientes WHERE id = $1 FOR UPDATE`

	var limit, balance int

	err = tx.QueryRow(context.Background(), query, client_id).Scan(&limit, &balance)
	if err != nil {
		log.Print("err = tx.QueryRow(context.Background(), query, id).Scan(&limit, &balance)")
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

	batch := &pgx.Batch{}
	batch.Queue(query2, client_id, t.Value, t.Typ, t.Description, t.Created_at)
	batch.Queue("UPDATE clientes SET saldo=$1 WHERE id=$2", newBalance, client_id)
	br := tx.SendBatch(context.Background(), batch)
	_, err = br.Exec()
	if err != nil {

		log.Print("	br := tx.SendBatch(context.Background(), batch)")
		return &RespT{}, err
	}

	err = br.Close()
	if err != nil {
		log.Print("	err = br.Close()	")
		return &RespT{}, err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		log.Print("	err = tx.Commit(context.Background())")
		return &RespT{}, err
	}
	// _, err = tx.Exec(query2, cliente_id, t.Value, t.Typ, t.Description, t.Created_at)
	// if err != nil {
	// 	_ = tx.Rollback()
	// 	return nil, err
	// }

	// _, err = tx.Exec()
	// if err != nil {
	// 	return nil, err
	// }
	// _, err = tx.Exec()
	// if err != nil {
	// 	_ = tx.Rollback()
	// 	return nil, err
	// }

	// if err := tx.Commit(); err != nil {
	// 	return nil, err
	// }
	r := RespT{
		Limit:   limit,
		Balance: newBalance,
	}

	log.Print(r.Limit, r.Balance)

	return &r, nil
}

// func Tb(t *Transaction, clientId int, dbConn *pgxpool.Pool) (*RespT, error) {
// 	tx, err := dbConn.Begin(context.Background())
// 	if err != nil {
// 		return nil, err
// 	}
// 	log.Print("passei 1")

// 	defer tx.Rollback(context.Background())

// 	query := `SELECT  clientes.id, clientes.limite , saldos.valor
//     FROM clientes JOIN saldos ON saldos.cliente_id = clientes.id
//     WHERE clientes.id = $1`

// 	client := new(Client)

// 	err = tx.QueryRow(context.Background(), query, clientId).Scan(&client.Id, &client.Limit, &client.Balance)
// 	if err != nil {
// 		return nil, err
// 	}
// 	log.Print("passei 2")

// 	var newAccountBalance int

// 	if t.Typ == "d" {
// 		newAccountBalance = client.Balance - t.Value
// 	} else {
// 		newAccountBalance = client.Balance + t.Value
// 	}

// 	if (client.Limit + newAccountBalance) < 0 {
// 		return nil, fmt.Errorf("le")
// 	}
// 	log.Print("passei 3")

// 	batch := &pgx.Batch{}
// 	batch.Queue("INSERT INTO transactions(client_id,amount,operation,description) values ($1, $2, $3, $4, $5)", clientId, t.Value, t.Typ, t.Description, t.Created_at)
// 	batch.Queue("UPDATE saldos SET balance = $1 WHERE cliente_id = $2", newAccountBalance, clientId)
// 	br := tx.SendBatch(context.Background(), batch)
// 	_, err = br.Exec()
// 	if err != nil {
// 		return nil, err
// 	}
// 	log.Print("passei 4")

// 	err = br.Close()
// 	if err != nil {
// 		return nil, err
// 	}

// 	log.Print("passei 5")
// 	err = tx.Commit(context.Background())
// 	if err != nil {
// 		return nil, err
// 	}
// 	log.Print("passei 6")

// 	result := RespT{
// 		Limit:   client.Limit,
// 		Balance: newAccountBalance,
// 	}

// 	return &result, nil
// }

func Ex(id int, dbConn *pgxpool.Pool) (*Ext, error) {
	// var limit int
	// var balance int

	query := `SELECT  saldo, now(),limite 
    FROM clientes WHERE id = $1`

	log.Print("comecou ex ")

	rows, _ := dbConn.Query(context.Background(), query, id)
	bE, err := pgx.CollectOneRow(rows, pgx.RowToStructByPos[BalanceE])
	if err != nil {
		return nil, err
	}

	log.Print("passei 1")
	query2 := `SELECT valor, tipo, descricao, realizada_em FROM transacoes WHERE id=$1  ORDER BY id  DESC LIMIT 10`

	rows, _ = dbConn.Query(context.Background(), query2, id)
	lastTransactions, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Transaction])
	if err != nil {
		return nil, err
	}
	log.Print("passei 2")
	e := Ext{
		Balance:         bE,
		LastTransaction: lastTransactions,
	}

	// rows, err := dbConn.Query(context.Background(), query2, id)
	// if err != nil {
	// 	return nil, err
	// }

	// if rows != nil {
	// 	for rows.Next() {

	// 		var trans Transaction

	// 		err := rows.Scan(&trans.Value, &trans.Typ, &trans.Description, &trans.Created_at)
	// 		if err != nil {
	// 			return nil, err
	//
	// 		e.LastTransaction = append(e.LastTransaction, trans)
	// 	}

	// 	rows.Close()
	// }
	return &e, nil
}
