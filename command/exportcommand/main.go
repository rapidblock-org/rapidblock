package exportcommand

import (
	"bytes"
	"context"
	"encoding/csv"
	"sort"
	"strings"

	getopt "github.com/pborman/getopt/v2"
	"github.com/rs/zerolog"

	"github.com/chronos-tachyon/rapidblock/blockfile"
	"github.com/chronos-tachyon/rapidblock/command"
	"github.com/chronos-tachyon/rapidblock/internal/iohelpers"
)

type exportFactory struct {
	command.BaseFactory
}

func (exportFactory) Name() string {
	return "export"
}

func (exportFactory) Aliases() []string {
	return []string{"export-csv"}
}

func (exportFactory) Description() string {
	return "Exports the RapidBlock blocklist file to Mastodon's CSV format."
}

func (exportFactory) New(dispatcher command.Dispatcher) (*getopt.Set, command.MainFunc) {
	var (
		inputFile  string
		outputFile string
	)

	options := getopt.New()
	options.SetParameters("")
	options.FlagLong(&inputFile, "input-file", 'i', "path to the JSON file to export from")
	options.FlagLong(&outputFile, "output-file", 'o', "path to the CSV file to export to")

	return options, func(ctx context.Context) int {
		if inputFile == "" {
			zerolog.Ctx(ctx).
				Error().
				Msg("missing required flag -i / --input-file")
			return 1
		}
		if outputFile == "" {
			zerolog.Ctx(ctx).
				Error().
				Msg("missing required flag -o / --output-file")
			return 1
		}
		return Main(ctx, inputFile, outputFile)
	}
}

var Factory command.Factory = exportFactory{}

func Main(ctx context.Context, inputFile string, outputFile string) int {
	logger := zerolog.Ctx(ctx).
		With().
		Str("inputFile", inputFile).
		Str("outputFile", outputFile).
		Logger()

	var file blockfile.BlockFile
	err := iohelpers.Load(&file, inputFile, true)
	if err != nil {
		logger.Error().Err(err).Send()
		return 1
	}

	rows := make([][]string, 0, len(file.Blocks))
	for domain := range file.Blocks {
		row := make([]string, 1)
		row[0] = domain
		rows = append(rows, row)
	}
	sort.Sort(firstColumnDomainNameSort(rows))

	var buf bytes.Buffer
	buf.Grow(1 << 16) // 64 KiB
	w := csv.NewWriter(&buf)
	w.UseCRLF = true
	err = w.WriteAll(rows)
	if err != nil {
		logger.Error().Err(err).Send()
		return 1
	}
	err = w.Error()
	if err != nil {
		logger.Error().Err(err).Send()
		return 1
	}
	err = iohelpers.WriteFile(outputFile, false, buf.Bytes())
	if err != nil {
		logger.Error().Err(err).Send()
		return 1
	}
	return 0
}

type firstColumnDomainNameSort [][]string

func (list firstColumnDomainNameSort) Len() int {
	return len(list)
}

func (list firstColumnDomainNameSort) Less(i, j int) bool {
	a := list[i][0]
	b := list[j][0]
	aList := splitDomainName(a)
	bList := splitDomainName(b)
	aLen := uint(len(aList))
	bLen := uint(len(bList))
	minLen := aLen
	if minLen > bLen {
		minLen = bLen
	}
	for k := uint(0); k < minLen; k++ {
		aWord := aList[k]
		bWord := bList[k]
		if aWord < bWord {
			return true
		}
		if aWord > bWord {
			return false
		}
	}
	return aLen < bLen
}

func (list firstColumnDomainNameSort) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

var _ sort.Interface = firstColumnDomainNameSort(nil)

var splitDomainNameCache map[string][]string

func splitDomainName(str string) []string {
	str = strings.TrimRight(str, ".")
	if cached, found := splitDomainNameCache[str]; found {
		return cached
	}

	pieces := strings.Split(str, ".")
	i := uint(0)
	j := uint(len(pieces)) - 1
	for i < j {
		pieces[i], pieces[j] = pieces[j], pieces[i]
		i++
		j--
	}
	if splitDomainNameCache == nil {
		splitDomainNameCache = make(map[string][]string, 1024)
	}
	splitDomainNameCache[str] = pieces
	return pieces
}
