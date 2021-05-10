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

var cachedData *CachedData

type DBConfs struct {
	user     string
	password string
	host     string
	schema   string
}

type CachedData struct {
	db *sql.DB
}

type UserToken struct {
	user  string
	token string
}

func initCache(opts *DBConfs) error {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s", opts.user, opts.password, opts.host, opts.schema))
	if err != nil {
		return err
	}

	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	var queriedTableName string
	err = db.QueryRow("SHOW TABLES LIKE '" + tableName + "'").Scan(&queriedTableName)
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
			return tableErr
		}
	} else if err != nil {
		return err
	}

	cachedData = &CachedData{
		db: db,
	}

	return nil
}

func (cd *CachedData) storeToken(user string, token string) error {
	_, err := cd.db.Exec("INSERT INTO "+tableName+"(user, token, last_updated) VALUES(?, ?, ?)", user, token, time.Now())
	if err != nil {
		_, err = cd.db.Exec("UPDATE "+tableName+" SET token = ?, last_updated = ? WHERE user = ?", token, time.Now(), user)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cd *CachedData) loadUserTokens() ([]*UserToken, error) {
	var user, token string
	var tokens []*UserToken

	rows, err := cd.db.Query("SELECT user, token FROM " + tableName)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&user, &token)
		if err != nil {
			return nil, err
		}

		tokens = append(tokens, &UserToken{
			user:  user,
			token: token,
		})
	}

	return tokens, nil
}
