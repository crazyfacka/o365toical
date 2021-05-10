package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	tableName = "logged_users"
)

type DBConfs struct {
	user     string
	password string
	host     string
	schema   string
}

type CachedData struct {
	db *sql.DB
}

func initCache(opts *DBConfs) (*CachedData, error) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s", opts.user, opts.password, opts.host, opts.schema))
	if err != nil {
		return nil, err
	}

	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	err = db.QueryRow("SHOW TABLES LIKE '" + tableName + "'").Scan(nil)
	if err == sql.ErrNoRows {
		_, tableErr := db.Exec("CREATE TABLE `" + tableName + "` (" +
			"`id` INT NOT NULL AUTO_INCREMENT," +
			"`user` VARCHAR(8) NOT NULL," +
			"`token` VARCHAR(60) NOT NULL," +
			"`last_updated` DATETIME NOT NULL," +
			"PRIMARY KEY (`id`)," +
			"UNIQUE INDEX `user_UNIQUE` (`user` ASC) VISIBLE," +
			"UNIQUE INDEX `token_UNIQUE` (`token` ASC) VISIBLE," +
			"UNIQUE INDEX `id_UNIQUE` (`id` ASC) VISIBLE);")

		if tableErr != nil {
			return nil, tableErr
		}
	} else if err != nil {
		return nil, err
	}

	return &CachedData{
		db: db,
	}, nil
}
