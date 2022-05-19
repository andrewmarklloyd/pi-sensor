package postgres

import (
	"database/sql"
	"fmt"
	"math"
	"strings"

	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
	_ "github.com/lib/pq"
)

const (
	createTableStmt = `CREATE TABLE IF NOT EXISTS status(source text, status text, timestamp text, version text);`
	limit           = 100
)

type Client struct {
	sqlDB *sql.DB
}

func NewPostgresClient(databaseURL string) (Client, error) {
	postgresClient := Client{}
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return postgresClient, err
	}
	postgresClient.sqlDB = db

	_, err = db.Exec(createTableStmt)
	if err != nil {
		return postgresClient, err
	}
	return postgresClient, nil
}

func (c *Client) WriteSensorStatus(s config.SensorStatus) error {
	stmt := "INSERT INTO status(source, status, timestamp, version) VALUES($1, $2, $3, $4)"
	_, err := c.sqlDB.Exec(stmt, s.Source, s.Status, s.Timestamp, s.Version)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) GetSensorStatus(source string, page int) ([]config.SensorStatus, int, error) {

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
		rows, err = c.sqlDB.Query(stmt, limit, offset)

		countStmt := "SELECT COUNT(*) FROM status"
		countRow = c.sqlDB.QueryRow(countStmt)
	} else {
		stmt = `SELECT * FROM status WHERE source = $1 ORDER by timestamp DESC LIMIT $2 OFFSET $3`
		rows, err = c.sqlDB.Query(stmt, source, limit, offset)

		countStmt := "SELECT COUNT(*) FROM status WHERE source = $1"
		countRow = c.sqlDB.QueryRow(countStmt, source)
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

	var messages []config.SensorStatus
	for rows.Next() {
		var m config.SensorStatus
		err := rows.Scan(&m.Source, &m.Status, &m.Timestamp, &m.Version)

		if err != nil {
			return nil, numPages, err
		}
		messages = append(messages, m)
	}
	err = rows.Err()
	return messages, numPages, nil
}

func (c *Client) GetRowCount() (int, error) {
	countStmt := "SELECT COUNT(*) FROM status"
	countRow := c.sqlDB.QueryRow(countStmt)

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

func (c *Client) DeleteRows(rowsAboveMax []config.SensorStatus) error {
	query := "DELETE FROM status WHERE timestamp BETWEEN $1 AND $2"
	firstRow := rowsAboveMax[0].Timestamp
	lastRow := rowsAboveMax[len(rowsAboveMax)-1].Timestamp
	res, err := c.sqlDB.Exec(query, firstRow, lastRow)
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

func (c *Client) GetRowsAboveMax(max int) ([]config.SensorStatus, error) {
	var statuses []config.SensorStatus

	rowCount, err := c.GetRowCount()
	if err != nil {
		return statuses, err
	}

	if rowCount <= max {
		return statuses, nil
	}

	rowsAboveMax := rowCount - max

	stmt := `SELECT * FROM status ORDER by timestamp ASC LIMIT $1`
	rows, err := c.sqlDB.Query(stmt, rowsAboveMax)
	defer rows.Close()

	for rows.Next() {
		var s config.SensorStatus
		err := rows.Scan(&s.Source, &s.Status, &s.Timestamp, &s.Version)
		if err != nil {
			return statuses, err
		}
		statuses = append(statuses, s)
	}
	return statuses, nil
}
