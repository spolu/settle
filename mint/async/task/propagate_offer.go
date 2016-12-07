// OWNER: stan

package task

import (
	"context"
	"time"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/async"
	"github.com/spolu/settle/mint/model"
)

const (
	// TkPropagateOffer propagates an operation
	TkPropagateOffer mint.TkName = "PropagateOffer"
)

func init() {
	async.Registrar[TkPropagateOffer] = NewPropagateOffer
}

// PropagateOffer is in charge of propagating the offer to all required mints
// (up to one mint since the base asset of an offer must be owned by the offer
// owner).
type PropagateOffer struct {
	ID string
}

// NewPropagateOffer constructs and initializes the task.
func NewPropagateOffer(
	ctx context.Context,
	subject string,
) async.Task {
	return &PropagateOffer{
		ID: subject,
	}
}

// Name returns the task name.
func (t *PropagateOffer) Name() mint.TkName {
	return TkPropagateOffer
}

// Subject returns the task subject.
func (t *PropagateOffer) Subject() string {
	return t.ID
}

// MaxRetries returns the max retries for the task.
func (t *PropagateOffer) MaxRetries() uint {
	return 18
}

// DeadlineForRetry returns the deadline for the provided retry count.
func (t *PropagateOffer) DeadlineForRetry(
	retry uint,
) time.Time {
	return time.Now().Add((1<<retry - 1) * time.Second)
}

// Execute idempotently runs the task to completion or errors.
func (t *PropagateOffer) Execute(
	ctx context.Context,
) error {
	client := &mint.Client{}
	err := client.Init(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	offer, err := model.LoadCanonicalOfferByID(ctx, t.ID)
	if err != nil {
		return errors.Trace(err)
	} else if offer == nil {
		return errors.Trace(errors.Newf("Canonical offer not found: %s", t.ID))
	}

	db.Commit(ctx)

	asset, err := mint.AssetResourceFromName(ctx, offer.QuoteAsset)
	if err != nil {
		return errors.Trace(err)
	}

	_, host, err := mint.UsernameAndMintHostFromAddress(ctx, asset.Owner)
	if err != nil {
		return errors.Trace(err)
	}

	if host != mint.GetHost(ctx) {
		_, err := client.PropagateOffer(ctx, t.ID, host)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
