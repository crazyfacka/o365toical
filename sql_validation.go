package main

import (
	"database/sql"
)

func createLoggedUsersTable(db *sql.DB) error {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS " + loggedUsersTable + " (" +
		"id SERIAL," +
		"\"user\" VARCHAR(8) NOT NULL UNIQUE," +
		"token VARCHAR(60) NOT NULL UNIQUE," +
		"last_updated TIMESTAMP NOT NULL," +
		"PRIMARY KEY (id));")

	return err
}

func createAttachmentsTable(db *sql.DB) error {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS " + attachmentsTable + " (" +
		"id SERIAL," +
		"att_id VARCHAR(256) NOT NULL UNIQUE," +
		"fname VARCHAR(256) NOT NULL," +
		"content_type VARCHAR(128) NOT NULL," +
		"last_updated TIMESTAMP NOT NULL," +
		"PRIMARY KEY (id));")

	return err
}

func createMonthCacheTable(db *sql.DB) error {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS " + monthCacheTable + " (" +
		"id SERIAL," +
		"\"user\" VARCHAR(8) NOT NULL," +
		"start TIMESTAMP NOT NULL," +
		"\"end\" TIMESTAMP NOT NULL," +
		"contents TEXT NOT NULL," +
		"last_updated TIMESTAMP NOT NULL," +
		"UNIQUE (start,\"end\",\"user\")," +
		"PRIMARY KEY (id));")

	return err
}

func validateTables(schema string, db *sql.DB) error {
	err := createLoggedUsersTable(db)

	if err != nil {
		return err
	}

	err = createAttachmentsTable(db)

	if err != nil {
		return err
	}

	err = createMonthCacheTable(db)

	if err != nil {
		return err
	}

	return nil
}
