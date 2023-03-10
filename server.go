package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/lib/pq"
)

type Expenses struct {
	ID     int      `json:"id"`
	Title  string   `json:"title"`
	Amount int      `json:"amount"`
	Note   string   `json:"note"`
	Tags   []string `json:"tags"`
}

type Err struct {
	Message string `json:"message"`
}

var db *sql.DB

func getExpenses(c echo.Context) error {
	stmt, err := db.Prepare("SELECT id, title, amount , note ,tags FROM expenses")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Err{Message: "can't prepare query all expenses:" + err.Error()})
	}

	rows, err := stmt.Query()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Err{Message: "can't query all expenses:" + err.Error()})
	}

	expenses := []Expenses{}

	for rows.Next() {
		exp := Expenses{}
		err := rows.Scan(&exp.ID, &exp.Title, &exp.Amount, &exp.Note, pq.Array(&exp.Tags))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, Err{Message: "can't scan expenses:" + err.Error()})
		}
		expenses = append(expenses, exp)
	}

	return c.JSON(http.StatusOK, expenses)
}

func getExpense(c echo.Context) error {
	id := c.Param("id")
	stmt, err := db.Prepare("SELECT id, title, amount , note ,tags FROM expenses WHERE id = $1")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Err{Message: "can't prepare query expenses :" + err.Error()})
	}

	row := stmt.QueryRow(id)
	exp := Expenses{}
	// u.Title, u.Amount, u.Note
	err = row.Scan(&exp.ID, &exp.Title, &exp.Amount, &exp.Note, pq.Array(&exp.Tags))
	switch err {
	case sql.ErrNoRows:
		return c.JSON(http.StatusNotFound, Err{Message: "expenses not found"})
	case nil:
		return c.JSON(http.StatusOK, exp)
	default:
		return c.JSON(http.StatusInternalServerError, Err{Message: "can't scan expenses:" + err.Error()})
	}
}
func createExpenses(c echo.Context) error {

	exp := Expenses{}
	err := c.Bind(&exp)
	if err != nil {
		return c.JSON(http.StatusBadRequest, Err{Message: err.Error()})
	}

	row := db.QueryRow("INSERT INTO expenses (title, amount , note ,tags) values ($1, $2, $3 , $4)  RETURNING id", exp.Title, exp.Amount, exp.Note, pq.Array(&exp.Tags))
	err = row.Scan(&exp.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Err{Message: err.Error()})
	}

	return c.JSON(http.StatusCreated, exp)
}

func updateExpenses(c echo.Context) error {

	id := c.Param("id")

	exp := Expenses{}
	err := c.Bind(&exp)
	if err != nil {

		return c.JSON(http.StatusBadRequest, Err{Message: err.Error()})
	}

	stmt, err := db.Prepare("UPDATE expenses SET title=$2, amount=$3 , note=$4 ,tags=$5 WHERE id=$1;")

	if err != nil {

		log.Fatal("can't prepare statment update", err)
	}

	if _, err := stmt.Exec(id, exp.Title, exp.Amount, exp.Note, pq.Array(&exp.Tags)); err != nil {
		log.Fatal("error execute update ", err)
	}
	stmt, err = db.Prepare("SELECT id, title, amount , note ,tags FROM expenses WHERE id = $1")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Err{Message: "can't prepare query expenses:" + err.Error()})
	}
	row := stmt.QueryRow(id)

	err = row.Scan(&exp.ID, &exp.Title, &exp.Amount, &exp.Note, pq.Array(&exp.Tags))
	switch err {
	case sql.ErrNoRows:
		return c.JSON(http.StatusNotFound, Err{Message: "expenses not found"})
	case nil:
		return c.JSON(http.StatusOK, exp)
	default:
		return c.JSON(http.StatusInternalServerError, Err{Message: "can't scan expenses:" + err.Error()})
	}

}

func main() {
	var err error
	url := "postgres://bvmnqtid:TYwIzLz0EPRo-v7Ztb8kYZ-PFjdUCNqE@john.db.elephantsql.com/bvmnqtid"
	db, err = sql.Open("postgres", url)
	if err != nil {
		log.Fatal("Connect to database error", err)
	}
	defer db.Close()

	createTb := `
	CREATE TABLE IF NOT EXISTS expenses ( id SERIAL PRIMARY KEY,
		title TEXT,
		amount FLOAT,
		note TEXT,
		tags TEXT[] );
	`
	_, err = db.Exec(createTb)

	if err != nil {
		log.Fatal("can't create table", err)
	}

	e := echo.New()

	e.Use(middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
		if username == "expenses" || password == "123456" {
			return true, nil
		}
		return false, nil
	}))

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/expenses", getExpenses)
	e.GET("/expenses/:id", getExpense)
	e.POST("/expenses", createExpenses)
	e.PUT("/expenses/:id", updateExpenses)
	log.Fatal(e.Start(":2565"))
}
