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
	createTableStmt          = `CREATE TABLE IF NOT EXISTS status(source text, status text, timestamp text);`
	createTableStmtMigration = `CREATE TABLE IF NOT EXISTS status(source text, status text, timestamp text, version text);`
	limit                    = 100
)

type Client struct {
	sqlDB   *sql.DB
	migrate bool
}

func NewPostgresClient(databaseURL string, migrate bool) (Client, error) {
	postgresClient := Client{
		migrate: migrate,
	}
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return postgresClient, err
	}
	postgresClient.sqlDB = db

	if migrate {
		err := postgresClient.migrateTable()
		if err != nil {
			return postgresClient, fmt.Errorf("error migrating table: %s", err)
		}
	} else {
		_, err = db.Exec(createTableStmt)
		if err != nil {
			return postgresClient, err
		}
	}
	return postgresClient, nil
}

func (c *Client) WriteSensorStatus(s config.SensorStatus) error {
	if c.migrate {
		stmt := "INSERT INTO status(source, status, timestamp, version) VALUES($1, $2, $3, $4)"
		_, err := c.sqlDB.Exec(stmt, s.Source, s.Status, s.Timestamp, s.Version)
		if err != nil {
			return err
		}
	} else {
		stmt := "INSERT INTO status(source, status, timestamp) VALUES($1, $2, $3)"
		_, err := c.sqlDB.Exec(stmt, s.Source, s.Status, s.Timestamp)
		if err != nil {
			return err
		}
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
		var err error
		if c.migrate {
			err = rows.Scan(&m.Source, &m.Status, &m.Timestamp, &m.Version)
		} else {
			err = rows.Scan(&m.Source, &m.Status, &m.Timestamp)
		}

		if err != nil {
			return nil, numPages, err
		}
		messages = append(messages, m)
	}
	err = rows.Err()
	return messages, numPages, nil
}

func (c *Client) migrateTable() error {
	stmt := `ALTER TABLE status ADD COLUMN IF NOT EXISTS version text DEFAULT '';`
	_, err := c.sqlDB.Exec(stmt)
	if err != nil {
		return err
	}

	stmt = `UPDATE status SET version='' WHERE version IS NULL;`
	_, err = c.sqlDB.Exec(stmt)
	if err != nil {
		return err
	}

	return nil
}
