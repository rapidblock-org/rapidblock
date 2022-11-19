package main

import (
	"context"
	"fmt"
	"os"
	"time"

	pgx "github.com/jackc/pgx/v5"
)

const RubyTimeFormat = "2006-01-02 15:04:05.000000000 Z07:00"

const (
	SeveritySilence = 0
	SeveritySuspend = 1
	SeverityNoOp    = 2
)

func cmdApply() {
	switch {
	case flagDataFile == "":
		fmt.Fprintf(os.Stderr, "fatal: missing required flag -d / --data-file\n")
		os.Exit(1)
	case flagDatabaseURL == "":
		fmt.Fprintf(os.Stderr, "fatal: missing required flag -D / --database-url\n")
		os.Exit(1)
	case flagSoftware == "":
		fmt.Fprintf(os.Stderr, "fatal: missing required flag -x / --software\n")
		os.Exit(1)
	}

	var applyFn func(context.Context, BlockFile) (int, int, int)
	switch flagSoftware {
	case Mastodon3x:
		applyFn = ApplyMastodon
	case Mastodon4x:
		applyFn = ApplyMastodon
	default:
		fmt.Fprintf(os.Stderr, "fatal: software %q not implemented\n", flagSoftware)
		os.Exit(1)
	}

	ctx := context.Background()

	var file BlockFile
	ReadJsonFile(&file, flagDataFile)

	insertCount, updateCount, deleteCount := applyFn(ctx, file)
	if insertCount > 0 {
		fmt.Printf("added %d new block(s)\n", insertCount)
	}
	if updateCount > 0 {
		fmt.Printf("modified %d existing block(s)\n", updateCount)
	}
	if deleteCount > 0 {
		fmt.Printf("deleted %d existing block(s) that are now remediated\n", deleteCount)
	}
}

func ApplyMastodon(ctx context.Context, file BlockFile) (insertCount int, updateCount int, deleteCount int) {
	conn, err := pgx.Connect(ctx, flagDatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to connect to PostgreSQL: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	tx, err := conn.Begin(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to Begin transaction: %v\n", err)
		os.Exit(1)
	}
	defer tx.Rollback(ctx)

	existingBlocks := GetMastodonDomainBlocks(ctx, tx)
	now := time.Now().UTC()

	for domain, block := range file.Blocks {
		// TODO: add Block.IsBlock and possibility of deleting blocks
		existing, hasExisting := existingBlocks[domain]

		// If an admin has made a local decision for this domain, leave it alone.
		if hasExisting && existing.PrivateComment != "RapidBlock" {
			continue
		}

		switch {
		case hasExisting && block.IsBlocked:
			updated := existing
			updated.PublicComment = block.Reason
			updated.Severity = SeveritySuspend
			updated.RejectMedia = false
			updated.RejectReports = false
			updated.Obfuscate = false
			if updated != existing {
				updated.UpdatedAt = now
				UpdateMastodonDomainBlock(ctx, tx, updated)
				updateCount++
			}

		case hasExisting:
			deleted := existing
			deleted.UpdatedAt = now
			DeleteMastodonDomainBlock(ctx, tx, deleted)
			deleteCount++

		case block.IsBlocked:
			var inserted MastodonDomainBlock
			inserted.Domain = domain
			inserted.PrivateComment = "RapidBlock"
			inserted.PublicComment = block.Reason
			inserted.CreatedAt = now
			inserted.UpdatedAt = now
			inserted.Severity = SeveritySuspend
			inserted.RejectMedia = false
			inserted.RejectReports = false
			inserted.Obfuscate = false
			InsertMastodonDomainBlock(ctx, tx, inserted)
			insertCount++

		default:
			// nothing to do
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to Commit transaction: %v\n", err)
		os.Exit(1)
	}
	return
}

func GetMastodonDomainBlocks(ctx context.Context, tx pgx.Tx) map[string]MastodonDomainBlock {
	const sql = `SELECT id, domain, private_comment, public_comment, created_at, updated_at, severity, reject_media, reject_reports, obfuscate FROM public.domain_blocks`
	rows, err := tx.Query(ctx, sql)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to Query %q: %v\n", sql, err)
		os.Exit(1)
	}
	defer rows.Close()

	out := make(map[string]MastodonDomainBlock, 1024)
	for rows.Next() {
		var row MastodonDomainBlock
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
			fmt.Fprintf(os.Stderr, "fatal: failed to process result row from Query %q: %v\n", sql, err)
			os.Exit(1)
		}
		out[row.Domain] = row
	}

	rows.Close()
	err = rows.Err()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to process results from Query %q: %v\n", sql, err)
		os.Exit(1)
	}

	return out
}

func InsertMastodonDomainBlock(ctx context.Context, tx pgx.Tx, block MastodonDomainBlock) {
	var args [9]any

	args[0] = block.Domain
	args[1] = block.PrivateComment
	args[2] = block.PublicComment
	args[3] = block.CreatedAt
	args[4] = block.UpdatedAt
	args[5] = block.Severity
	args[6] = block.RejectMedia
	args[7] = block.RejectReports
	args[8] = block.Obfuscate

	const sqlDomainBlock = `
	INSERT INTO public.domain_blocks
		(domain, private_comment, public_comment, created_at, updated_at, severity, reject_media, reject_reports, obfuscate)
	VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9)
	RETURNING id
	`

	var insertID uint64
	err := tx.QueryRow(ctx, sqlDomainBlock, args[:9]...).Scan(&insertID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to QueryRow %q: %v\n", sqlDomainBlock, err)
		os.Exit(1)
	}

	block.ID = insertID

	args[0] = block.CreatedAt
	args[1] = block.UpdatedAt
	args[2] = -99
	args[3] = "create"
	args[4] = "DomainBlock"
	args[5] = insertID

	var n int
	var sqlAuditLog string
	switch flagSoftware {
	case Mastodon3x:
		n = 7
		args[6] = block.AsYAML()
		sqlAuditLog = `
		INSERT INTO public.admin_action_logs
			(created_at, updated_at, account_id, action, target_type, target_id, recorded_changes)
		VALUES
			($1, $2, $3, $4, $5, $6, $7)
		`
	default:
		n = 9
		args[6] = block.Domain
		args[7] = ""
		args[8] = ""
		sqlAuditLog = `
		INSERT INTO public.admin_action_logs
			(created_at, updated_at, account_id, action, target_type, target_id, human_identifier, route_param, permalink)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`
	}

	_, err = tx.Exec(ctx, sqlAuditLog, args[:n]...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to Exec %q: %v\n", sqlAuditLog, err)
		os.Exit(1)
	}
}

func UpdateMastodonDomainBlock(ctx context.Context, tx pgx.Tx, block MastodonDomainBlock) {
	var args [9]any

	args[0] = block.ID
	args[1] = block.PrivateComment
	args[2] = block.PublicComment
	args[3] = block.UpdatedAt
	args[4] = block.Severity
	args[5] = block.RejectMedia
	args[6] = block.RejectReports
	args[7] = block.Obfuscate

	const sqlDomainBlock = `
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

	_, err := tx.Exec(ctx, sqlDomainBlock, args[:8]...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to Exec %q: %v\n", sqlDomainBlock, err)
		os.Exit(1)
	}

	args[0] = block.CreatedAt
	args[1] = block.UpdatedAt
	args[2] = -99
	args[3] = "update"
	args[4] = "DomainBlock"
	args[5] = block.ID

	var n int
	var sqlAuditLog string
	switch flagSoftware {
	case Mastodon3x:
		n = 7
		args[6] = block.AsYAML()
		sqlAuditLog = `
		INSERT INTO public.admin_action_logs
			(created_at, updated_at, account_id, action, target_type, target_id, recorded_changes)
		VALUES
			($1, $2, $3, $4, $5, $6, $7)
		`

	default:
		n = 9
		args[6] = block.Domain
		args[7] = ""
		args[8] = ""
		sqlAuditLog = `
		INSERT INTO public.admin_action_logs
			(created_at, updated_at, account_id, action, target_type, target_id, human_identifier, route_param, permalink)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`
	}

	_, err = tx.Exec(ctx, sqlAuditLog, args[:n]...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to Exec %q: %v\n", sqlAuditLog, err)
		os.Exit(1)
	}
}

func DeleteMastodonDomainBlock(ctx context.Context, tx pgx.Tx, block MastodonDomainBlock) {
	var args [9]any

	args[0] = block.ID

	const sqlDomainBlock = `DELETE FROM public.domain_blocks WHERE id = $1`

	_, err := tx.Exec(ctx, sqlDomainBlock, args[:1]...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to Exec %q: %v\n", sqlDomainBlock, err)
		os.Exit(1)
	}

	args[0] = block.CreatedAt
	args[1] = block.UpdatedAt
	args[2] = -99
	args[3] = "destroy"
	args[4] = "DomainBlock"
	args[5] = block.ID

	var n int
	var sqlAuditLog string
	switch flagSoftware {
	case Mastodon3x:
		n = 7
		args[6] = block.AsYAML()
		sqlAuditLog = `
		INSERT INTO public.admin_action_logs
			(created_at, updated_at, account_id, action, target_type, target_id, recorded_changes)
		VALUES
			($1, $2, $3, $4, $5, $6, $7)
		`

	default:
		n = 9
		args[6] = block.Domain
		args[7] = ""
		args[8] = ""
		sqlAuditLog = `
		INSERT INTO public.admin_action_logs
			(created_at, updated_at, account_id, action, target_type, target_id, human_identifier, route_param, permalink)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`
	}

	_, err = tx.Exec(ctx, sqlAuditLog, args[:n]...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to Exec %q: %v\n", sqlAuditLog, err)
		os.Exit(1)
	}
}

type MastodonDomainBlock struct {
	ID             uint64
	Domain         string
	PrivateComment string
	PublicComment  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Severity       int
	RejectMedia    bool
	RejectReports  bool
	Obfuscate      bool
}

func (block MastodonDomainBlock) AsYAML() string {
	var severity string
	switch block.Severity {
	case SeveritySilence:
		severity = "silence"
	case SeveritySuspend:
		severity = "suspend"
	default:
		severity = "noop"
	}

	createdAtStr := block.CreatedAt.Format(RubyTimeFormat)
	updatedAtStr := block.UpdatedAt.Format(RubyTimeFormat)

	return yamlToString(
		yamlMakeDoc(
			yamlMakeMap(
				yamlMakeString("id"),
				yamlMakeInt(block.ID),
				yamlMakeString("domain"),
				yamlMakeString(block.Domain),
				yamlMakeString("created_at"),
				yamlMakeTaggedMap(
					"!ruby/object:ActiveSupport::TimeWithZone",
					yamlMakeString("utc"),
					yamlMakeString(createdAtStr),
					yamlMakeString("zone"),
					yamlMakeTaggedMap(
						"!ruby/object:ActiveSupport::TimeZone",
						yamlMakeString("name"),
						yamlMakeString("Etc/UTC"),
					),
					yamlMakeString("time"),
					yamlMakeString(createdAtStr),
				),
				yamlMakeString("updated_at"),
				yamlMakeTaggedMap(
					"!ruby/object:ActiveSupport::TimeWithZone",
					yamlMakeString("utc"),
					yamlMakeString(updatedAtStr),
					yamlMakeString("zone"),
					yamlMakeTaggedMap(
						"!ruby/object:ActiveSupport::TimeZone",
						yamlMakeString("name"),
						yamlMakeString("Etc/UTC"),
					),
					yamlMakeString("time"),
					yamlMakeString(updatedAtStr),
				),
				yamlMakeString("severity"),
				yamlMakeString(severity),
				yamlMakeString("reject_media"),
				yamlMakeBool(block.RejectMedia),
				yamlMakeString("reject_reports"),
				yamlMakeBool(block.RejectReports),
				yamlMakeString("private_comment"),
				yamlMakeString(block.PrivateComment),
				yamlMakeString("public_comment"),
				yamlMakeString(block.PublicComment),
				yamlMakeString("obfuscate"),
				yamlMakeBool(block.Obfuscate),
			),
		),
	)
}
