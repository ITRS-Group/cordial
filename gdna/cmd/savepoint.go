/*
Copyright © 2024 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/itrs-group/cordial/pkg/config"
)

// savepoints counts the savepoints created, so that every one has a
// name that is unique and is not derived from the data
var savepoints atomic.Uint64

func savepointName(n uint64) string {
	return "gdna_source_" + strconv.FormatUint(n, 10)
}

// errSavepoint is returned, joined with the error that caused the
// rollback, if the changes of a source cannot be rolled back, in which
// case the transaction cannot be used and must be abandoned.
var errSavepoint = errors.New("cannot roll back to savepoint")

// runSource calls load, which reads a source into the tables using tx,
// inside a savepoint that is released if it succeeds. If load fails, or
// the context is cancelled, then all the rows added to the tables are
// rolled back and, once that is done, the failure is recorded in the
// sources table, unless the context has been cancelled as then the
// transaction is being rolled back anyway. The error from load is
// returned and the transaction can be used for the other sources.
//
// All of the reports of a source, such as the detail and summary, must
// be loaded by one call so that they succeed or fail together.
func runSource(ctx context.Context, cf *config.Config, tx *sql.Tx, source, sourceType, path string, t time.Time, load func() error) (err error) {
	name := savepointName(savepoints.Add(1))
	if _, err = tx.ExecContext(ctx, "SAVEPOINT "+name); err != nil {
		return
	}

	if err = load(); err == nil {
		_, err = tx.ExecContext(ctx, "RELEASE "+name)
		return
	}

	// the rollback is not cancelled with the context, so that the
	// transaction is left consistent
	rctx := context.WithoutCancel(ctx)
	for _, query := range []string{"ROLLBACK TO " + name, "RELEASE " + name} {
		if _, rerr := tx.ExecContext(rctx, query); rerr != nil {
			if errors.Is(rerr, sql.ErrTxDone) {
				// already rolled back by the cancellation of the context
				// the transaction was started with
				return err
			}
			return errors.Join(err, errSavepoint, rerr)
		}
	}

	if ctx.Err() == nil {
		if uerr := updateSources(ctx, cf, tx, source, sourceType, path, false, t, err); uerr != nil {
			return errors.Join(err, uerr)
		}
	}
	return err
}
