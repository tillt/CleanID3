package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/tillt/cleanid3/pkg/cleanid3"
)

// Initialize text list.
func readLines(path string) ([]string, error) {
	// FIXME(tillt): I have no idea what I am doing here.
	lines := make([]string, 0, 512)

	file, err := os.Open(path)
	if err != nil {
		return lines, errors.Wrapf(err, "failed to open '%s'", path)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, nil
}

func usage() {
	flag.PrintDefaults()

	os.Exit(2)
}

var (
	buildVersion string = "undefined"
	buildTime    string = "?"
)

func main() {
	// Flags ininitializing.
	flag.Usage = usage

	showVersion :=
		flag.Bool(
			"version",
			false,
			"shows version information")

	verbose :=
		flag.Bool(
			"verbose",
			false,
			"shows debug info in stderr")

	dryRun :=
		flag.Bool(
			"dry",
			false,
			"do not write to file")

	forbiddenWordsPath :=
		flag.String(
			"forbidden",
			"/usr/local/share/cleanid3/forbidden.txt",
			"forbidden words list path")

	// Arguments parsing.
	flag.Parse()

	if *showVersion {
		fmt.Printf("%s (%s)\n", buildVersion, buildTime)
		os.Exit(1)
	}

	if *verbose {
		flag.Lookup("logtostderr").Value.Set("true")

		fmt.Println("Enabled verbose logging")
	}

	if *dryRun {
		fmt.Println("Disabled write to file for dry run")
	}

	var files []string

	// We are expecting input from stdin if there are no parameters.
	if len(flag.Args()) < 1 {
		scanner := bufio.NewScanner(os.Stdin)

		for {
			scanner.Scan()
			t := scanner.Text()

			if err := scanner.Err(); err != nil {
				glog.Fatal("Error reading from input: ", err)
			}
			if t == "" {
				break
			}

			files = append(files, t)
		}
	} else {
		files = flag.Args()
	}

	// Forbidden words list initializing.
	forbiddenWords, err := readLines(*forbiddenWordsPath)
	if err != nil {
		glog.Fatal("Error initializing blacklist: ", err)
	}
	for _, word := range forbiddenWords {
		glog.Infof("forbidden: \"%s\"", word)
	}

	var waitGroup sync.WaitGroup
	waitGroup.Add(len(files))

	// Our actual job, cleaning ID3 tags of all the given `files`.
	for _, file := range files {
		go func(file string) {
			if err := cleanid3.Clean(forbiddenWords, file, *dryRun); err != nil {
				glog.Error(err)
			}
			if !*dryRun {
				if err := cleanid3.RemoveID3V1(file, cleanid3.ID3V1_TAG_AT_UNKNOWN); err != nil {
					glog.Error(err)
				}
			}
			waitGroup.Done()
		}(file)
	}
	waitGroup.Wait()
	glog.Flush()
}
