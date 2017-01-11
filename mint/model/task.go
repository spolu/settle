package model

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/lib/token"
	"github.com/spolu/settle/mint"
)

// Task represents a task object.
type Task struct {
	Token   string
	Created time.Time

	Name    mint.TkName
	Subject string

	Status mint.TkStatus
	Retry  uint
}

// CreateTask creates and stores a new Task.
func CreateTask(
	ctx context.Context,
	created time.Time,
	name mint.TkName,
	subject string,
	status mint.TkStatus,
	retry uint,
) (*Task, error) {
	task := Task{
		Token:   token.New("task"),
		Created: created.UTC(),

		Name:    name,
		Subject: subject,
		Status:  status,
		Retry:   retry,
	}

	ext := db.Ext(ctx, "mint")
	if _, err := sqlx.NamedExec(ext, `
INSERT INTO tasks
  (token, created, name, subject, status, retry)
VALUES
  (:token, :created, :name, :subject, :status, :retry)
`, task); err != nil {
		switch err := err.(type) {
		case *pq.Error:
			if err.Code.Name() == "unique_violation" {
				return nil, errors.Trace(ErrUniqueConstraintViolation{err})
			}
		case sqlite3.Error:
			if err.ExtendedCode == sqlite3.ErrConstraintUnique {
				return nil, errors.Trace(ErrUniqueConstraintViolation{err})
			}
		}
		return nil, errors.Trace(err)
	}

	return &task, nil
}

// Save updates the object database representation with the in-memory values.
func (o *Task) Save(
	ctx context.Context,
) error {
	ext := db.Ext(ctx, "mint")
	_, err := sqlx.NamedExec(ext, `
UPDATE tasks
SET status = :status, retry = :retry
WHERE token = :token
`, o)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

// LoadPendingTasks loads all tasks that are marked as pending.
func LoadPendingTasks(
	ctx context.Context,
) ([]*Task, error) {
	query := Task{
		Status: mint.TkStPending,
	}

	ext := db.Ext(ctx, "mint")
	rows, err := sqlx.NamedQuery(ext, `
SELECT *
FROM tasks
WHERE status = :status
`, query)
	if err != nil {
		return nil, errors.Trace(err)
	}

	tasks := []*Task{}

	defer rows.Close()
	for rows.Next() {
		op := Task{}
		err := rows.StructScan(&op)
		if err != nil {
			return nil, errors.Trace(err)
		}
		tasks = append(tasks, &op)
	}

	return tasks, nil
}
