package main

import (
	"encoding/json"
	"fmt"
	"io"
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

	s.Every(1).Day().At("05:00").Do(commands.CheckDiskSpace, false)
	s.Every(1).Day().At("05:05").Do(commands.CheckBackups, false)
	s.Every(1).Day().At("06:55").Do(commands.DisplayFirewallRules)
	s.Every(1).Day().At("08:04").Do(commands.VpnPiTunnelChecks)
	s.Every(1).Day().At("17:30").Do(commands.ShowBBGamesCron, "")

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
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		commands.Logger.Printf("[ERROR] Fail to read request body: %v", err)
		return nil
	}

	// Verify signing secret
	sv, err := slack.NewSecretsVerifier(r.Header, os.Getenv("SLACK_SIGNING_SECRET"))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		commands.Logger.Printf("[ERROR] Fail to verify SigningSecret: %v", err)
		return nil
	}
	sv.Write(body)
	if err := sv.Ensure(); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		commands.Logger.Printf("[ERROR] Fail to verify SigningSecret: %v", err)
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
		commands.Logger.Printf("[ERROR] Fail to parseEvent: %v\n", e)
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

func isValidMessageEvent(ev *slackevents.MessageEvent) bool {
	return ev.User != "" && ev.User != botID && // don't talk to yourself or ghosts!
		ev.SubType != "message_deleted" && // don't respond to msgs being deleted
		(strings.Contains(ev.Text, "<@"+botID+">") || // the msg is directed @ you in a channel you're in
			strings.HasPrefix(ev.Channel, "D") || // or the msg is sent to you directly
			ev.Channel == commands.SlackReportChannel) // or the msg is sent to #bender_rodriguez
}

func processMessage(ev *slackevents.MessageEvent, api *slack.Client) {
	originalMessage := ev.Text

	if isValidMessageEvent(ev) {
		// strip out bot's name and spaces
		parsedMessage := strings.TrimSpace(strings.Replace(originalMessage, "<@"+botID+">", "", -1))
		r, n := utf8.DecodeRuneInString(parsedMessage)
		parsedMessage = string(unicode.ToLower(r)) + parsedMessage[n:]

		user, err := api.GetUserInfo(ev.User)
		if err != nil {
			commands.Logger.Printf("NO user info for %s: %s", ev.User, err.Error())
		} else {
			commands.Logger.Printf("%s(%s) asks '%v'", user.Name, ev.User, parsedMessage)
		}
		commands.CheckCommand(ev, user, parsedMessage)
	}
}

func main() {
	api := slack.New(
		os.Getenv("SLACK_NEW_API_TOKEN"),
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
					go processMessage(ev, api)
					// HTTP 202 -> we heard and are working on an async response
					w.WriteHeader(http.StatusAccepted)
				}
			}
		})

	fmt.Println("[INFO] Bender listening")
	http.ListenAndServe(":3000", nil)
}
