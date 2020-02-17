package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const concurrency = 5
const wordToCount = "go"

type processedLink struct {
	address *url.URL
	count   int
	error   string
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	wg := new(sync.WaitGroup)
	semaphore := make(chan bool, concurrency)
	processed := make(chan processedLink)
	finished := make(chan bool)

	go processResults(processed, finished)

	for scanner.Scan() {
		wg.Add(1)
		semaphore <- true

		link := scanner.Text()
		go worker(link, semaphore, processed, wg)
	}

	if err := scanner.Err(); err != nil {
		log.Println(err)
	}

	wg.Wait()

	close(processed)

	<-finished
}

func worker(link string, semaphore chan bool, processed chan processedLink, wg *sync.WaitGroup) {
	defer wg.Done()

	defer func() {
		<-semaphore
	}()

	parsedLink, err := url.ParseRequestURI(link)
	if err != nil {
		processed <- processedLink{
			address: parsedLink,
			error:   err.Error(),
		}
		return
	}

	processed <- processedLink{
		address: parsedLink,
		count:   countWordAtLink(parsedLink),
	}
}

func countWordAtLink(link *url.URL) int {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	response, err := client.Get(link.String())
	if err != nil {
		log.Fatal(err)
	}

	defer response.Body.Close()

	dataInBytes, err := ioutil.ReadAll(response.Body)
	pageContent := string(dataInBytes)

	return strings.Count(pageContent, wordToCount)
}

func processResults(processed chan processedLink, finished chan bool) {
	total := 0

	for link := range processed {
		if link.error != "" {
			fmt.Printf("Error for %s: %s \n", link.address, link.error)
			continue
		}

		fmt.Printf("Count for %s: %d \n", link.address, link.count)

		total += link.count
	}

	fmt.Println("Total: ", total)

	finished <- true

	return
}
