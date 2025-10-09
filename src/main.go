package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"Hippocampus/client"
)

func main() {
	binary := flag.String("binary", "tree.bin", "database file")

	insertCmd := flag.NewFlagSet("insert", flag.ExitOnError)
	insertKey := insertCmd.String("key", "", "key/identifier for the text")
	insertText := insertCmd.String("text", "", "text to embed and store")
	
	csvCmd := flag.NewFlagSet("insert-csv", flag.ExitOnError)
	csvFile := csvCmd.String("csv", "csvFile.csv", "csv file")

	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
	searchText := searchCmd.String("text", "", "text to search for")
	searchEpsilon := searchCmd.Float64("e", 0.1, "search tolerance")
	searchTop := searchCmd.Int("top", 0, "limit results to top K")

	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  hippocampus insert -file tree.bin -key <id> -text <text>")
		fmt.Println("  hippocampus search -file tree.bin -text <text> -epsilon 0.1")
		os.Exit(1)
	}


	client, err := client.New(*binary, "us-east-1")
	if err != nil {
		fmt.Println(err)
	}

	switch os.Args[1] {
	case "insert":
		insertCmd.Parse(os.Args[2:])
		if *insertKey == "" || *insertText == "" {
			log.Fatal("both -key and -text are required")
		}
		client.Insert(*insertKey, *insertText)

	case "search":
		searchCmd.Parse(os.Args[2:])
		if *searchText == "" {
			log.Fatal("-text is required")
		}
		client.Search(*searchText, float32(*searchEpsilon), *searchTop)
	
	case "insert-csv":
		csvCmd.Parse(os.Args[2:])
		if *csvFile == "" {
			log.Fatalf("-file is required")
		}
		client.InsertCSV(*csvFile)

	default:
		log.Fatalf("unknown command: %s", os.Args[1])
	}
}
