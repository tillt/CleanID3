package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/bogem/id3v2"
	"github.com/golang/glog"
)

// Clean input string.
//
// Removes additions if tainted by useless information.
func cleanedString(forbiddenWords []string, value string) (bool, string) {
	for _, dirt := range forbiddenWords {
		index := strings.Index(value, dirt)
		if index != -1 {
			// Those dirty tag additions commonly are trailing
			// useful data, assume we only need to remove the
			// dirt from the tail.
			runes := []rune(value)
			return true, strings.TrimSpace(string(runes[:index]))
		}
	}

	return false, value
}

// Process file.
//
// Parses the ID3 tags from the given file, removes occurances of
// forbidden words in any text frame, updates the file if needed.
func process(words []string, file string, dryRun bool) error {
	glog.Infof("Processing %s\n", file)

	// Open file and parse tag in it.
	tag, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err != nil {
		return err
	}
	defer tag.Close()

	isFileDirty := false

	// Going through all tag frames, checking text based frames for dirt.
	// URL based frames get killed alltogether, no matter what.
	//
	// Most text based frames identify themselves starting with `T`,
	// like eg. `TALB` or `TIT2` and are uniform in their data structure.
	// `TXXX` is an extension and needs extra handling. The `COMM`
	// comment frame is also demanding slightly different parsing
	// and patching. The `USLT` lyrics frame being yet another extra
	// for our purposes.
	//
	// TODO(tillt): Find a way to split this into functions or even
	// templates or alike.

	for k, s := range tag.AllFrames() {
		// Any text frame "T***".
		if k[0] == 'T' {
			for _, f := range s {
				if k != "TXXX" {
					// Any text frame that is not "TXXX".
					tf, _ := f.(id3v2.TextFrame)
					glog.Infof("%s: %s", k, tf.Text)

					cleaned, value := cleanedString(words, tf.Text)

					if cleaned {
						if len(value) > 0 {
							fmt.Printf("Updating %s: %s\n", k, value)
							tag.AddTextFrame(k, tf.Encoding, value)
						} else {
							fmt.Printf("Removing frame %s\n", k)
							tag.DeleteFrames(k)
						}
						isFileDirty = true
					}
				} else {
					// "TXXX".
					tf, _ := f.(id3v2.UserDefinedTextFrame)
					glog.Infof("%s: %s: %s", k, tf.Description, tf.Value)

					cleaned, value := cleanedString(words, tf.Value)

					if cleaned {
						if len(value) > 0 {
							fmt.Printf("Updating %s: %s\n", k, value)
							udtf := id3v2.UserDefinedTextFrame{
								Encoding:    tf.Encoding,
								Description: tf.Description,
								Value:       value,
							}
							tag.AddUserDefinedTextFrame(udtf)
						} else {
							fmt.Printf("Removing frame %s\n", k)
							tag.DeleteFrames(k)
						}
						isFileDirty = true
					}
				}
			}
		} else if k == "COMM" {
			// "COMM".
			for _, f := range s {
				cf, _ := f.(id3v2.CommentFrame)

				message := fmt.Sprintf("%s: ", k)
				if len(cf.Description) > 0 {
					message += fmt.Sprintf("%s: ", cf.Description)
				}
				if len(cf.Language) > 0 {
					message += fmt.Sprintf("%s: ", cf.Language)
				}
				message += cf.Text
				glog.Info(message)

				cleaned, value := cleanedString(words, cf.Text)

				if cleaned {
					if len(value) > 0 {
						fmt.Printf("Updating %s: %s\n", k, value)
						newcf := id3v2.CommentFrame{
							Encoding:    cf.Encoding,
							Language:    cf.Language,
							Description: cf.Description,
							Text:        value,
						}
						tag.AddCommentFrame(newcf)
					} else {
						fmt.Printf("Removing frame %s\n", k)
						tag.DeleteFrames(k)
					}
					isFileDirty = true
				}
			}
		} else if k == "USLT" {
			// "USLT".
			for _, f := range s {
				uslf, _ := f.(id3v2.UnsynchronisedLyricsFrame)

				message := fmt.Sprintf("%s: ", k)
				if len(uslf.ContentDescriptor) > 0 {
					message += fmt.Sprintf("%s: ", uslf.ContentDescriptor)
				}
				if len(uslf.Language) > 0 {
					message += fmt.Sprintf("%s: ", uslf.Language)
				}
				message += uslf.Lyrics
				glog.Info(message)

				cleaned, value := cleanedString(words, uslf.Lyrics)

				if cleaned {
					if len(value) > 0 {
						fmt.Printf("Updating %s: %s\n", k, value)
						newuslf := id3v2.UnsynchronisedLyricsFrame{
							Encoding:          uslf.Encoding,
							Language:          uslf.Language,
							ContentDescriptor: uslf.ContentDescriptor,
							Lyrics:            value,
						}
						tag.AddUnsynchronisedLyricsFrame(newuslf)
					} else {
						fmt.Printf("Removing frame %s\n", k)
						tag.DeleteFrames(k)
					}
					isFileDirty = true
				}
			}
		} else if k[0] == 'W' {
			// Any URL frame "W***".
			glog.Infof("%s: ????", k)
			fmt.Printf("Removing frame %s\n", k)
			tag.DeleteFrames(k)
			isFileDirty = true
		} else {
			// Any other frame id.
			glog.Infof("%s: ????", k)
		}
	}

	if isFileDirty {
		if !dryRun {
			glog.Info("Saving cleaned file")
			if err = tag.Save(); err != nil {
				return err
			}
		} else {
			glog.Info("Skipping save for dry run")
		}
	} else {
		glog.Info("File was clean already")
	}

	return nil
}

// Initialize text list.
func readLines(path string) ([]string, error) {
	// FIXME(tillt): I have no idea what I am doing here.
	lines := make([]string, 0, 512)

	file, err := os.Open(path)
	if err != nil {
		return lines, err
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

func main() {
	// Flags ininitializing.
	flag.Usage = usage

	verbose := flag.Bool("verbose", false, "shows debug info in stderr")

	dryRun := flag.Bool("dry", false, "do not write to file")

	forbiddenWordsPath :=
		flag.String(
			"forbidden",
			"/usr/local/share/cleanid3/forbidden.txt",
			"forbidden words list path")

	// Arguments parsing.
	flag.Parse()

	if *verbose {
		flag.Lookup("logtostderr").Value.Set("true")

		fmt.Println("Enabled verbose logging")
	}

	if *dryRun {
		fmt.Println("Disabled write to file for dry run")
	}

	var files []string

	if len(flag.Args()) < 1 {
		scanner := bufio.NewScanner(os.Stdin)

		for true {
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

	// Our actual job, cleaning ID3 tags of all the given `files`.
	for _, file := range files {
		if err := process(forbiddenWords, file, *dryRun); err != nil {
			glog.Fatal(err)
		}
	}

	glog.Flush()
}
