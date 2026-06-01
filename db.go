package main

import (
    "database/sql"
    "time"

    _ "github.com/mattn/go-sqlite3"
)

func initDB(path string) (*sql.DB, error) {
    db, err := sql.Open("sqlite3", path)
    if err != nil {
        return nil, err
    }

    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS events (
            id        INTEGER PRIMARY KEY AUTOINCREMENT,
            name      TEXT    NOT NULL,
            url       TEXT    NOT NULL,
            up        BOOLEAN NOT NULL,
            ts        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        )
    `)
    if err != nil {
        return nil, err
    }

    return db, nil
}

func logEvent(db *sql.DB, name, url string, up bool) error {
    _, err := db.Exec(
        `INSERT INTO events (name, url, up, ts) VALUES (?, ?, ?, ?)`,
        name, url, up, time.Now().UTC(),
    )
    return err
}

func calcUptime(db *sql.DB, name string, since time.Duration) float64 {
    cutoff := time.Now().UTC().Add(-since)
    row := db.QueryRow(`
        SELECT
            COUNT(*) FILTER (WHERE up = 1),
            COUNT(*)
        FROM events
        WHERE name = ? AND ts >= ?
    `, name, cutoff)

    var up, total int
    if err := row.Scan(&up, &total); err != nil || total == 0 {
        return -1
    }
    return float64(up) / float64(total) * 100
}