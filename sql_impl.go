package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
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
	db, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", opts.user, opts.password, opts.host, opts.schema))
	if err != nil {
		return err
	}

	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	if err := validateTables(opts.schema, db); err != nil {
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

	_, err = cd.db.Exec("INSERT INTO "+monthCacheTable+"(user, start, end, contents, last_updated) VALUES(?, ?, ?, ?, ?)", user, start, end, string(jsonData), time.Now())
	if err != nil {
		_, err = cd.db.Exec("UPDATE "+monthCacheTable+" SET contents = ?, last_updated = ? WHERE user = ? AND start = ? AND end = ?", string(jsonData), time.Now(), user, start, end)
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
