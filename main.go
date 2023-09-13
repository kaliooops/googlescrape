package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var googleDomains = map[string]string{
	"com": "https://www.google.com/search?q=",
	"ac":  "https://www.google.ac/search?q=",
}

type SearchResult struct {
	ResultRank  int
	ResultURL   string
	ResultTitle string
	ResultDesc  string
}

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_5_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
}

func randomUserAgent() string {
	rand.Seed(time.Now().Unix())
	randNum := rand.Int() % len(userAgents)
	return userAgents[randNum]
}

func buildGoogleUrls(searchTerm, countryCode string, pages, count int) ([]string, error) {
	toScrape := []string{}
	searchTerm = strings.Trim(searchTerm, " ")
	searchTerm = strings.Replace(searchTerm, " ", "+", -1)
	if googleBase, found := googleDomains[countryCode]; found {
		for i := 0; i < pages; i++ {
			start := i * count
			scrapeURL := fmt.Sprintf("%s%s&num=%d&start=%d&filter=0", googleBase, searchTerm, count, start)
			toScrape = append(toScrape, scrapeURL)
			fmt.Println(scrapeURL)
		}
	} else {
		err := fmt.Errorf("country (%s) is currently not supported", countryCode)
		return nil, err
	}
	return toScrape, nil

}

func googleResultParsing(response *http.Response, rank int) ([]SearchResult, error) {
	doc, err := goquery.NewDocumentFromResponse(response)

	if err != nil {
		return nil, err
	}

	results := []SearchResult{}
	sel := doc.Find("div.g")
	rank++
	for i := range sel.Nodes {
		item := sel.Eq(i)
		linkTag := item.Find("a")
		link, _ := linkTag.Attr("href")
		titleTag := item.Find("h3.r")
		descTag := item.Find("span.st")
		desc := descTag.Text()
		title := titleTag.Text()
		link = strings.Trim(link, " ")

		if link != "" && link != "#" && !strings.HasPrefix(link, "/") && strings.Contains(link, "scripts/setup.php") && !strings.Contains(link, "%") && !strings.Contains(link, "?") && !strings.Contains(link, "=") && !strings.Contains(link, "1") && !strings.Contains(link, "2") && !strings.Contains(link, "3") && !strings.Contains(link, "4") && !strings.Contains(link, "5") {
			result := SearchResult{
				rank,
				link,
				title,
				desc,
			}
			fmt.Println(link)
			WriteFile("found.txt", link+"\n")
			results = append(results, result)
			rank++
		}
	}
	return results, err

}

func getScrapeClient(proxyString interface{}) *http.Client {

	switch v := proxyString.(type) {

	case string:
		proxyUrl, _ := url.Parse(v)
		return &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}
	default:
		return &http.Client{}
	}
}

func GoogleScrape(searchTerm, countryCode string, proxyString interface{}, pages, count, backoff int) ([]SearchResult, error) {
	results := []SearchResult{}
	resultCounter := 0
	googlePages, err := buildGoogleUrls(searchTerm, countryCode, pages, count)
	if err != nil {
		return nil, err
	}
	for _, page := range googlePages {
		res, err := scrapeClientRequest(page, proxyString)
		if err != nil {
			return nil, err
		}
		data, err := googleResultParsing(res, resultCounter)
		if err != nil {
			return nil, err
		}
		resultCounter += len(data)
		for _, result := range data {
			results = append(results, result)
		}
		time.Sleep(time.Duration(backoff) * time.Second)
	}
	return results, nil
}

func scrapeClientRequest(searchURL string, proxyString interface{}) (*http.Response, error) {
	baseClient := getScrapeClient(proxyString)
	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Set("User-Agent", randomUserAgent())

	res, err := baseClient.Do(req)
	if res.StatusCode != 200 {
		err := fmt.Errorf("scraper received a non-200 status code suggesting a ban")
		return nil, err
	}

	if err != nil {
		return nil, err
	}
	return res, nil
}

func main() {
	list := string(os.Args[1])
	results, _ := strconv.Atoi(string(os.Args[2]))
	data := read_file(list)
	for _, query := range data {
		res, err := GoogleScrape("inurl:"+query, "com", nil, 1, results, 10)
		if err == nil {
			fmt.Println("Found: ", len(res), " On Query: "+query)
		}
	}
	fmt.Println("Done!")
}

func read_file(filePath string) []string {
	readFile, err := os.Open(filePath)
	if err != nil {
		fmt.Println("eroare")
	}

	if err != nil {
		fmt.Println(err)
	}
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	var fileLines []string

	for fileScanner.Scan() {
		fileLines = append(fileLines, fileScanner.Text())
	}

	readFile.Close()
	return fileLines
}

func WriteFile(filename string, content string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("eroare")
	}
	if _, err := f.WriteString(content); err != nil {
		fmt.Println("eroare")
	}
	if err := f.Close(); err != nil {
		fmt.Println("eroare")
	}
}
