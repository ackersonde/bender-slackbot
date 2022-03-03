package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/ackersonde/bender-slackbot/commands"
	"github.com/go-co-op/gocron"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

var botID = os.Getenv("SLACK_BENDER_BOT_USERID")

func prepareScheduler() {
	s := gocron.NewScheduler(time.Now().Local().Location())

	s.Every(1).Day().At("08:04").Do(
		commands.VpnPiTunnelChecks, commands.VPNCountry)
	s.Every(1).Day().At("06:55").Do(commands.DisplayFirewallRules)
	//s.Every(1).Day().At("17:30").Do(commands.ShowBBGamesCron, "")

	ensureWiFiOffOvernight(s)

	s.StartAsync()
	// more examples: https://github.com/go-co-op/gocron/blob/master/README.md
}

func ensureWiFiOffOvernight(s *gocron.Scheduler) {
	s.Every(1).Day().At("00:00").Do(commands.WifiAction, "0")
	s.Every(1).Day().At("01:00").Do(commands.WifiAction, "0")
	s.Every(1).Day().At("02:00").Do(commands.WifiAction, "0")
	s.Every(1).Day().At("03:00").Do(commands.WifiAction, "0")
	s.Every(1).Day().At("04:00").Do(commands.WifiAction, "0")
}

func verifiedSlackMessage(w http.ResponseWriter, r *http.Request) []byte {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("[ERROR] Fail to read request body: %v", err)
		return nil
	}

	// Verify signing secret
	sv, err := slack.NewSecretsVerifier(r.Header, os.Getenv("SLACK_SIGNING_SECRET"))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		log.Printf("[ERROR] Fail to verify SigningSecret: %v", err)
		return nil
	}
	sv.Write(body)
	if err := sv.Ensure(); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		log.Printf("[ERROR] Fail to verify SigningSecret: %v", err)
		body = nil
	}

	return body
}

func parseSlackEvent(w http.ResponseWriter, r *http.Request) slackevents.EventsAPIEvent {
	msgBody := verifiedSlackMessage(w, r)
	eventsAPIEvent, e := slackevents.ParseEvent(json.RawMessage(msgBody),
		slackevents.OptionVerifyToken(&slackevents.TokenComparator{
			VerificationToken: os.Getenv("SLACK_VERIFICATION_TOKEN")}))

	if e != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("[ERROR] Fail to parseEvent: %v\n", e)
	}

	// when you change the Bender Bot app server URL
	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(msgBody), &r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.Header().Set("Content-Type", "text")
			w.Write([]byte(r.Challenge))
		}
	}

	return eventsAPIEvent
}

func processMessage(ev *slackevents.MessageEvent) {
	originalMessage := ev.Text

	if ev.User != "" && ev.User != botID && ev.User != "U2NQSPHHD" &&
		ev.SubType != "message_deleted" &&
		(strings.Contains(ev.Text, "<@"+botID+">") ||
			strings.HasPrefix(ev.Channel, "D") ||
			ev.Channel == commands.SlackReportChannel) {
		// strip out bot's name and spaces
		parsedMessage := strings.TrimSpace(strings.Replace(originalMessage, "<@"+botID+">", "", -1))
		r, n := utf8.DecodeRuneInString(parsedMessage)
		parsedMessage = string(unicode.ToLower(r)) + parsedMessage[n:]

		commands.Logger.Printf("%s(%s) asks '%v'\n", ev.Username, ev.User, parsedMessage)
		commands.CheckCommand(ev, parsedMessage)
	}
}

func main() {
	api := slack.New(
		os.Getenv("CTX_SLACK_NEW_API_TOKEN"),
		slack.OptionDebug(false),
		slack.OptionLog(commands.Logger),
	)
	commands.SetAPI(api)
	go prepareScheduler() // spawn cron scheduler jobs

	http.HandleFunc("/"+os.Getenv("SLACK_EVENTSAPI_ENDPOINT"),
		func(w http.ResponseWriter, r *http.Request) {
			eventsAPIEvent := parseSlackEvent(w, r)
			if eventsAPIEvent.Type == slackevents.CallbackEvent {
				innerEvent := eventsAPIEvent.InnerEvent
				switch ev := innerEvent.Data.(type) {
				case *slackevents.MessageEvent:
					go processMessage(ev)
					// HTTP 202 -> we heard and are working on an async response
					w.WriteHeader(http.StatusAccepted)
				}
			}
		})

	fmt.Println("[INFO] Bender listening")
	http.ListenAndServe(":3000", nil)
}
