package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	loggedUsersTable = "logged_users"
	attachmentsTable = "attachments"
	monthCacheTable  = "month_cache"
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

func initCache(opts *DBConfs) error {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", opts.user, opts.password, opts.host, opts.schema))
	if err != nil {
		return err
	}

	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	var queriedTableName string
	err = db.QueryRow("SHOW TABLES LIKE '" + loggedUsersTable + "'").Scan(&queriedTableName)
	if err == sql.ErrNoRows {
		_, tableErr := db.Exec("CREATE TABLE `" + loggedUsersTable + "` (" +
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

	err = db.QueryRow("SHOW TABLES LIKE '" + attachmentsTable + "'").Scan(&queriedTableName)
	if err == sql.ErrNoRows {
		_, tableErr := db.Exec("CREATE TABLE `" + attachmentsTable + "` (" +
			"`id` INT NOT NULL AUTO_INCREMENT," +
			"`att_id` VARCHAR(256) NOT NULL," +
			"`fname` VARCHAR(256) NOT NULL," +
			"`content_type` VARCHAR(128) NOT NULL," +
			"`last_updated` DATETIME NOT NULL," +
			"PRIMARY KEY (`id`)," +
			"UNIQUE INDEX `att_id_UNIQUE` (`att_id` ASC) VISIBLE," +
			"UNIQUE INDEX `id_UNIQUE` (`id` ASC) VISIBLE);")

		if tableErr != nil {
			return tableErr
		}
	} else if err != nil {
		return err
	}

	err = db.QueryRow("SHOW TABLES LIKE '" + monthCacheTable + "'").Scan(&queriedTableName)
	if err == sql.ErrNoRows {
		_, tableErr := db.Exec("CREATE TABLE `" + monthCacheTable + "` (" +
			"`id` INT NOT NULL AUTO_INCREMENT," +
			"`user` VARCHAR(8) NOT NULL," +
			"`start` DATETIME NOT NULL," +
			"`end` DATETIME NOT NULL," +
			"`contents` MEDIUMTEXT NOT NULL," +
			"`last_updated` DATETIME NOT NULL," +
			"PRIMARY KEY (`id`)," +
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
	_, err := cd.db.Exec("INSERT INTO "+loggedUsersTable+"(user, token, last_updated) VALUES(?, ?, ?)", user, token, time.Now())
	if err != nil {
		_, err = cd.db.Exec("UPDATE "+loggedUsersTable+" SET token = ?, last_updated = ? WHERE user = ?", token, time.Now(), user)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cd *CachedData) loadUserTokens() (map[string]string, error) {
	var user, token string

	tokens := make(map[string]string)

	rows, err := cd.db.Query("SELECT user, token FROM " + loggedUsersTable)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&user, &token)
		if err != nil {
			return nil, err
		}

		tokens[user] = token
	}

	return tokens, nil
}

func (cd *CachedData) attachmentExists(id string) []string {
	var fname, contentType string

	err := cd.db.QueryRow("SELECT fname, content_type FROM "+attachmentsTable+" WHERE att_id = ?", id).Scan(&fname, &contentType)
	if err != nil {
		return nil
	}

	return []string{fname, contentType}
}

func (cd *CachedData) saveAttachment(id string, name string, contentType string) error {
	_, err := cd.db.Exec("INSERT INTO "+attachmentsTable+"(att_id, fname, content_type, last_updated) VALUES(?, ?, ?, ?)", id, name, contentType, time.Now())
	if err != nil {
		return err
	}

	return nil
}

func (cd *CachedData) getCacheForUserLastUpdate(user string, start time.Time, end time.Time) (time.Time, error) {
	var lastUpdated sql.NullTime

	err := cd.db.QueryRow("SELECT last_updated FROM "+monthCacheTable+" WHERE user = ? AND start = ? AND end = ?", user, start, end).Scan(&lastUpdated)
	if err != nil {
		return time.Now(), err
	}

	return lastUpdated.Time, nil
}

func (cd *CachedData) saveCacheForUser(user string, start time.Time, end time.Time, cachedValues []interface{}) error {
	jsonData, err := json.Marshal(cachedValues)
	if err != nil {
		return err
	}

	_, err = cd.db.Exec("UPDATE "+monthCacheTable+" SET contents = ?, last_updated = ? WHERE user = ? AND start = ? AND end = ?", string(jsonData), time.Now(), user, start, end)
	if err != nil {
		_, err = cd.db.Exec("INSERT INTO "+monthCacheTable+"(user, start, end, contents, last_updated) VALUES(?, ?, ?, ?, ?)", user, start, end, string(jsonData), time.Now())
		if err != nil {
			return err
		}
	}

	return nil
}

func (cd *CachedData) getUserCache(user string, start time.Time, end time.Time) ([]interface{}, error) {
	var cachedValues []interface{}
	var cachedValuesText string

	err := cd.db.QueryRow("SELECT contents FROM "+monthCacheTable+" WHERE user = ? AND start = ? AND end = ?", user, start, end).Scan(&cachedValuesText)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(cachedValuesText), &cachedValues)
	if err != nil {
		return nil, err
	}

	return cachedValues, nil
}
