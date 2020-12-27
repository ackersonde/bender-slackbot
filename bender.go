package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ackersonde/bender-slackbot/commands"
	"github.com/jasonlvhit/gocron"
	"github.com/nlopes/slack/slackevents"
	"github.com/slack-go/slack"
)

var botID = os.Getenv("SLACK_BENDER_BOT_USERID")

func prepareScheduler() {
	gocron.Every(1).Day().At("08:04").Do(
		commands.ChangeToFastestVPNServer, commands.VPNCountry)
	gocron.Every(1).Friday().At("09:05").Do(commands.CheckMediaDiskSpace, "")
	gocron.Every(1).Friday().At("09:05").Do(commands.CheckServerDiskSpace, "")
	//gocron.Every(1).Day().At("17:30").Do(commands.ShowBBGames, "")
	<-gocron.Start()

	// more examples: https://github.com/jasonlvhit/gocron/blob/master/example/example.go#L19
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
