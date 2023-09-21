package lea

import (
	"fmt"
	"github.com/gen2brain/beeep"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/nenad/amt/internal/config"
	"github.com/nenad/amt/internal/telegram"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	domain             = "https://otv.verwalt-berlin.de"
	homeURL            = domain + "/ams/TerminBuchen"
	bookAppointmentURL = domain + "/ams/TerminBuchen/wizardng?sprachauswahl=en"
)

type LeaScenario struct {
	TelegramClient *telegram.Client
}

func (s LeaScenario) Run(lea config.Lea) error {
	u := launcher.MustResolveURL("")
	// Trace shows verbose debug information for each action executed
	// SlowMotion is a debug related function that waits 2 seconds between
	// each action, making it easier to inspect what your code is doing.
	browser := rod.New().ControlURL(u).Trace(true).MustConnect().SlowMotion(time.Millisecond * 200)
	attempt := 0
	for {
		l := log.New(os.Stdout, fmt.Sprintf("lea-%d ", attempt), log.LstdFlags)
		if err := browser.SetCookies(nil); err != nil {
			panic("Could not clear cookies: " + err.Error())
		}

		page := browser.MustPage(homeURL).MustWaitDOMStable()

		//go func() {
		//	for {
		//		info := page.MustInfo()
		//		if strings.Contains(info.URL, "Error") {
		//			l.Println("Error page detected, reloading...")
		//			page.MustReload()
		//		}
		//	}
		//}()
		//
		//select {
		//case <-closeChan:
		//
		//}

		// We click the EN button on the main page
		page.MustElement("div.lang-link > a").MustClick()

		// Click the "Book Appointment" button
		page.MustElementR("a", "Book Appointment").MustClick()

		// Check the declaration checkbox
		page.MustElement("#xi-cb-1").MustClick()

		// Click the "Next" button
		page.MustElement(`#applicationForm\:managedForm\:proceed`).MustClick()

		// Select the citizenship
		el, err := page.Element("#xi-sel-400")
		if err != nil {
			l.Println("Could not find citizenship select: %s", err)
			continue
		}
		time.Sleep(time.Second * 1)
		el.MustSelect(lea.Citizenship)
		time.Sleep(time.Second * 2)
		// TODO - add a check for the number of applicants
		page.MustElement("#xi-sel-422").MustSelect("one person")
		// TODO - add a check for Live in Berlin
		page.MustElement("#xi-sel-427").MustSelect("no")

		// Click the main reason
		page.MustElementR("p", lea.MainReason).MustClick()
		// Click the category
		page.MustElementR("p", lea.Category).MustClick()
		// Click the subcategory
		page.MustElementX(fmt.Sprintf("//*[@class=\"level2-content\"]//label[normalize-space()=\"%s\"]", lea.Subcategory)).MustClick()

		success := false
	outer:
		for {
			remainingTime := s.getRemainingTime(page)

			if remainingTime < time.Minute {
				break
			}
			l.Println("Remaining time: ", remainingTime.String())

			activeTab := page.MustElement("li.antcl_active > span").MustText()
			if activeTab != "Service selection" && activeTab != "Servicewahl" {
				success = true
				break
			}
			l.Println("Active tab: ", activeTab)

			// Click the "Next" button
			page = page.MustWaitLoad()
			for {
				remainingTime := s.getRemainingTime(page)
				// Refresh the page if we don't have time
				if remainingTime < time.Minute {
					break outer
				}

				// Refresh the page if session expired
				info := page.MustInfo()
				if strings.Contains(info.URL, "logout") {
					break outer
				}

				el, err := page.Timeout(time.Second * 3).Element(`#applicationForm\:managedForm\:proceed`)
				if err != nil {
					l.Println("not found, error: ", err.Error())
					time.Sleep(time.Second)
					continue
				}

				if err := el.CancelTimeout().Timeout(time.Second*60).Click(proto.InputMouseButtonLeft, 1); err != nil {
					l.Println("click error found, error: ", err.Error())
					time.Sleep(time.Second)
					continue
				}

				break
			}
		}

		// We are on the calendar view, great success! But the war is still not over.
		// We need to have appointments in the select.
		//appointmentSelector := page.MustElement("#xi-sel-3")
		//options := appointmentSelector.MustElements("option")
		//success = len(options) > 2
		//Select the first available appointment
		//appointmentSelector.MustSelect(options[2].MustText())

		if success {
			info := page.MustInfo()
			path := "/tmp/screenshot-browser.png"
			_ = page.MustScreenshotFullPage(path)

			err := beeep.Notify("Appointment ready!", "", "")
			if err != nil {
				l.Println("Could not send notification: ", err)
			}

			if err := s.TelegramClient.Send(fmt.Sprintf("Appointment ready: %s", info.URL), path); err != nil {
				l.Println("Could not send telegram notification: ", err)
			}
			return nil
		}
	}

	return nil
}

func (s LeaScenario) getRemainingTime(page *rod.Page) time.Duration {
	// Get remaining time
	el, err := page.Timeout(time.Second * 30).Element("div.bar")
	if err != nil {
		return time.Second * 1
	}
	timeRemainingText := ""
	for i := 0; i < 10; i++ {
		if el.MustText() == "" {
			time.Sleep(time.Second * 1)
			continue
		}
		timeRemainingText = el.MustText()
	}

	parts := strings.Split(timeRemainingText, ":")
	if len(parts) != 2 {
		return time.Second * 1
	}
	mins, _ := strconv.Atoi(parts[0])
	secs, _ := strconv.Atoi(parts[1])
	remainingTime := time.Duration(mins)*time.Minute + time.Duration(secs)*time.Second
	return remainingTime
}
