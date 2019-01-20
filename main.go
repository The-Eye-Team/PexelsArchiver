package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/labstack/gommon/color"
)

var arguments = struct {
	Output      string
	Concurrency int
	RandomUA    bool
	Verbose     bool
	StartID     int
	StopID      int
}{}

var client = http.Client{}

var checkPre = color.Yellow("[") + color.Green("✓") + color.Yellow("]")
var tildPre = color.Yellow("[") + color.Green("~") + color.Yellow("]")
var crossPre = color.Yellow("[") + color.Red("✗") + color.Yellow("]")

func init() {
	// Disable HTTP/2: Empty TLSNextProto map
	client.Transport = http.DefaultTransport
	client.Transport.(*http.Transport).TLSNextProto =
		make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
}

func downloadPicture(link, uploaderName, index string) (err error) {
	// Replace slash
	// uploaderName = strings.Replace(name, "/", "_", -1)

	// Check if file exist
	if _, err := os.Stat(arguments.Output + "/" + uploaderName + "/" + index + ".jpg"); !os.IsNotExist(err) {
		fmt.Println(checkPre +
			color.Yellow("[") +
			color.Green(index) +
			color.Yellow("]") +
			color.Green(" Already downloaded: ") +
			color.Yellow(uploaderName+" - "+index))
		return err
	}

	// Create output dir
	os.MkdirAll(arguments.Output+"/"+uploaderName+"/", os.ModePerm)

	// Fetch the data from the URL
	resp, err := client.Get(link)
	if err != nil {
		fmt.Println(crossPre+color.Red(" Unable to download the file:"), color.Yellow(err))
		text := []byte(link + "\n")
		err = ioutil.WriteFile("./error.txt", text, 0644)
		if err != nil {
			return err
		}
		return nil
	}
	defer resp.Body.Close()

	// Create ebook's file
	pictureFile, err := os.Create(arguments.Output + "/" + uploaderName + "/" + index + ".jpg")
	if err != nil {
		log.Println(crossPre+color.Red(" Unable to create the file:"), color.Yellow(err))
		text := []byte(link + "\n")
		err = ioutil.WriteFile("./error.txt", text, 0644)
		if err != nil {
			return err
		}
		return err
	}
	defer pictureFile.Close()

	// Write the data to the file
	_, err = io.Copy(pictureFile, resp.Body)
	if err != nil {
		log.Println(crossPre+color.Red(" Unable to write to the file:"), color.Yellow(err))
		text := []byte(link + "\n")
		err = ioutil.WriteFile("./error.txt", text, 0644)
		if err != nil {
			return err
		}
		return err
	}

	fmt.Println(checkPre +
		color.Yellow("[") +
		color.Green(index) +
		color.Yellow("]") +
		color.Green(" Downloaded: ") +
		color.Yellow(uploaderName+" - "+index))

	return nil
}

func scrapePictureLink(url string, index int, worker *sync.WaitGroup) {
	defer worker.Done()
	var uploaderName string
	var pictureURL string

	// Create collector
	c := colly.NewCollector()

	// Randomize user agent on every request
	if arguments.RandomUA == true {
		extensions.RandomUserAgent(c)
	}

	// Get uploader's name
	c.OnHTML("h3.mini-profile__name", func(e *colly.HTMLElement) {
		uploaderName = e.ChildText("a")
	})

	// Get download link
	c.OnHTML("ul.select-list", func(e *colly.HTMLElement) {
		pictureURL = e.ChildAttr("input", "data-alt-url")
	})

	c.Visit(url)

	if len(pictureURL) > 0 {
		if len(uploaderName) < 1 {
			uploaderName = "Various"
		}
		err := downloadPicture(pictureURL, uploaderName, strconv.Itoa(index))
		if err != nil {
			return
		}
	}
}

func main() {
	var worker sync.WaitGroup
	var count int

	// Parse arguments and fill the arguments structure
	parseArgs(os.Args)

	// Set maxIdleConnsPerHost
	client.Transport.(*http.Transport).MaxIdleConnsPerHost = arguments.Concurrency

	for index := arguments.StartID; index <= arguments.StopID; index++ {
		worker.Add(1)
		count++
		url := "https://www.pexels.com/photo/" + strconv.Itoa(index)
		go scrapePictureLink(url, index, &worker)
		if count == arguments.Concurrency {
			worker.Wait()
			count = 0
		}
	}

	worker.Wait()
}
