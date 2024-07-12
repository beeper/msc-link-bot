package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	globallog "github.com/rs/zerolog/log"
	"go.mau.fi/util/dbutil"
	"gopkg.in/yaml.v3"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/event"
)

var MSC_REGEX *regexp.Regexp = regexp.MustCompile(`\b(?:MSC|msc)(\d+)\b`)

func main() {
	// Arg parsing
	configPath := flag.String("config", "./config.yaml", "config file location")
	flag.Parse()

	// Load configuration
	globallog.Info().Str("config_path", *configPath).Msg("Reading config")
	configYaml, err := os.ReadFile(*configPath)
	if err != nil {
		globallog.Fatal().Err(err).Str("config_path", *configPath).Msg("Failed reading the config")
	}

	var config Configuration
	err = yaml.Unmarshal(configYaml, &config)
	if err != nil {
		globallog.Fatal().Err(err).Msg("Failed to parse configuration YAML")
	}

	// Setup logging
	log, err := config.Logging.Compile()
	if err != nil {
		globallog.Fatal().Err(err).Msg("Failed to compile logging configuration")
	}

	// Open the database
	db, err := dbutil.NewFromConfig("msclinkbot", config.Database, dbutil.ZeroLogger(*log))
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't open database")
	}

	// Log In
	client, err := mautrix.NewClient(config.Homeserver, "", "")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create matrix client")
	}
	client.Log = *log
	cryptoHelper, err := cryptohelper.NewCryptoHelper(client, []byte("xyz.hnitbjorg.msc_link_bot"), db)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create crypto helper")
	}
	password, err := config.GetPassword(log)
	if err != nil {
		log.Fatal().Err(err).Str("password_file", config.PasswordFile).Msg("Could not read password from file")
	}
	cryptoHelper.LoginAs = &mautrix.ReqLogin{
		Type:       mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{Type: mautrix.IdentifierTypeUser, User: config.Username.String()},
		Password:   password,
	}
	cryptoHelper.DBAccountID = config.Username.String()

	err = cryptoHelper.Init(context.TODO())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize crypto helper")
	}
	client.Crypto = cryptoHelper

	syncer := client.Syncer.(mautrix.ExtensibleSyncer)
	syncer.OnSync(client.DontProcessOldEvents)
	if config.AutoJoin {
		syncer.OnEventType(event.StateMember, func(ctx context.Context, evt *event.Event) {
			if evt.StateKey == nil || *evt.StateKey != config.Username.String() {
				return
			}
			if evt.Content.AsMember().Membership == event.MembershipInvite {
				_, err := client.JoinRoom(ctx, evt.RoomID.String(), "", nil)
				if err != nil {
					log.Error().Err(err).Str("room_id", evt.RoomID.String()).Msg("Failed to join room")
				}
			}
		})
	}
	syncer.OnEventType(event.EventMessage, func(ctx context.Context, evt *event.Event) {
		retContent := getMsgResponse(log, evt)
		if retContent == nil {
			return
		}
		resp, err := client.SendMessageEvent(ctx, evt.RoomID, event.EventMessage, retContent)
		if err != nil {
			log.Err(err).Msg("couldn't send event")
			return
		}
		log.Info().Str("event_id", resp.EventID.String()).Msg("sent event")
	})

	err = client.Sync()
	if err != nil {
		log.Fatal().Err(err).Msg("error syncing")
	}
}

// this function assumes evt.Type is EventMessage
// return value is the message content to send back, if any
func getMsgResponse(log *zerolog.Logger, evt *event.Event) *event.MessageEventContent {
	// only respond to messages that were sent in the last five minutes so
	// that during an initial sync we don't respond to old messages
	if time.Unix(evt.Timestamp/1000, evt.Timestamp%1000).Before(time.Now().Add(time.Minute * -5)) {
		return nil
	}
	content := evt.Content.AsMessage()
	if content.MsgType != event.MsgText {
		return nil
	}
	mscs := getMSCs(content.Body)
	retBody := ""
	for i, msc := range mscs {
		log.Info().
			Str("room_id", evt.RoomID.String()).
			Str("event_id", evt.ID.String()).
			Uint("msc", msc).
			Msg("found MSC")
		if i > 0 {
			retBody += "\n"
		}
		retBody += getMSCResponse(log, msc)
	}
	if retBody == "" {
		return nil
	}
	return &event.MessageEventContent{
		MsgType: event.MsgNotice,
		Body:    retBody,
	}
}

func getMSCs(body string) (mscs []uint) {
	bodyNoReplies := event.TrimReplyFallbackText(body)
	matches := MSC_REGEX.FindAllStringSubmatch(bodyNoReplies, -1)
	mscSet := make(map[int]struct{})
	for _, match := range matches {
		// error can never happen because of %d in regex
		msc, _ := strconv.Atoi(match[1])
		_, exists := mscSet[msc]
		if exists {
			// don't add the same MSC twice
			continue
		}
		mscSet[msc] = struct{}{}
		mscs = append(mscs, uint(msc))
	}
	return mscs
}

func getMSCResponse(log *zerolog.Logger, msc uint) string {
	mscPR := fmt.Sprintf("https://github.com/matrix-org/matrix-spec-proposals/pull/%d", msc)
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/matrix-org/matrix-spec-proposals/pulls/%d", msc))
	if err != nil {
		log.Warn().Err(err).Uint("msc", msc).Msg("couldn't get MSC details")
		return mscPR
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log := log.With().Uint("msc", msc).Int("status_code", resp.StatusCode).Logger()
		byts, err := io.ReadAll(resp.Body)
		if err == nil {
			log = log.With().Str("body", string(byts)).Logger()
		}
		log.Warn().Msg("received non-200 status code while fetching MSC details")
		return mscPR
	}
	decoder := json.NewDecoder(resp.Body)
	var body struct {
		Title string `json:"title"` // only param we care about
	}
	err = decoder.Decode(&body)
	if err != nil {
		log.Warn().Err(err).Msg("couldn't decode PR details json")
		return mscPR
	}
	return fmt.Sprintf("%s %s", body.Title, mscPR)
}
