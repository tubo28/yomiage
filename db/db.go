package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
)

var (
	db *sql.DB
)

const (
	createStmt string = `
	create table if not exists guild (
		id integer not null primary key,
		discord_id string not null,
		language string
	);
	create table if not exists user (
		id integer not null primary key,
		discord_id string not null,
		language string,
		voice_token string
	);
	`
)

// Init creates tables if not exists
func Init() {
	var err error
	db, err = sql.Open("sqlite3", "db-data/app.db")
	if err != nil {
		log.Fatal("failed to open db: ", err)
	}

	if _, err := db.Exec(createStmt); err != nil {
		log.Fatal("failed to initialize db: ", err)
	}
}

// Close closes db
func Close() {
	if err := db.Close(); err != nil {
		log.Print("error closeing db: ", err.Error())
	}
}

// UpsertUserVoiceToken updates or inserts user's voice_token
func UpsertUserVoiceToken(userID, voiceToken string) error {
	return upsertImpl(userID, voiceToken, "voice_token")
}

// UpsertUserLanguage updates or inserts user's language
func UpsertUserLanguage(userID, voiceToken string) error {
	return upsertImpl(userID, voiceToken, "language")
}

func upsertImpl(userID, val, col string) error {
	var res string
	err := db.QueryRow(`select discord_id from user where discord_id = ? limit 1`, userID).Scan(&res)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("error select user by discord_id: %w", err)
	} else if errors.Is(err, sql.ErrNoRows) {
		if _, err := db.Exec(`insert into user(discord_id, `+col+`) values(?, ?)`, userID, val); err != nil {
			return fmt.Errorf("error insert new user: %w", err)
		}
		return nil
	} else {
		if _, err := db.Exec(`update user set `+col+` = ? where discord_id = ?`, val, userID); err != nil {
			return fmt.Errorf("error update user: %w", err)
		}
		return nil
	}
}

// GetUserVoiceToken get user's voice_token
func GetUserVoiceToken(userID string) (string, error) {
	return getImpl(userID, "voice_token")
}

// GetUserLanguage get user's language
func GetUserLanguage(userID string) (string, error) {
	return getImpl(userID, "language")
}

func getImpl(userID, col string) (string, error) {
	var res string
	err := db.QueryRow(`select `+col+` from user where discord_id = ? limit 1`, userID).Scan(&res)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("error get column of "+col+" by select user by discord_id: %w", err)
	} else if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	} else {
		return res, nil
	}
}
