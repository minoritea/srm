package testdata

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"testing"
)

var (
	db *sql.DB
)

func runTest(m *testing.M) error {
	err := exec.Command("go", "generate").Run()
	if err != nil {
		return fmt.Errorf("failed to generate test file: %w", err)
	}

	f, err := ioutil.TempFile("", "srm-test-*.db")
	if err != nil {
		return fmt.Errorf("failed to create database file: %w", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	db, err = sql.Open("sqlite3", f.Name())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE users(id text, name text, age int, created_at datetime)`)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE emailers(id text, email text)`)
	if err != nil {
		return fmt.Errorf("failed to create emailers table: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE programmers(id text, language text)`)
	if err != nil {
		return fmt.Errorf("failed to create emailers table: %w", err)
	}

	_, err = db.Exec(`INSERT INTO users VALUES('1', 'Alice', 20, '2021-07-21')`)
	if err != nil {
		return fmt.Errorf("failed to insert an user: %w", err)
	}

	_, err = db.Exec(`INSERT INTO emailers VALUES('1', 'alice@example.com')`)
	if err != nil {
		return fmt.Errorf("failed to insert an emailer: %w", err)
	}

	_, err = db.Exec(`INSERT INTO programmers VALUES('1', 'go')`)
	if err != nil {
		return fmt.Errorf("failed to insert a programmer: %w", err)
	}

	m.Run()
	return nil
}

func TestMain(m *testing.M) {
	err := runTest(m)
	if err != nil {
		log.Printf("%+v", err)
		os.Exit(1)
	}
}

func TestBind(t *testing.T) {
	var users Users
	err := users.Bind(
		db.Query(`select * from users u join emailers e on u.id = e.id join programmers p on u.id = p.id`),
	)
	if err != nil {
		t.Fatalf("failed to bind users: %v", err)
	}

	if len(users) == 0 {
		t.Fatalf("failed to bind users: %#v", users)
	}

	if users[0].Name != "Alice" {
		t.Fatalf("failed to bind users: %#v", users[0])
	}
}
