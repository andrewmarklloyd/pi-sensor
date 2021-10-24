package main

import (
	"database/sql"
	"math"
	"strings"

	_ "github.com/lib/pq"
)

const (
	createTableStmt = `CREATE TABLE IF NOT EXISTS status(source text, status text, timestamp text);`
	limit           = 100
)

type postgresClient struct {
	client *sql.DB
}

func newPostgresClient(databaseURL string) (postgresClient, error) {
	postgresClient := postgresClient{}
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return postgresClient, err
	}
	postgresClient.client = db
	_, err = db.Exec(createTableStmt)
	if err != nil {
		return postgresClient, err
	}
	return postgresClient, nil
}

func (c *postgresClient) writeSensorStatus(m Message) error {
	stmt := "INSERT INTO status(source, status, timestamp) VALUES($1, $2, $3)"
	_, err := c.client.Exec(stmt, m.Source, m.Status, m.Timestamp)
	if err != nil {
		return err
	}
	return nil
}

func (c *postgresClient) getSensorStatus(source string, page int) ([]Message, int, error) {

	if page < 1 {
		page = 1
	}
	offset := limit * (page - 1)

	var stmt string
	var rows *sql.Rows
	var countRow *sql.Row
	var err error
	numPages := 0
	if strings.ToLower(source) == "all" {
		stmt = "SELECT * FROM status ORDER by timestamp DESC LIMIT $1 OFFSET $2"
		rows, err = c.client.Query(stmt, limit, offset)

		countStmt := "SELECT COUNT(*) FROM status"
		countRow = c.client.QueryRow(countStmt)
	} else {
		stmt = `SELECT * FROM status WHERE source = $1 ORDER by timestamp DESC LIMIT $2 OFFSET $3`
		rows, err = c.client.Query(stmt, source, limit, offset)

		countStmt := "SELECT COUNT(*) FROM status WHERE source = $1"
		countRow = c.client.QueryRow(countStmt, source)
	}

	if err != nil {
		return nil, numPages, err
	}

	if countRow.Err() != nil {
		return nil, numPages, countRow.Err()
	}

	defer rows.Close()

	var count int
	err = countRow.Scan(&count)
	if err != nil {
		return nil, 0, err
	}

	numPages = int(math.Ceil(float64(count) / float64(limit)))

	var messages []Message
	for rows.Next() {
		var m Message
		err := rows.Scan(&m.Source, &m.Status, &m.Timestamp)
		if err != nil {
			return nil, numPages, err
		}
		messages = append(messages, m)
	}
	err = rows.Err()
	return messages, numPages, nil
}
