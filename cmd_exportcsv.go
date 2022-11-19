package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"
)

func cmdExportCSV() {
	switch {
	case flagDataFile == "":
		fmt.Fprintf(os.Stderr, "fatal: missing required flag -d / --data-file\n")
		os.Exit(1)
	case flagCsvFile == "":
		fmt.Fprintf(os.Stderr, "fatal: missing required flag -c / --csv-file\n")
		os.Exit(1)
	}

	var file BlockFile
	ReadJsonFile(&file, flagDataFile)

	rows := make([][]string, 0, len(file.Blocks))
	for domain := range file.Blocks {
		row := make([]string, 1)
		row[0] = domain
		rows = append(rows, row)
	}
	sort.Sort(firstColumnDomainNameSort(rows))

	gBuffer.Reset()
	w := csv.NewWriter(&gBuffer)
	w.UseCRLF = true

	err := w.WriteAll(rows)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %q: failed to convert data to CSV: %v\n", flagCsvFile, err)
		os.Exit(1)
	}

	err = w.Error()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %q: failed to convert data to CSV: %v\n", flagCsvFile, err)
		os.Exit(1)
	}

	WriteFile(flagCsvFile, gBuffer.Bytes(), false)
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
