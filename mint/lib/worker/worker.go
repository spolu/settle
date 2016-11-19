package worker

import (
	"context"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/logging"
	"github.com/spolu/settle/mint/model"
)

// Worker retrieves, executes and updates Updates received on its channel.
type Worker struct {
	Ctx context.Context
	In  <-chan string
}

// Run starts the worker.
func (w *Worker) Run() {
	go func() {
		for token := range w.In {
			ctx := db.Begin(w.Ctx)
			defer db.LoggedRollback(ctx)

			update, err := model.LoadUpdateByToken(ctx, token)
			if err != nil {
				logging.Logf(ctx,
					"Error retrieveing update: update=%s error=%q",
					token, err.Error())
			} else {
				err := update.Execute(ctx)
				if err != nil {
					logging.Logf(ctx,
						"Error executing update: update=%s type=%s "+
							"subject=%s[%s] source=%s destination=%s "+
							"attempts=%d error=%q",
						update.Token, update.Type,
						update.SubjectOwner, update.SubjectToken,
						update.Source, update.Destination, update.Attempts,
						err.Error())
					update.Attempts++
				} else {
					update.Attempts++
					update.Status = model.UpStSucceeded
					update.Save(ctx)
				}
			}

			db.Commit(ctx)
		}
	}()
}

// ContextKey is the type of the key used with context to carry contextual
// environment.
type ContextKey string

const (
	// workerKey the context.Context key to store the worker.
	workerKey ContextKey = "worker.worker"
)

// With stores the environment in the provided context.
func With(
	ctx context.Context,
	worker *Worker,
) context.Context {
	return context.WithValue(ctx, workerKey, worker)
}

// Get returns the worker currently stored in the context.
func Get(
	ctx context.Context,
) *Worker {
	return ctx.Value(workerKey).(*Worker)
}
