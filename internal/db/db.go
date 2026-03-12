package db

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(path string) (*sql.DB, error) {
	if DB != nil {
		return DB, nil
	}

	var err error
	DB, err = sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	schema := `
CREATE TABLE IF NOT EXISTS tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
    type TEXT,
    command TEXT,
    cron TEXT,
    enabled INTEGER,
    chat_id INTEGER,
    notify INTEGER,
    prompt TEXT
);

CREATE TABLE IF NOT EXISTS task_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER,
    start_time TEXT,
    end_time TEXT,
    status TEXT,
    output TEXT
);

CREATE TABLE IF NOT EXISTS trumpstruth (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT,
    taskname TEXT,
    content TEXT
);
`

	_, err = DB.Exec(schema)
	if err != nil {
		return nil, err
	}

	// Try to add columns if they don't exist (migration for existing dbs)
	DB.Exec("ALTER TABLE tasks ADD COLUMN chat_id INTEGER DEFAULT 0;")
	DB.Exec("ALTER TABLE tasks ADD COLUMN notify INTEGER DEFAULT 1;")
	DB.Exec("ALTER TABLE tasks ADD COLUMN prompt TEXT DEFAULT '';")

	return DB, nil
}

func GetDB() *sql.DB {
	if DB == nil {
		log.Fatal("Database not initialized. Call Init() first.")
	}
	return DB
}
