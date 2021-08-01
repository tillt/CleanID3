package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"

	"github.com/bogem/id3v2"
	"github.com/golang/glog"
)

// Frame interface unifiying the basic functions we need accross frame
// types.
type Frame interface {
	// Log tag name and all important details for this frame.
	Log(string)
	// Update the tag with new frame information.
	UpdateTag(*id3v2.Tag, string, string)
	// Get displayed text we may need to clean.
	FrameText() string
}

//
// TextFrame.
//

// TextFrame pulls in the id3v2-package type to make it extendable.
type TextFrame id3v2.TextFrame

// FF FE 44 00 49 00 53 00 43 00 49 00 44 00 00 00

// Log for TextFrame.
func (tf TextFrame) Log(k string) {
	glog.Infof("%s: %s (%s)", k, tf.Text, tf.Encoding.Name)
}

// UpdateTag for TextFrame.
func (tf TextFrame) UpdateTag(tag *id3v2.Tag, k string, value string) {
	tag.AddTextFrame(k, tf.Encoding, value)
}

// FrameText for TextFrame.
func (tf TextFrame) FrameText() string {
	return tf.Text
}

//
// UserDefinedTextFrame.
//

// UserDefinedTextFrame pulls in the id3v2-package type to make it extendable.
type UserDefinedTextFrame id3v2.UserDefinedTextFrame

// Log for UserDefinedTextFrame.
func (tf UserDefinedTextFrame) Log(k string) {
	glog.Infof("%s: %s: %s (%s)", k, tf.Description, tf.Value, tf.Encoding.Name)
}

// UpdateTag for UserDefinedTextFrame.
func (tf UserDefinedTextFrame) UpdateTag(tag *id3v2.Tag, k string, value string) {
	udtf := id3v2.UserDefinedTextFrame{
		Encoding:    tf.Encoding,
		Description: tf.Description,
		Value:       value,
	}
	tag.AddUserDefinedTextFrame(udtf)
}

// FrameText for UserDefinedTextFrame.
func (tf UserDefinedTextFrame) FrameText() string {
	return tf.Value
}

//
// CommentFrame.
//

// CommentFrame pulls in the id3v2-package type to make it extendable.
type CommentFrame id3v2.CommentFrame

// Log for CommentFrame.
func (cf CommentFrame) Log(k string) {
	message := fmt.Sprintf("%s: ", k)
	if len(cf.Description) > 0 {
		message += fmt.Sprintf("%s: ", cf.Description)
	}
	if len(cf.Language) > 0 {
		message += fmt.Sprintf("%s: ", cf.Language)
	}
	message += cf.Text
	glog.Info(message)
}

// UpdateTag for CommentFrame.
func (cf CommentFrame) UpdateTag(tag *id3v2.Tag, k string, value string) {
	newcf := id3v2.CommentFrame{
		Encoding:    cf.Encoding,
		Language:    cf.Language,
		Description: cf.Description,
		Text:        value,
	}
	tag.AddCommentFrame(newcf)
}

// FrameText for CommentFrame.
func (cf CommentFrame) FrameText() string {
	return cf.Text
}

//
// UnsynchronisedLyricsFrame.
//

// UnsynchronisedLyricsFrame pulls in the id3v2-package type to make it extendable.
type UnsynchronisedLyricsFrame id3v2.UnsynchronisedLyricsFrame

// Log for UnsynchronisedLyricsFrame.
func (uslf UnsynchronisedLyricsFrame) Log(k string) {
	message := fmt.Sprintf("%s: ", k)
	if len(uslf.ContentDescriptor) > 0 {
		message += fmt.Sprintf("%s: ", uslf.ContentDescriptor)
	}
	if len(uslf.Language) > 0 {
		message += fmt.Sprintf("%s: ", uslf.Language)
	}
	message += uslf.Lyrics
	glog.Info(message)
}

// UpdateTag for UnsynchronisedLyricsFrame.
func (uslf UnsynchronisedLyricsFrame) UpdateTag(tag *id3v2.Tag, k string, value string) {
	newuslf := id3v2.UnsynchronisedLyricsFrame{
		Encoding:          uslf.Encoding,
		Language:          uslf.Language,
		ContentDescriptor: uslf.ContentDescriptor,
		Lyrics:            value,
	}
	tag.AddUnsynchronisedLyricsFrame(newuslf)
}

// FrameText for UnsynchronisedLyricsFrame.
func (uslf UnsynchronisedLyricsFrame) FrameText() string {
	return uslf.Lyrics
}

//
//
//

// Clean input string.
//
// Removes additions if tainted by useless information.
func cleanedString(forbiddenWords []string, value string) (bool, string) {
	// Left most dirty word location.
	smallestIndex := math.MaxInt32

	for _, dirt := range forbiddenWords {
		index := strings.Index(value, dirt)

		if index != -1 {
			if index < smallestIndex {
				smallestIndex = index
			}
		}
	}

	if smallestIndex == math.MaxInt32 {
		return false, value
	}

	// Those dirty tag additions commonly are trailing
	// useful data, assume we only need to remove the
	// dirt from the tail.
	runes := []rune(value)
	return true, strings.TrimSpace(string(runes[:smallestIndex]))
}

// Process file.
//
// Parses the ID3 tags from the given file, removes occurances of
// forbidden words in any text frame, update the file if needed.
func process(words []string, file string, dryRun bool) error {
	glog.Infof("Processing %s\n", file)

	// Open file and parse tag in it.
	tag, err := id3v2.Open(file, id3v2.Options{Parse: true})
	if err != nil {
		return err
	}
	defer tag.Close()

	isFileDirty := false

	for k, s := range tag.AllFrames() {
		// Any cleanable frame.
		if k[0] == 'T' || k == "COMM" || k == "USLT" {
			for _, f := range s {
				var frame Frame

				if k == "COMM" {
					// "COMM".
					cf, _ := f.(id3v2.CommentFrame)
					frame = CommentFrame(cf)
				} else if k == "USLT" {
					// "USLT".
					uslf, _ := f.(id3v2.UnsynchronisedLyricsFrame)
					frame = UnsynchronisedLyricsFrame(uslf)
				} else if k == "TXXX" {
					// "TXXX".
					tf, _ := f.(id3v2.UserDefinedTextFrame)
					frame = UserDefinedTextFrame(tf)
				} else {
					// Any text frame that is not "TXXX".
					tf, _ := f.(id3v2.TextFrame)
					frame = TextFrame(tf)
				}

				frame.Log(k)

				cleaned, value := cleanedString(words, frame.FrameText())

				if cleaned {
					if len(value) > 0 {
						fmt.Printf("Updating %s: %s\n", k, value)
						frame.UpdateTag(tag, k, value)
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

	var waitGroup sync.WaitGroup
	waitGroup.Add(len(files))

	// Our actual job, cleaning ID3 tags of all the given `files`.
	for _, file := range files {
		go func(file string) {
			if err := process(forbiddenWords, file, *dryRun); err != nil {
				glog.Error(err)
			}
			waitGroup.Done()
		}(file)
	}
	waitGroup.Wait()
	glog.Flush()
}
