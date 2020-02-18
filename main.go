package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"gopkg.in/alecthomas/kingpin.v2"
	"io"
	"net/url"
	"os"
	"strings"

	"log"
	"net/http"

	"sync"
	"time"
)

type (
	ConfHttp struct {
		MaxIdleConnts      int
		IdleConnTimeout    time.Duration
		DisableCompression bool
	}

	ImageOi struct {
		URLImage string
		URL      string
	}
)

var (
	fileName string
	page     int
	keyword  = kingpin.Flag("keyword", "Name of keyword.").Required().String()
)

// NewConfig
func NewConfig(cfg ConfHttp) *ConfHttp {
	return &ConfHttp{
		MaxIdleConnts:      cfg.MaxIdleConnts,
		IdleConnTimeout:    cfg.IdleConnTimeout,
		DisableCompression: cfg.DisableCompression,
	}
}

// NewHttp
func (c ConfHttp) NewHttp() *http.Client {
	tr := &http.Transport{
		MaxIdleConns:       c.MaxIdleConnts,
		IdleConnTimeout:    c.IdleConnTimeout,
		DisableCompression: c.DisableCompression,
	}
	return &http.Client{Transport: tr}
}

// buildSearchImage
func buildSearchImage(c *http.Client, cImg chan<- ImageOi, keyword string, wg *sync.WaitGroup, page int) {
	defer wg.Done()

	url := fmt.Sprintf("https://www.google.com/search?q=%v&safe=strict&espv=2&biw=1366&bih=667&sout=1&tbm=isch&sxsrf=ACYBGNTT9AO4BK5R3-6sEMjkrxG-9ZSHjA:1581706285375&ei=LexGXuCNFOiZ4-EPw7aLuAc&start=%d&sa=N", keyword, page)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36")
	res, err := c.Do(req)

	if err != nil {
		log.Fatal(err)
	}

	document, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal("Error loading HTTP response body. ", err)
	}
	document.Find("table td").Each(func(index int, element *goquery.Selection) {

		imgSrc, _ := element.Find("img").Attr("src")
		URL, _ := element.Find("a").Attr("href")

		if strings.Contains(URL, "/url?q=https") {
			cImg <- ImageOi{
				URLImage: imgSrc,
				URL:      URL,
			}
		}
	})
}

// fetchImage
func getURLPage(cImg <-chan ImageOi, c *http.Client, urlImage chan string) {

	for s := range cImg {
		u, err := url.Parse(s.URL)
		if err != nil {
			panic(err)
		}
		newURL := u.Query()["q"][0]

		req, _ := http.NewRequest("GET", newURL, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36")
		res, err := c.Do(req)

		if err != nil {
			log.Fatal(err)
		}

		document, err := goquery.NewDocumentFromReader(res.Body)

		document.Find("meta").Each(func(i int, s *goquery.Selection) {
			var imageURL string
			if name, _ := s.Attr("property"); name == "og:image" {
				imageURL, _ = s.Attr("content")
				NewPageURLImage, _ := url.Parse(imageURL)

				if NewPageURLImage.Scheme != "" {
					imageURL, _ = s.Attr("content")
					urlImage <- imageURL
				}
			}

		})
	}
	close(urlImage)

}

// fetchImage
func fetchImage(url string, client *http.Client, keyword string, fileName int) {

	res, err := client.Get(url)

	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		os.MkdirAll(keyword, os.ModePerm)
		extensi := strings.Split(url, ".")
		newEkstenstion := extensi[len(extensi)-1]

		fileName := fmt.Sprintf("%v/%v.%v", keyword, fileName, newEkstenstion)
		file, err := os.Create(fileName)

		if err != nil {
			log.Fatal(err)
		}

		defer file.Close()
		_, err = io.Copy(file, res.Body)

		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("*Successfuly download image to Path : %v \n", fileName)
	}

}

func main() {

	kingpin.Parse()

	*keyword = strings.Replace(*keyword, " ", "+", -1)

	c := ConfHttp{
		MaxIdleConnts:      2,
		IdleConnTimeout:    90 * time.Second,
		DisableCompression: true,
	}

	cImg := make(chan ImageOi)
	cURLImage := make(chan string)
	var wg sync.WaitGroup

	cfg := NewConfig(c)
	client := cfg.NewHttp()
	go func() {
		for i := 0; i < 10; i++ {
			if i == 0 {
				page = 0
			} else {
				page = i * 20
			}

			wg.Add(1)
			buildSearchImage(client, cImg, *keyword, &wg, page)
		}
	}()

	go func() {
		wg.Add(1)
		getURLPage(cImg, client, cURLImage)
		wg.Done()
	}()

	wg.Wait()

	var noFilename int
	for imageURLGetIN := range cURLImage {
		fmt.Printf("Proccess download image from URL : %v \n", imageURLGetIN)
		fetchImage(imageURLGetIN, client, *keyword, noFilename)
		noFilename++
	}
}
