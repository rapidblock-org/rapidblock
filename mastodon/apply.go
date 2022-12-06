package mastodon

import (
	"context"
	"fmt"
	"time"

	"github.com/chronos-tachyon/rapidblock/blockapply"
	"github.com/chronos-tachyon/rapidblock/blockfile"
)

const WellKnownPrivateComment = "RapidBlock"

type Applier interface {
	Query(context.Context, map[string]DomainBlock) error
	Insert(context.Context, DomainBlock) error
	Update(context.Context, DomainBlock) error
	Delete(context.Context, DomainBlock) error
}

func Apply(ctx context.Context, dba Applier, file blockfile.BlockFile) (stats blockapply.Stats, err error) {
	existingBlocks := make(map[string]DomainBlock, 1024)
	err = dba.Query(ctx, existingBlocks)
	if err != nil {
		err = fmt.Errorf("failed to query the existing domain blocks: %w", err)
		return
	}

	now := time.Now().UTC()
	for domain, block := range file.Blocks {
		var goal DomainBlock
		goal.Domain = domain
		goal.PrivateComment = MakeNullString(true, WellKnownPrivateComment)
		goal.PublicComment = NullString{}
		goal.CreatedAt = now
		goal.UpdatedAt = now
		goal.Severity = SeverityNoOp
		goal.RejectMedia = false
		goal.RejectReports = false
		goal.Obfuscate = false

		if block.IsBlocked {
			goal.PublicComment = MakeNullString(true, block.Reason)
			goal.Severity = SeveritySuspend
		}

		existing, hasExisting := existingBlocks[domain]
		if hasExisting {
			goal.ID = existing.ID
			goal.CreatedAt = existing.CreatedAt
			goal.UpdatedAt = existing.UpdatedAt
		}
		if hasExisting && existing == goal {
			continue
		}
		if hasExisting && existing.PrivateComment != goal.PrivateComment {
			// If an admin has made a local decision for this domain, leave it alone.
			continue
		}

		goal.UpdatedAt = now

		switch {
		case block.IsBlocked && hasExisting:
			err = dba.Update(ctx, goal)
			if err != nil {
				err = fmt.Errorf("failed to update existing domain block: %q: %w", domain, err)
				return
			}
			stats.UpdateCount++

		case block.IsBlocked:
			err = dba.Insert(ctx, goal)
			if err != nil {
				err = fmt.Errorf("failed to insert new domain block: %q: %w", domain, err)
				return
			}
			stats.InsertCount++

		case hasExisting:
			err = dba.Delete(ctx, goal)
			if err != nil {
				err = fmt.Errorf("failed to delete existing domain block: %q: %w", domain, err)
				return
			}
			stats.DeleteCount++
		}
	}
	return
}
