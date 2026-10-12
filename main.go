package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/urfave/cli/v2"
)

type Config struct {
	filePath string
	threads  int
	output   string
}

func resolve(domain string, result chan<- string) {
	domain = strings.ToLower(domain)
	result <- domain
	// I don't know what lib I should use
}
func getDomains(path string) []string {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var domains []string
	for scanner.Scan() {
		domains = append(domains, scanner.Text())
	}
	return domains
}
func worker(jobs <-chan string, result chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	for domain := range jobs {
		resolve(domain, result)
	}
}
func main() {
	config := &Config{}
	cmd := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "list",
				Aliases:     []string{"l"},
				Destination: &config.filePath,
				Required:    true,
			},
			&cli.IntFlag{
				Name:        "threads",
				Aliases:     []string{"t"},
				Destination: &config.threads,
			},
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Destination: &config.output,
			},
		},
		Action: func(c *cli.Context) error {
			return nil
		},
	}
	if err := cmd.Run(os.Args); err != nil {
		log.Fatal(err)
	}
	maxWorkers := config.threads
	if maxWorkers < 1 {
		fmt.Println("You must specify at least one worker to run.")
		return
	}
	domainsPath := config.filePath
	if domainsPath == "" {
		fmt.Println("You must provide a domains file path")
		return
	}
	domains := getDomains(domainsPath)
	result := make(chan string)
	jobs := make(chan string)
	var wg sync.WaitGroup

	for range maxWorkers {
		wg.Add(1)
		go worker(jobs, result, &wg)
	}
	go func() {
		for _, domain := range domains {
			jobs <- domain
		}
	}()
	go func() {
		wg.Wait()
		close(result)
	}()

}
