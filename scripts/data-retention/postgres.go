package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type Message struct {
	Source    string `json:"source"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

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
	return postgresClient, nil
}

func (c *postgresClient) getRowCount() (int, error) {
	countStmt := "SELECT COUNT(*) FROM status"
	countRow := c.client.QueryRow(countStmt)

	if countRow.Err() != nil {
		return -1, countRow.Err()
	}

	var rowCount int
	err := countRow.Scan(&rowCount)
	if err != nil {
		return -1, err
	}
	return rowCount, nil
}

func (c *postgresClient) deleteRows(rowsAboveMax []Message) error {
	query := "DELETE FROM status WHERE timestamp BETWEEN $1 AND $2"
	firstRow := rowsAboveMax[0].Timestamp
	lastRow := rowsAboveMax[len(rowsAboveMax)-1].Timestamp
	res, err := c.client.Exec(query, firstRow, lastRow)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if len(rowsAboveMax) == int(rowsAffected) {
		fmt.Println(fmt.Sprintf("Successfully deleted '%d' rows", rowsAffected))
	} else {
		fmt.Println(fmt.Sprintf("WARN: number of rows deleted '%d' did not match expected number '%d'. This could indicate a data loss situation", rowsAffected, len(rowsAboveMax)))
	}

	return nil
}

func (c *postgresClient) getRowsAboveMax(max int) ([]Message, error) {
	var messages []Message

	rowCount, err := c.getRowCount()
	if err != nil {
		return messages, err
	}

	if rowCount <= max {
		fmt.Println(fmt.Sprintf("Row count: '%d' is less than or equal to max: '%d', no action required", rowCount, max))
		return messages, nil
	}
	fmt.Println(fmt.Sprintf("Row count: '%d' is greater than max: '%d', getting rows above the max", rowCount, max))
	rowsAboveMax := rowCount - max
	fmt.Println(fmt.Sprintf("Number of rows above max: %d", rowsAboveMax))

	stmt := `SELECT * FROM status ORDER by timestamp ASC LIMIT $1`
	rows, err := c.client.Query(stmt, rowsAboveMax)
	defer rows.Close()

	for rows.Next() {
		var m Message
		err := rows.Scan(&m.Source, &m.Status, &m.Timestamp)
		if err != nil {
			return messages, err
		}
		messages = append(messages, m)
	}
	return messages, nil

}
