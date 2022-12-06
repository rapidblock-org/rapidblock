package mastodon

import (
	"context"
	"fmt"

	pgx "github.com/jackc/pgx/v5"

	"github.com/chronos-tachyon/rapidblock/blockapply"
	"github.com/chronos-tachyon/rapidblock/blockfile"
)

const (
	serverAccountID       = -99
	adminActionCreate     = "create"
	adminActionUpdate     = "update"
	adminActionDestroy    = "destroy"
	targetTypeDomainBlock = "DomainBlock"
)

const sqlSelectDomainBlocks = `
SELECT
	id, domain, private_comment, public_comment, created_at, updated_at, severity, reject_media, reject_reports, obfuscate
FROM public.domain_blocks
`

const sqlInsertDomainBlocks = `
INSERT INTO public.domain_blocks
	(domain, private_comment, public_comment, created_at, updated_at, severity, reject_media, reject_reports, obfuscate)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id
`

const sqlUpdateDomainBlocks = `
UPDATE public.domain_blocks
SET
	private_comment = $2,
	public_comment = $3,
	updated_at = $4,
	severity = $5,
	reject_media = $6,
	reject_reports = $7,
	obfuscate = $8
WHERE
	id = $1
`

const sqlDeleteDomainBlocks = `
DELETE FROM public.domain_blocks WHERE id = $1
`

const sqlInsertAdminActionLog3x = `
INSERT INTO public.admin_action_logs
	(created_at, updated_at, account_id, action, target_type, target_id, recorded_changes)
VALUES
	($1, $2, $3, $4, $5, $6, $7)
`

const sqlInsertAdminActionLog4x = `
INSERT INTO public.admin_action_logs
	(created_at, updated_at, account_id, action, target_type, target_id, human_identifier, route_param, permalink)
VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9)
`

func ApplySQL(ctx context.Context, server blockapply.Server, file blockfile.BlockFile) (stats blockapply.Stats, err error) {
	conn, err := pgx.Connect(ctx, server.URI)
	if err != nil {
		err = fmt.Errorf("failed to connect to PostgreSQL: %w", err)
		return
	}
	defer conn.Close(ctx)

	tx, err := conn.Begin(ctx)
	if err != nil {
		err = fmt.Errorf("failed to Begin transaction: %w", err)
		return
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	applier := &SQLApplier{Tx: tx, Mode: server.Mode}

	stats, err = Apply(ctx, applier, file)
	if err != nil {
		return
	}

	err = tx.Commit(ctx)
	if err != nil {
		err = fmt.Errorf("failed to Commit transaction: %w", err)
	}
	return
}

type SQLApplier struct {
	Tx   pgx.Tx
	Mode blockapply.Mode
}

func (applier *SQLApplier) Query(ctx context.Context, out map[string]DomainBlock) error {
	const sql = sqlSelectDomainBlocks

	rows, err := applier.Tx.Query(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to Query %q: %w", sql, err)
	}
	defer rows.Close()

	for rows.Next() {
		var row DomainBlock
		err = rows.Scan(
			&row.ID,
			&row.Domain,
			&row.PrivateComment,
			&row.PublicComment,
			&row.CreatedAt,
			&row.UpdatedAt,
			&row.Severity,
			&row.RejectMedia,
			&row.RejectReports,
			&row.Obfuscate,
		)
		if err != nil {
			return fmt.Errorf("failed to process result row from Query %q: %w", sql, err)
		}
		out[row.Domain] = row
	}

	rows.Close()
	err = rows.Err()
	if err != nil {
		return fmt.Errorf("failed to process results from Query %q: %w", sql, err)
	}

	return nil
}

func (applier *SQLApplier) Insert(ctx context.Context, block DomainBlock) error {
	var args [9]any
	var sql string

	args[0] = block.Domain
	args[1] = block.PrivateComment
	args[2] = block.PublicComment
	args[3] = block.CreatedAt
	args[4] = block.UpdatedAt
	args[5] = block.Severity
	args[6] = block.RejectMedia
	args[7] = block.RejectReports
	args[8] = block.Obfuscate
	sql = sqlInsertDomainBlocks

	var insertID uint64
	err := applier.Tx.QueryRow(ctx, sql, args[:9]...).Scan(&insertID)
	if err != nil {
		return fmt.Errorf("failed to QueryRow %q: %w", sql, err)
	}

	block.ID = StringableU64(insertID)

	args[0] = block.CreatedAt
	args[1] = block.UpdatedAt
	args[2] = serverAccountID
	args[3] = adminActionCreate
	args[4] = targetTypeDomainBlock
	args[5] = insertID

	var n int
	switch applier.Mode {
	case blockapply.ModeMastodon3xSQL:
		n = 7
		args[6] = block.AsYAML()
		sql = sqlInsertAdminActionLog3x
	default:
		n = 9
		args[6] = block.Domain
		args[7] = ""
		args[8] = ""
		sql = sqlInsertAdminActionLog4x
	}

	_, err = applier.Tx.Exec(ctx, sql, args[:n]...)
	if err != nil {
		return fmt.Errorf("failed to Exec %q: %w", sql, err)
	}

	return nil
}

func (applier *SQLApplier) Update(ctx context.Context, block DomainBlock) error {
	var args [9]any
	var sql string

	args[0] = block.ID
	args[1] = block.PrivateComment
	args[2] = block.PublicComment
	args[3] = block.UpdatedAt
	args[4] = block.Severity
	args[5] = block.RejectMedia
	args[6] = block.RejectReports
	args[7] = block.Obfuscate
	sql = sqlUpdateDomainBlocks

	_, err := applier.Tx.Exec(ctx, sql, args[:8]...)
	if err != nil {
		return fmt.Errorf("failed to Exec %q: %w", sql, err)
	}

	args[0] = block.CreatedAt
	args[1] = block.UpdatedAt
	args[2] = serverAccountID
	args[3] = adminActionUpdate
	args[4] = targetTypeDomainBlock
	args[5] = block.ID

	var n int
	switch applier.Mode {
	case blockapply.ModeMastodon3xSQL:
		n = 7
		args[6] = block.AsYAML()
		sql = sqlInsertAdminActionLog3x
	default:
		n = 9
		args[6] = block.Domain
		args[7] = ""
		args[8] = ""
		sql = sqlInsertAdminActionLog4x
	}

	_, err = applier.Tx.Exec(ctx, sql, args[:n]...)
	if err != nil {
		return fmt.Errorf("failed to Exec %q: %w", sql, err)
	}

	return nil
}

func (applier *SQLApplier) Delete(ctx context.Context, block DomainBlock) error {
	var args [9]any
	var sql string

	args[0] = block.ID
	sql = sqlDeleteDomainBlocks

	_, err := applier.Tx.Exec(ctx, sql, args[:1]...)
	if err != nil {
		return fmt.Errorf("failed to Exec %q: %w", sql, err)
	}

	args[0] = block.CreatedAt
	args[1] = block.UpdatedAt
	args[2] = serverAccountID
	args[3] = adminActionDestroy
	args[4] = targetTypeDomainBlock
	args[5] = block.ID

	var n int
	switch applier.Mode {
	case blockapply.ModeMastodon3xSQL:
		n = 7
		args[6] = block.AsYAML()
		sql = sqlInsertAdminActionLog3x
	default:
		n = 9
		args[6] = block.Domain
		args[7] = ""
		args[8] = ""
		sql = sqlInsertAdminActionLog4x
	}

	_, err = applier.Tx.Exec(ctx, sql, args[:n]...)
	if err != nil {
		return fmt.Errorf("failed to Exec %q: %w", sql, err)
	}

	return nil
}

var _ Applier = (*SQLApplier)(nil)

func init() {
	blockapply.SetFunc(blockapply.ModeMastodon3xSQL, ApplySQL)
	blockapply.SetFunc(blockapply.ModeMastodon4xSQL, ApplySQL)
}
