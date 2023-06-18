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

	clean :=
		flag.Bool(
			"clean",
			true,
			"clean file from unwanted garbage")

	enhance :=
		flag.Bool(
			"enhance",
			true,
			"enhance file with meta from path")

	dryRun :=
		flag.Bool(
			"dry",
			false,
			"do not write to file")

	forbiddenWordsPath :=
		flag.String(
			"forbidden-words",
			"/usr/local/share/cleanid3/forbidden-words.txt",
			"forbidden words list path")

	forbiddenBinariesPath :=
		flag.String(
			"forbidden-bins",
			"/usr/local/share/cleanid3/forbidden-bins.txt",
			"forbidden binaries list path")

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

	// Forbidden words and binaries lists initializing.
	forbiddenWords, err := readLines(*forbiddenWordsPath)
	if err != nil {
		glog.Fatal("Error initializing text blacklist: ", err)
	}
	for _, word := range forbiddenWords {
		glog.Infof("forbidden-word: \"%s\"", word)
	}

	forbiddenBinaries, err := readLines(*forbiddenBinariesPath)
	if err != nil {
		glog.Fatal("Error initializing binary blacklist: ", err)
	}
	for _, sha := range forbiddenBinaries {
		glog.Infof("forbidden-bin: \"%s\"", sha)
	}

	var waitGroup sync.WaitGroup
	waitGroup.Add(len(files))

	// Our actual job, cleaning ID3 tags of all the given `files`.
	for _, file := range files {
		go func(file string) {
			// Remove ID3V1 if existing.
			// TODO(tillt): Consider parsing it before destruction.
			if !*dryRun {
				err := cleanid3.RemoveID3V1(file, cleanid3.ID3V1TagAtUnknown)
				if err != nil {
					glog.Error(err)
				}
			}

			if *enhance {
				err = cleanid3.Enhance(file, *dryRun)
				if err != nil {
					glog.Error(err)
				}
			}

			if *clean {
				err = cleanid3.Clean(forbiddenWords, forbiddenBinaries, file, *dryRun)
				if err != nil {
					glog.Error(err)
				}
			}

			waitGroup.Done()
		}(file)
	}
	waitGroup.Wait()
	glog.Flush()
}
