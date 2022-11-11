package main

import (
	"context"
	"fmt"
	"os"
	"time"

	pgx "github.com/jackc/pgx/v5"
)

func cmdApply() {
	switch {
	case flagDataFile == "":
		fmt.Fprintf(os.Stderr, "fatal: missing required flag -d / --data-file\n")
		os.Exit(1)
	case flagDatabaseURL == "":
		fmt.Fprintf(os.Stderr, "fatal: missing required flag -D / --database-url\n")
		os.Exit(1)
	}

	var file BlockFile
	ReadJsonFile(&file, flagDataFile)

	ctx := context.Background()
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

	sql := `SELECT domain FROM public.domain_blocks`
	rows, err := tx.Query(ctx, sql)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to Query %q: %v\n", sql, err)
		os.Exit(1)
	}
	defer rows.Close()

	existingBlocks := make(map[string]struct{}, 1024)
	for rows.Next() {
		var domain string
		err = rows.Scan(&domain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal: failed to process result row from Query %q: %v\n", sql, err)
			os.Exit(1)
		}
		existingBlocks[domain] = struct{}{}
	}

	err = rows.Err()
	rows.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to process results from Query %q: %v\n", sql, err)
		os.Exit(1)
	}

	now := time.Now().UTC()
	sql = `
	INSERT INTO public.domain_blocks
		(domain, created_at, updated_at, severity, reject_media, reject_reports, private_comment, public_comment, obfuscate)
	VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	args := make([]any, 9)
	args[1] = now
	args[2] = now
	args[3] = 1
	args[4] = true
	args[5] = true
	args[6] = "FediBlock"
	args[8] = false
	insertCount := 0
	for domain, block := range file.Blocks {
		if _, found := existingBlocks[domain]; found {
			continue
		}

		args[0] = domain
		args[7] = block.Reason
		_, err = tx.Exec(ctx, sql, args...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal: failed to Exec %q: %v\n", sql, err)
			os.Exit(1)
		}

		insertCount++
	}

	err = tx.Commit(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to Commit transaction: %v\n", err)
		os.Exit(1)
	}

	if insertCount <= 0 {
		return
	}

	fmt.Printf("added %d new block(s)\n", insertCount)
}
