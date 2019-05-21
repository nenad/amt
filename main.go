package main

import (
	"fmt"
	"net/http/cookiejar"
	"net/url"

	"github.com/PuerkitoBio/goquery"

	"log"
	"net/http"
	"time"
)

const host = "https://service.berlin.de"
const fahrerlaubnissbehörde = "https://service.berlin.de/terminvereinbarung/termin/tag.php?termin=1&dienstleister=121646&anliegen[]=327537&herkunft=1"

func main() {
	cookies, _ := cookiejar.New(nil)
	client := &http.Client{
		Timeout: time.Second * 5,
		Jar:     cookies,
	}

	req, err := http.NewRequest("GET", fahrerlaubnissbehörde, nil)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

    for {
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		free := getBookableAppointments(doc)
		if len(free) != 0 {
			fmt.Println("FOUND APPOINTMENTS!")
            for _, a := range free {
                fmt.Println(a.Text())
			}
            break
		}

		next := getNextPage(doc)
		if next == "" {
			fmt.Println("No more pages to check")
            break
		}

        nextURL, _ := url.Parse(next)
		req.URL = nextURL
		resp, err = client.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Checked " + next)
	}


}

func getBookableAppointments(doc *goquery.Document) []*goquery.Selection {
	tables := doc.Find(`.calendar-month-table`)
    var free []*goquery.Selection
	tables.Each(func(i int, selection *goquery.Selection) {
		appointments := selection.Find("td.buchbar")
		if appointments.Length() != 0 {
			free = append(free, appointments)
		}
	})

	return free
}

func getNextPage(doc *goquery.Document) string {
	if attr, exists := doc.Find("th.next").Find("a").Attr("href"); exists {
        return host + attr
	}

	return ""
}
