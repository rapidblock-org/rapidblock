package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
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

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.UseCRLF = true
	for domain := range file.Blocks {
		w.Write([]string{domain})
	}
	w.Flush()
	WriteFile(flagCsvFile, buf.Bytes(), false)
}
