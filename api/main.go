package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/FelipeMCassiano/rinhabackend-2024/api/repo"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

var dbConn *pgxpool.Pool

type CustomV struct {
	validator *validator.Validate
}

func (cv *CustomV) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity)
	}
	return nil
}

func main() {
	r := echo.New()
	r.Validator = &CustomV{validator: validator.New()}

	r.GET("/clientes/:id/extrato", Eh)
	r.POST("/clientes/:id/transacoes", Th)

	conn, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	dbConn = conn

	r.Logger.Fatal(r.Start(":3000"))
}

type TR struct {
	Talue       int    `json:"valor" validate:"required,gt=0"`
	Description string `json:"descricao" validate:"required,min=1,max=10"`
	Typ         string `json:"tipo" validate:"required,oneof=c d"`
}

// func (tr *TR) Validate() error {
// 	if tr.Description == "" || tr.Typ == "" || tr.Talue <= 0 {
// 		return fmt.Errorf("camps empty")
// 	}
// 	if len(tr.Description) > 10 {
// 		return fmt.Errorf("desc too big or small")
// 	}
// 	// if tr.Typ != "c" && tr.Typ != "d" {
// 	// 	return fmt.Errorf("not d or c")
// 	// }
// 	if tr.Typ != "c" {
// 		return fmt.Errorf("err")
// 	}

// 	return nil
// }

func Th(ctx echo.Context) error {
	tr := new(TR)

	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return ctx.JSON(http.StatusUnprocessableEntity, err.Error())
		// return ctx.NoContent(http.StatusUnprocessableEntity)
	}
	if err := ctx.Bind(&tr); err != nil {
		return ctx.JSON(http.StatusUnprocessableEntity, err.Error())
		// return ctx.NoContent(http.StatusUnprocessableEntity)
	}

	if err := ctx.Validate(tr); err != nil {
		return ctx.JSON(http.StatusUnprocessableEntity, err.Error())
		// return ctx.NoContent(http.StatusUnprocessableEntity)
	}

	client, err := repo.Gc(id, dbConn)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, err.Error())
	}

	if client == nil {
		return ctx.NoContent(http.StatusNotFound)
	}
	now := time.Now()

	t := repo.Transaction{
		Value:       tr.Talue,
		Typ:         tr.Typ,
		Description: tr.Description,
		Created_at:  &now,
	}

	resp, err := repo.Tb(&t, id, dbConn)
	if err != nil {
		switch e := err.Error(); e {
		case "le":
			return ctx.NoContent(http.StatusUnprocessableEntity)
		default:
			return ctx.JSON(http.StatusInternalServerError, err.Error())
			// return ctx.NoContent(http.StatusInternalServerError)

		}
	}

	return ctx.JSON(http.StatusOK, resp)
}

func Eh(ctx echo.Context) error {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return ctx.NoContent(http.StatusUnprocessableEntity)
	}

	client, err := repo.Gc(id, dbConn)
	if err != nil {
		return ctx.NoContent(http.StatusInternalServerError)
	}
	if client == nil {
		return ctx.NoContent(http.StatusNotFound)
	}

	ext, err := repo.Ex(id, dbConn)
	if err != nil {
		return ctx.JSON(http.StatusNotFound, err.Error())
		// return ctx.NoContent(http.StatusNotFound)
	}
	return ctx.JSON(http.StatusOK, ext)
}
