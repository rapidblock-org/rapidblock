package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	pgx "github.com/jackc/pgx/v5"
	yaml "gopkg.in/yaml.v3"
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

	const sqlSelect = `SELECT domain FROM public.domain_blocks`
	rows, err := tx.Query(ctx, sqlSelect)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to Query %q: %v\n", sqlSelect, err)
		os.Exit(1)
	}
	defer rows.Close()

	existingBlocks := make(map[string]struct{}, 1024)
	for rows.Next() {
		var domain string
		err = rows.Scan(&domain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal: failed to process result row from Query %q: %v\n", sqlSelect, err)
			os.Exit(1)
		}
		existingBlocks[domain] = struct{}{}
	}

	rows.Close()
	err = rows.Err()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to process results from Query %q: %v\n", sqlSelect, err)
		os.Exit(1)
	}

	const sqlDomainBlock = `
	INSERT INTO public.domain_blocks
		(domain, created_at, updated_at, severity, reject_media, reject_reports, private_comment, public_comment, obfuscate)
	VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9)
	RETURNING id
	`
	const sqlAuditLog = `
	INSERT INTO public.admin_action_logs
		(account_id, action, target_type, target_id, recorded_changes, created_at, updated_at)
	VALUES
		($1, $2, $3, $4, $5, $6, $7)
	`

	now := time.Now().UTC()

	argsDomainBlock := make([]any, 9)
	argsDomainBlock[1] = now
	argsDomainBlock[2] = now
	argsDomainBlock[3] = 1
	argsDomainBlock[4] = true
	argsDomainBlock[5] = true
	argsDomainBlock[6] = "RapidBlock"
	argsDomainBlock[8] = false

	argsAuditLog := make([]any, 7)
	argsAuditLog[0] = -99
	argsAuditLog[1] = "create"
	argsAuditLog[2] = "DomainBlock"
	argsAuditLog[5] = now
	argsAuditLog[6] = now

	var lastAnchorID uint64
	makeLiteral := func(tag string, value string) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value}
	}
	makeString := func(value string) *yaml.Node {
		node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str"}
		node.SetString(value)
		return node
	}
	makeMap := func(content ...*yaml.Node) *yaml.Node {
		return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: content}
	}
	makeAlias := func(source *yaml.Node) *yaml.Node {
		if source.Anchor == "" {
			lastAnchorID++
			source.Anchor = strconv.FormatUint(lastAnchorID, 10)
		}
		return &yaml.Node{
			Kind:  yaml.AliasNode,
			Style: source.Style,
			Tag:   source.Tag,
			Alias: source,
			Value: source.Anchor,
		}
	}
	makeTaggedMap := func(tag string, content ...*yaml.Node) *yaml.Node {
		return &yaml.Node{
			Kind:    yaml.MappingNode,
			Style:   yaml.TaggedStyle,
			Tag:     tag,
			Content: content,
		}
	}

	yamlKeyID := makeString("id")
	yamlKeyDomain := makeString("domain")
	yamlKeyCreatedAt := makeString("created_at")
	yamlKeyUpdatedAt := makeString("updated_at")
	yamlKeySeverity := makeString("severity")
	yamlKeyRejectMedia := makeString("reject_media")
	yamlKeyRejectReports := makeString("reject_reports")
	yamlKeyPrivateComment := makeString("private_comment")
	yamlKeyPublicComment := makeString("public_comment")
	yamlKeyObfuscate := makeString("obfuscate")
	yamlKeyUTC := makeString("utc")
	yamlKeyZone := makeString("zone")
	yamlKeyName := makeString("name")
	yamlKeyTime := makeString("time")

	yamlBoolFalse := makeLiteral("!!bool", "false")
	yamlBoolTrue := makeLiteral("!!bool", "true")
	yamlStringEtcUTC := makeString("Etc/UTC")
	yamlStringSuspend := makeString("suspend")
	yamlStringRapidBlock := makeString("RapidBlock")

	yamlIntIDValue := makeLiteral("!!int", "%$#!@ REPLACE ME")
	yamlStringDomainValue := makeString("%$#!@ REPLACE ME")
	yamlStringPublicCommentValue := makeString("%$#!@ REPLACE ME")

	yamlStringTimeValue := makeString(now.Format("2006-01-02 15:04:05.000000000 Z07:00"))
	yamlAliasTimeValue := makeAlias(yamlStringTimeValue)

	yamlTimeZone := makeTaggedMap(
		"!ruby/object:ActiveSupport::TimeZone",
		yamlKeyName,
		yamlStringEtcUTC,
	)
	yamlTimeWithZone := makeTaggedMap(
		"!ruby/object:ActiveSupport::TimeWithZone",
		yamlKeyUTC,
		yamlStringTimeValue,
		yamlKeyZone,
		yamlTimeZone,
		yamlKeyTime,
		yamlAliasTimeValue,
	)
	yamlAliasTimeWithZone := makeAlias(yamlTimeWithZone)

	yamlDocMap := makeMap(
		yamlKeyID,
		yamlIntIDValue,
		yamlKeyDomain,
		yamlStringDomainValue,
		yamlKeyCreatedAt,
		yamlTimeWithZone,
		yamlKeyUpdatedAt,
		yamlAliasTimeWithZone,
		yamlKeySeverity,
		yamlStringSuspend,
		yamlKeyRejectMedia,
		yamlBoolTrue,
		yamlKeyRejectReports,
		yamlBoolTrue,
		yamlKeyPrivateComment,
		yamlStringRapidBlock,
		yamlKeyPublicComment,
		yamlStringPublicCommentValue,
		yamlKeyObfuscate,
		yamlBoolFalse,
	)
	yamlDoc := &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{yamlDocMap},
	}

	var buf bytes.Buffer
	insertCount := 0
	for domain, block := range file.Blocks {
		if _, found := existingBlocks[domain]; found {
			continue
		}

		argsDomainBlock[0] = domain
		argsDomainBlock[7] = block.Reason

		var insertID uint64
		err = tx.QueryRow(ctx, sqlDomainBlock, argsDomainBlock...).Scan(&insertID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal: failed to QueryRow %q: %v\n", sqlDomainBlock, err)
			os.Exit(1)
		}

		yamlIntIDValue.Value = strconv.FormatUint(insertID, 10)
		yamlStringDomainValue.SetString(domain)
		yamlStringPublicCommentValue.SetString(block.Reason)

		buf.Reset()
		buf.WriteString("---\n")

		e := yaml.NewEncoder(&buf)
		e.SetIndent(2)
		err = e.Encode(yamlDoc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal: failed to encode audit log data to YAML: %v\n", err)
			os.Exit(1)
		}

		err = e.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal: failed to encode audit log data to YAML: %v\n", err)
			os.Exit(1)
		}

		argsAuditLog[3] = insertID
		argsAuditLog[4] = buf.String()

		_, err = tx.Exec(ctx, sqlAuditLog, argsAuditLog...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal: failed to Exec %q: %v\n", sqlAuditLog, err)
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
