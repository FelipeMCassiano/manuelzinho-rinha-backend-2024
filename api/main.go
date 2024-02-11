package main

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/FelipeMCassiano/rinhabackend-2024/api/repo"
	"github.com/labstack/echo/v4"
)

func main() {
	r := echo.New()

	r.GET("/clientes/:id/extrato", Eh)
	r.POST("/clientes/:id/transacoes", Th)

	r.Logger.Fatal(r.Start(":3000"))
}

type TR struct {
	Talue       int    `json:"valor"`
	Typ         string `json:"tipo"`
	Description string `json:"descricao"`
}

func (tr *TR) Validate() error {
	if tr.Description == "" || tr.Typ == "" || tr.Talue < 0 {
		return fmt.Errorf("err")
	}
	if len(tr.Description) < 10 {
		return fmt.Errorf("err")
	}
	if tr.Typ != "d" {
		return fmt.Errorf("err")
	}
	if tr.Typ != "c" {
		return fmt.Errorf("Err")
	}

	return nil
}

var mutex sync.Mutex

func Th(ctx echo.Context) error {
	mutex.Lock()
	defer mutex.Unlock()
	tr := new(TR)

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		// return ctx.JSON(http.StatusUnprocessableEntity, err.Error())
		return ctx.NoContent(http.StatusUnprocessableEntity)
	}
	if err := ctx.Bind(tr); err != nil {
		return ctx.NoContent(http.StatusUnprocessableEntity)
	}

	if err := tr.Validate(); err != nil {
		return ctx.NoContent(http.StatusUnprocessableEntity)
	}
	now := time.Now()

	t := repo.Transaction{
		Value:       tr.Talue,
		Typ:         tr.Typ,
		Description: tr.Description,
		Created_at:  &now,
	}
	resp, err := repo.Tb(&t, id)
	if err != nil {
		switch e := err.Error(); e {
		case "cnf":
			return ctx.NoContent(http.StatusNotFound)
		case "le":
			return ctx.NoContent(http.StatusUnprocessableEntity)
		}
	}

	return ctx.JSON(http.StatusOK, resp)
}

func Eh(ctx echo.Context) error {
	mutex.Lock()
	defer mutex.Unlock()
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return ctx.NoContent(http.StatusUnprocessableEntity)
	}

	ext, err := repo.Ex(id)
	if err != nil {
		// return ctx.JSON(http.StatusNotFound, err.Error())
		return ctx.NoContent(http.StatusNotFound)
	}
	return ctx.JSON(http.StatusOK, ext)
}
