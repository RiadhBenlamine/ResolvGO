package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/urfave/cli/v2"
)

type Config struct {
	filePath string
	threads  int
	output   string
}

func resolve(domain string, result chan<- string, client *dns.Client) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	msg.RecursionDesired = true

	response, _, err := client.Exchange(msg, "1.1.1.1:53")
	if err != nil || response.Rcode != dns.RcodeSuccess {
		return
	}

	var ips []string

	for _, answer := range response.Answer {
		if aRecord, ok := answer.(*dns.A); ok {
			ips = append(ips, aRecord.A.String())
		}
	}

	if len(ips) > 0 {
		result <- domain + ":" + strings.Join(ips, ",")
	}
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
func worker(jobs <-chan string, result chan<- string, wg *sync.WaitGroup, client *dns.Client) {
	defer wg.Done()
	for domain := range jobs {
		resolve(domain, result, client)
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
				Value:       3,
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
	client := &dns.Client{
		Net:         "udp",
		DialTimeout: 2 * time.Second,
		ReadTimeout: 2 * time.Second,
	}

	for range maxWorkers {
		wg.Add(1)
		go worker(jobs, result, &wg, client)
	}
	go func() {
		for _, domain := range domains {
			jobs <- domain
		}
		close(jobs)
	}()
	go func() {
		wg.Wait()
		close(result)
	}()
	var toWrite []string
	for domain := range result {
		fmt.Println(domain)
		toWrite = append(toWrite, domain)
	}

	// TODO: Write output to a file
	if config.output == "" {
		log.Fatal("Output file must not be empty.")
		return
	}
	file, err := os.Create(config.output)
	if err != nil {
		log.Fatal(err)
	}
	writer := bufio.NewWriter(file)
	for _, domain := range toWrite {
		_, err := writer.WriteString(domain + "\n")
		if err != nil {
			return
		}
	}
	if err := writer.Flush(); err != nil {
		log.Fatal(err)
	}
	if err := file.Close(); err != nil {
		log.Fatal(err)
	}

}
