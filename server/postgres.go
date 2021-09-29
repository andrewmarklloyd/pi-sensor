package main

import (
	"database/sql"

	_ "github.com/lib/pq"
)

const (
	createTableStmt = `CREATE TABLE IF NOT EXISTS status(source text, status text, timestamp text);`
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

func (c *postgresClient) getAllSensorStatus() ([]Message, error) {
	stmt := "SELECT * FROM status ORDER by timestamp DESC"
	rows, err := c.client.Query(stmt)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		err := rows.Scan(&m.Source, &m.Status, &m.Timestamp)
		if err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	err = rows.Err()
	return messages, nil
}
