package preparecommand

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	getopt "github.com/pborman/getopt/v2"
	"github.com/rs/zerolog"

	"github.com/chronos-tachyon/rapidblock/blockfile"
	"github.com/chronos-tachyon/rapidblock/command"
	"github.com/chronos-tachyon/rapidblock/internal/groupsio"
	"github.com/chronos-tachyon/rapidblock/internal/httpclient"
	"github.com/chronos-tachyon/rapidblock/internal/iohelpers"
)

type prepareFactory struct {
	command.BaseFactory
}

func (prepareFactory) Name() string {
	return "prepare"
}

func (prepareFactory) Description() string {
	return "Produces a RapidBlock blocklist file by pulling data from a spreadsheet."
}

func (prepareFactory) New(dispatcher command.Dispatcher) (*getopt.Set, command.MainFunc) {
	var (
		configFile string
		dataFile   string
	)

	options := getopt.New()
	options.SetParameters("")
	options.FlagLong(&configFile, "config-file", 'c', "path to the groups.io cookies and database column mappings")
	options.FlagLong(&dataFile, "data-file", 'd', "path to the JSON file to create, export from, sign, verify, or apply")

	return options, func(ctx context.Context) int {
		if configFile == "" {
			zerolog.Ctx(ctx).
				Error().
				Msg("missing required flag -c / --config-file")
			return 1
		}
		if dataFile == "" {
			zerolog.Ctx(ctx).
				Error().
				Msg("missing required flag -d / --data-file")
			return 1
		}
		return Main(ctx, configFile, dataFile)
	}
}

var Factory command.Factory = prepareFactory{}

func Main(ctx context.Context, configFile string, dataFile string) int {
	logger := zerolog.Ctx(ctx).
		With().
		Str("configFile", configFile).
		Str("dataFile", dataFile).
		Logger()

	var config groupsio.AccountConfig
	err := iohelpers.Load(&config, configFile, true)
	if err != nil {
		logger.Error().Err(err).Send()
		return 1
	}

	var file blockfile.BlockFile
	file.Spec = blockfile.SpecV1
	file.PublishedAt = time.Now().UTC()
	file.Blocks = make(map[string]blockfile.Block, 1024)

	baseURL := &url.URL{
		Scheme:  "https",
		Host:    "groups.io",
		Path:    "/api/v1/getdatabaserows",
		RawPath: "/api/v1/getdatabaserows",
	}
	baseQuery := make(url.Values, 2)
	baseQuery.Set("database_id", strconv.FormatUint(config.DatabaseID, 10))
	baseQuery.Set("limit", "100")

	userAgent := httpclient.UserAgent
	cookie, hasCookie := config.CookieString()

	err = groupsio.ForEach(
		ctx,
		http.DefaultClient,
		baseURL,
		baseQuery,
		func(req *http.Request) {
			if req.Header == nil {
				req.Header = make(http.Header, 16)
			}
			req.Header.Set("user-agent", userAgent)
			if hasCookie {
				req.Header.Set("cookie", cookie)
			}
		},
		func(row groupsio.DatabaseRow) error {
			var block blockfile.Block
			var domain string
			var hasDomain bool

			for _, value := range row.Values {
				columnData := config.Columns[value.ID]
				switch columnData.ID {
				case groupsio.DomainColumn:
					domain = value.AsString()
					hasDomain = true
				case groupsio.IsBlockedColumn:
					block.IsBlocked = value.AsBool()
				case groupsio.DateRequestedColumn:
					block.DateRequested = value.AsTime()
				case groupsio.DateDecidedColumn:
					block.DateDecided = value.AsTime()
				case groupsio.ReasonColumn:
					block.Reason = value.AsString()
				case groupsio.TagsColumn:
					block.Tags = sortTags(value.AsSet(columnData.Choices))
				}
			}

			if hasDomain && !block.DateDecided.IsZero() && !block.DateDecided.After(file.PublishedAt) {
				file.Blocks[domain] = block
			}
			return nil
		},
	)
	if err != nil {
		logger.Error().Err(err).Send()
		return 1
	}

	err = iohelpers.Store(dataFile, false, file)
	if err != nil {
		logger.Error().Err(err).Send()
		return 1
	}
	return 0
}

func sortTags(set map[string]struct{}) []string {
	list := make([]string, 0, len(set))
	for tag := range set {
		list = append(list, tag)
	}
	sort.Strings(list)
	return list
}
