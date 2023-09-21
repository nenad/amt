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
	domain  = "https://otv.verwalt-berlin.de"
	homeURL = domain + "/ams/TerminBuchen"

	giveUpBefore = time.Second * 70
)

type LeaScenario struct {
	TelegramClient *telegram.Client
}

func (s LeaScenario) Run(lea config.Lea) error {
	u := launcher.MustResolveURL("")
	// Trace shows verbose debug information for each action executed
	// SlowMotion is a debug related function that waits 2 seconds between
	// each action, making it easier to inspect what your code is doing.
	browser := rod.New().
		ControlURL(u).
		Trace(true).
		MustConnect()
	attempt := 0
	page := browser.MustPage("https://google.com")
	l := log.New(os.Stdout, fmt.Sprintf("lea-%d ", attempt), log.LstdFlags)
	go page.EachEvent(func(e *proto.PageJavascriptDialogOpening) {
		restore := page.EnableDomain(&proto.PageEnable{})
		l.Println("Dialog opened: ", e.Message, e.DefaultPrompt)
		defer restore()
		_ = proto.PageHandleJavaScriptDialog{
			Accept:     true,
			PromptText: "",
		}.Call(page)
	})()

	for {
		l = log.New(os.Stdout, fmt.Sprintf("lea-%d ", attempt), log.LstdFlags)
		attempt++
		if err := browser.SetCookies(nil); err != nil {
			panic("Could not clear cookies: " + err.Error())
		}

		retry := true
		success := false

		err := rod.Try(func() {
			page = page.Timeout(time.Second * 90).MustNavigate(homeURL).MustWaitDOMStable()
			// We click the EN button on the main page
			page.MustElement("div.lang-link > a").MustClick()

			// Click the "Book Appointment" button
			page.MustElementR("a", "Book Appointment").MustClick()

			// Check the declaration checkbox
			page.MustElement("#xi-cb-1").MustClick()

			// Click the "Next" button
			page.MustElement(`#applicationForm\:managedForm\:proceed`).MustClick()

			page = page.CancelTimeout()

			// Select the citizenship
			el, err := page.Timeout(time.Minute).Element("#xi-sel-400")
			if err != nil {
				l.Println("Could not find citizenship select: %s", err)
				return
			}
			time.Sleep(time.Second * 1)
			el.MustSelect(lea.Citizenship)
			page.CancelTimeout()
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

			for {
				remainingTime := s.getRemainingTime(page)
				if remainingTime < giveUpBefore {
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
				// TODO Setup a Go routine to navigate if 2 mins left
				page = page.MustWaitLoad()
				for {
					remainingTime = s.getRemainingTime(page)
					// Refresh the page if we don't have time
					if remainingTime < time.Minute {
						return
					}

					// Refresh the page if session expired
					info := page.MustInfo()
					if strings.Contains(info.URL, "logout") {
						return
					}

					el, err := page.Timeout(time.Second * 3).Element(`#applicationForm\:managedForm\:proceed`)
					if err != nil {
						l.Println("not found, error: ", err.Error())
						continue
					}

					if err := el.CancelTimeout().Timeout(time.Second*60).Click(proto.InputMouseButtonLeft, 1); err != nil {
						l.Println("click error found, error: ", err.Error())
						continue
					}

					break
				}
			}

			if !success {
				return
			}

			// We are on the calendar view, great success! But the war is still not over.
			// We need to have appointments in the select.
			//appointmentSelector := page.MustElement("#xi-sel-3")
			//options := appointmentSelector.MustElements("option")
			//if options.Last().MustText() == "" {
			//	No appointments available, rerun the loop
			//return
			//}
			//Select the first available appointment
			//appointmentSelector.MustSelect(options[2].MustText())

			msgBox, err := page.Timeout(time.Second * 10).Element("#errorMessage")
			if err == nil {
				l.Println("Error message not found, good: ", err)
				if msgBox != nil && msgBox.MustText() != "" {
					l.Println("Error in message box found, retrying: ", el.MustText())
					success = false
					retry = true
				}
			}

			info := page.MustInfo()
			retry = false

			err = beeep.Notify("Appointment ready!", "", "")
			if err != nil {
				l.Println("Could not send notification: ", err)
			}

			err = rod.Try(func() {
				//path := "/tmp/screenshot-browser.png"
				//_ = page.MustScreenshotFullPage(path)
				if err := s.TelegramClient.Send(fmt.Sprintf("Appointment ready: %s", info.URL), ""); err != nil {
					l.Println("Could not send telegram notification: ", err)
				}
			})
			if err != nil {
				l.Println("error while taking screenshot: ", err)
			}
		})

		if err != nil {
			l.Println("Error in loop: ", err)
			retry = true
		}

		if retry && !success {
			continue
		} else {
			break
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
