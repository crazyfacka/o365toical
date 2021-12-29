package main

import (
	"database/sql"
	"errors"
)

func tableExists(schema string, tableName string, sqlDriver string, db *sql.DB) error {
	var queriedTableName string

	switch sqlDriver {
	case "mysql":
		return db.QueryRow("SHOW TABLES LIKE '" + tableName + "'").Scan(&queriedTableName)
	case "postgres":
		// With PG one can do "CREATE IF NOT EXISTS"
		return sql.ErrNoRows
	}

	return errors.New("unknown driver")
}

func createLoggedUsersTable(sqlDriver string, db *sql.DB) error {
	var err error

	if sqlDriver == "mysql" {
		_, err = db.Exec("CREATE TABLE `" + loggedUsersTable + "` (" +
			"`id` INT NOT NULL AUTO_INCREMENT," +
			"`user` VARCHAR(8) NOT NULL," +
			"`token` VARCHAR(60) NOT NULL," +
			"`last_updated` DATETIME NOT NULL," +
			"PRIMARY KEY (`id`)," +
			"UNIQUE INDEX `user_UNIQUE` (`user` ASC) VISIBLE," +
			"UNIQUE INDEX `token_UNIQUE` (`token` ASC) VISIBLE," +
			"UNIQUE INDEX `id_UNIQUE` (`id` ASC) VISIBLE);")
	}

	if sqlDriver == "postgres" {
		_, err = db.Exec("CREATE TABLE IF NOT EXISTS " + loggedUsersTable + " (" +
			"id SERIAL," +
			"\"user\" VARCHAR(8) NOT NULL UNIQUE," +
			"token VARCHAR(60) NOT NULL UNIQUE," +
			"last_updated TIMESTAMP NOT NULL," +
			"PRIMARY KEY (id));")
	}

	return err
}

func createAttachmentsTable(sqlDriver string, db *sql.DB) error {
	var err error

	if sqlDriver == "mysql" {
		_, err = db.Exec("CREATE TABLE `" + attachmentsTable + "` (" +
			"`id` INT NOT NULL AUTO_INCREMENT," +
			"`att_id` VARCHAR(256) NOT NULL," +
			"`fname` VARCHAR(256) NOT NULL," +
			"`content_type` VARCHAR(128) NOT NULL," +
			"`last_updated` DATETIME NOT NULL," +
			"PRIMARY KEY (`id`)," +
			"UNIQUE INDEX `att_id_UNIQUE` (`att_id` ASC) VISIBLE," +
			"UNIQUE INDEX `id_UNIQUE` (`id` ASC) VISIBLE);")
	}

	if sqlDriver == "postgres" {
		_, err = db.Exec("CREATE TABLE IF NOT EXISTS " + attachmentsTable + " (" +
			"id SERIAL," +
			"att_id VARCHAR(256) NOT NULL UNIQUE," +
			"fname VARCHAR(256) NOT NULL," +
			"content_type VARCHAR(128) NOT NULL," +
			"last_updated TIMESTAMP NOT NULL," +
			"PRIMARY KEY (id));")
	}

	return err
}

func createMonthCacheTable(sqlDriver string, db *sql.DB) error {
	var err error

	if sqlDriver == "mysql" {
		_, err = db.Exec("CREATE TABLE `" + monthCacheTable + "` (" +
			"`id` INT NOT NULL AUTO_INCREMENT," +
			"`user` VARCHAR(8) NOT NULL," +
			"`start` DATETIME NOT NULL," +
			"`end` DATETIME NOT NULL," +
			"`contents` MEDIUMTEXT NOT NULL," +
			"`last_updated` DATETIME NOT NULL," +
			"PRIMARY KEY (`id`)," +
			"UNIQUE INDEX `id_UNIQUE` (`id` ASC) VISIBLE," +
			"UNIQUE KEY `idx_month_cache_start_end_user` (`start`,`end`,`user`) VISIBLE);")
	}

	if sqlDriver == "postgres" {
		_, err = db.Exec("CREATE TABLE IF NOT EXISTS " + monthCacheTable + " (" +
			"id SERIAL," +
			"\"user\" VARCHAR(8) NOT NULL," +
			"start TIMESTAMP NOT NULL," +
			"\"end\" TIMESTAMP NOT NULL," +
			"contents TEXT NOT NULL," +
			"last_updated TIMESTAMP NOT NULL," +
			"UNIQUE (start,\"end\",\"user\")," +
			"PRIMARY KEY (id));")
	}

	return err
}

func validateTables(schema string, sqlDriver string, db *sql.DB) error {
	var err error

	err = tableExists(schema, loggedUsersTable, sqlDriver, db)
	if err == sql.ErrNoRows {
		tableErr := createLoggedUsersTable(sqlDriver, db)

		if tableErr != nil {
			return tableErr
		}
	} else if err != nil {
		return err
	}

	err = tableExists(schema, attachmentsTable, sqlDriver, db)
	if err == sql.ErrNoRows {
		tableErr := createAttachmentsTable(sqlDriver, db)

		if tableErr != nil {
			return tableErr
		}
	} else if err != nil {
		return err
	}

	err = tableExists(schema, monthCacheTable, sqlDriver, db)
	if err == sql.ErrNoRows {
		tableErr := createMonthCacheTable(sqlDriver, db)

		if tableErr != nil {
			return tableErr
		}
	} else if err != nil {
		return err
	}

	return nil
}
