package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestUpsertUserVoiceToken(t *testing.T) {
	os.Remove("./test.db")
	if db != nil {
		db.Close()
	}

	var err error
	db, err = sql.Open("sqlite3", "./test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		db.Close()
		os.Remove("./test.db")
	}()

	if _, err := db.Exec(createStmt); err != nil {
		log.Fatal(err)
	}

	if vt, err := GetUserVoiceToken("123"); !(vt == "" && err == nil) {
		t.Log(vt, err)
		t.FailNow()
	}
	if UpsertUserVoiceToken("123", "en-US"); err != nil {
		t.Log(err)
		t.FailNow()
	}
	if vt, err := GetUserVoiceToken("123"); !(vt == "en-US" && err == nil) {
		t.Log(vt, err)
		t.FailNow()
	}
	if UpsertUserVoiceToken("123", "ja-JP"); err != nil {
		t.Log(err)
		t.FailNow()
	}
	if vt, err := GetUserVoiceToken("123"); !(vt == "ja-JP" && err == nil) {
		t.Log(vt, err)
		t.FailNow()
	}
}
