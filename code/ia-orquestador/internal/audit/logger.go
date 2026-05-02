package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type Logger struct {
	db *sql.DB
}

func NewLogger(db *sql.DB) *Logger {
	return &Logger{db: db}
}

func (l *Logger) Log(ctx context.Context, userID, action, resource string, metadata map[string]string) error {
	metaJSON := ""
	if metadata != nil {
		b, _ := json.Marshal(metadata)
		metaJSON = string(b)
	}
	_, err := l.db.ExecContext(ctx, `
		INSERT INTO audit_logs (user_id, action, resource, metadata, timestamp)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, action, resource, metaJSON, time.Now().UTC())
	return err
}
