package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/TwiN/go-color"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

var start time.Time
var username, password, combolist string

// Function to handle parsing errors
func errorpars(err error) {
	if err != nil {
		log.Panic(err.Error())
	}
}

func main() {
	start = time.Now()
	runtime.GOMAXPROCS(1)
	// Command-line interface setup using urfave/cli
	app := &cli.App{
		Flags: []cli.Flag{
			// Flags for URL, username, password, rate limit, delay, and other options
			// Each flag has a corresponding destination variable to store the value

			&cli.StringFlag{
				Name:        "username",
				Value:       "",
				Aliases:     []string{"u"},
				Destination: &username,
				Usage:       "Enter Username Wordlist",
			},
			&cli.StringFlag{
				Name:        "password",
				Value:       "",
				Aliases:     []string{"p"},
				Destination: &password,
				Usage:       "Enter Password Wordlist",
			},
			&cli.StringFlag{
				Name:        "combolist",
				Value:       "",
				Aliases:     []string{"c"},
				Destination: &combolist,
				Usage:       "Enter Combo Wordlist output",
			},
		},
		Action: func(cCtx *cli.Context) error {
			// Switch case to handle different scenarios based on provided options
			switch {
			case cCtx.String("username") == "":
				fmt.Println(color.Colorize(color.Red, "[-] Please Enter Username Wordlist Address with -u"))
			case cCtx.String("password") == "":
				fmt.Println(color.Colorize(color.Red, "[-] Please Enter Password Wordlist Address with -u"))
			case cCtx.String("combolist") == "":
				fmt.Println(color.Colorize(color.Red, "[-] Please Enter Password Wordlist Address with -u"))
			}
			return nil
		},
	}
	// Run the CLI app
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
	usernamedic, err := os.OpenFile(username, os.O_RDONLY, 0600)
	errorpars(err)
	defer func(usernamedic *os.File) {
		err := usernamedic.Close()
		errorpars(err)
	}(usernamedic)

	passwordsdic, err := os.OpenFile(password, os.O_RDONLY, 0600)
	errorpars(err)
	defer func(passwordsdic *os.File) {
		err := passwordsdic.Close()
		errorpars(err)
	}(passwordsdic)

	linesuser := make([]string, 0)
	linespassw := make([]string, 0)
	var wg sync.WaitGroup
	doneUser := make(chan struct{})
	donePass := make(chan struct{})
	wg.Add(2)
	// Read content of wordlist files and generate combos in parallel
	go readInChunks(usernamedic, &linesuser, &wg, doneUser)
	go readInChunks(passwordsdic, &linespassw, &wg, donePass)
	wg.Wait()
	// Wait for files to finish reading
	<-doneUser
	<-donePass

	// Add jobs to the queue and process combos in parallel

	fmt.Println(len(linesuser), len(linespassw))
	combo := make(chan string, len(linesuser)*len(linespassw))

	// Create worker goroutines
	// Create worker goroutines
	for _, usern := range linesuser {
		for _, passw := range linespassw {
			wg.Add(1)
			go func(usern, passw string) {
				defer wg.Done()
				com := fmt.Sprintf("%s:%s\n", usern, passw)
				combo <- com
			}(usern, passw)
		}
	}

	// Close combo channel when all workers are done
	go func() {
		wg.Wait()
		close(combo)
	}()

	// Process combos and write results
	writeResults(combo, combolist)
}

// write result to file output
func writeResults(results <-chan string, outname string) {
	path := "output"
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}

	writeToFile(results, path+"/"+outname)
}

// write text output file
func writeTextFile(file *os.File, results <-chan string) {
	for co := range results {
		if _, err := file.WriteString(co); err != nil {
			log.Println(err)
		}
	}
}

// detect file format to save output file
func writeToFile(results <-chan string, filePath string) {
	file, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Println(err)
		}
	}(file)

	switch {
	case strings.HasSuffix(filePath, ".txt"):
		writeTextFile(file, results)
	default:
		log.Fatal("Unsupported file format just support .txt file")
	}
	elapsed := time.Since(start)
	fmt.Printf("Execution time: %s\n", elapsed)
}

//var counter int

// Function to read file in chunks
// Function to read file in chunks
func readInChunks(file *os.File, lines *[]string, wg *sync.WaitGroup, done chan<- struct{}) {
	defer func() {
		close(done)
		wg.Done()
	}()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		*lines = append(*lines, scanner.Text())
	}
}
