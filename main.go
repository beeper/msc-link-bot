package main

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"

	_ "github.com/mattn/go-sqlite3"

	log "github.com/sirupsen/logrus"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

var mscRegex *regexp.Regexp

func main() {
	store := NewMSCBotStore()
	client := mkClient(store)

	cryptoDB, err := sql.Open("sqlite3", "crypto.db")
	if err != nil {
		log.Fatalf("couldn't open crypto db: %v", err)
	}
	defer cryptoDB.Close()

	cryptoLogger := cryptoLogger{}
	cryptoStore := crypto.NewSQLCryptoStore(
		cryptoDB,
		"sqlite3",
		fmt.Sprintf("%s/%s", client.UserID, client.DeviceID),
		client.DeviceID,
		[]byte("xyz.hnitbjorg.msc_link_bot"),
		cryptoLogger,
	)
	err = cryptoStore.CreateTables()
	if err != nil {
		log.Fatalf("couldn't create crypto store tables: %v", err)
	}

	olmMachine := crypto.NewOlmMachine(client, cryptoLogger, cryptoStore, store)
	err = olmMachine.Load()
	if err != nil {
		log.Fatalf("couldn't load olm machine: %v", err)
	}

	mscRegex, err = regexp.Compile("\\b(?:MSC|msc)(\\d+)\\b")
	if err != nil {
		// should never happen
		log.Fatalf("couldn't compile regex: %v", err)
	}

	syncer := client.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnSync(olmMachine.ProcessSyncResponse)
	syncer.OnEventType(event.StateMember, func(_ mautrix.EventSource, evt *event.Event) {
		olmMachine.HandleMemberEvent(evt)
	})
	syncer.OnEvent(store.UpdateState)
	syncer.OnEventType(event.EventMessage, func(_ mautrix.EventSource, evt *event.Event) {
		ret := getMsgResponse(client, evt)
		if ret == "" {
			return
		}
		resp, err := client.SendMessageEvent(evt.RoomID, event.EventMessage, event.MessageEventContent{
			MsgType: event.MsgText,
			Body:    ret,
		})
		if err != nil {
			log.Errorf("couldn't send event: %v", err)
			return
		}
		log.Infof("sent event %v", resp.EventID)
	})
	syncer.OnEventType(event.EventEncrypted, func(_ mautrix.EventSource, encEvt *event.Event) {
		evt, err := olmMachine.DecryptMegolmEvent(encEvt)
		if err != nil {
			log.Errorf("couldn't decrypt event %v: %v", encEvt.ID, err)
			return
		}
		if evt.Type != event.EventMessage {
			return
		}
		ret := getMsgResponse(client, evt)
		if ret == "" {
			return
		}
		content := event.MessageEventContent{
			MsgType: event.MsgText,
			Body:    ret,
		}
		encrypted, err := olmMachine.EncryptMegolmEvent(evt.RoomID, evt.Type, content)
		if err != nil {
			if isBadEncryptError(err) {
				log.Errorf("couldn't encrypt event: %v", err)
				return
			}
			log.Debugf("got %s while trying to encrypt message; sharing group session and trying again...", err)
			err = olmMachine.ShareGroupSession(evt.RoomID, store.GetRoomMembers(evt.RoomID))
			if err != nil {
				log.Errorf("couldn't share group session: %v", err)
				return
			}
			encrypted, err = olmMachine.EncryptMegolmEvent(evt.RoomID, evt.Type, content)
			if err != nil {
				log.Errorf("couldn't encrypt event(2): %v", err)
				return
			}
		}
		resp, err := client.SendMessageEvent(evt.RoomID, event.EventEncrypted, encrypted)
		if err != nil {
			log.Errorf("couldn't send encrypted event: %v", err)
			return
		}
		log.Infof("sent encrypted event %v", resp.EventID)
	})
	syncer.OnEvent(func (_ mautrix.EventSource, evt *event.Event) {
		err := olmMachine.FlushStore()
		if err != nil {
			panic(err)
		}
	})

	err = client.Sync()
	if err != nil {
		log.Fatalf("error syncing: %v", err)
	}
}

func isBadEncryptError(err error) bool {
	return err != crypto.SessionExpired && err != crypto.SessionNotShared && err != crypto.NoGroupSession
}

// this function assumes evt.Type is EventMessage
// return value is the body of the message to send back, if any
func getMsgResponse(client *mautrix.Client, evt *event.Event) string {
	content := evt.Content.AsMessage()
	if content.MsgType != event.MsgText {
		return ""
	}
	mscs := getMSCs(content.Body)
	ret := ""
	for i, msc := range mscs {
		log.Infof("MSC: %v %v\n", evt.ID, msc)
		if i > 0 {
			ret += "\n"
		}
		ret += fmt.Sprintf("https://github.com/matrix-org/matrix-doc/pull/%v", msc)
	}
	return ret
}

func getMSCs(body string) (mscs []string) {
	matches := mscRegex.FindAllStringSubmatch(body, -1)
	for _, match := range matches {
		mscs = append(mscs, match[1])
	}
	return mscs
}

func mkClient(store mautrix.Storer) *mautrix.Client {
	homeserver := os.Getenv("HOMESERVER")
	if homeserver == "" {
		log.Fatal("required envvar HOMESERVER not set")
	}

	userID := os.Getenv("USER_ID")
	if userID == "" {
		log.Fatal("required envvar USER_ID not set")
	}

	deviceID := os.Getenv("DEVICE_ID")
	if deviceID == "" {
		log.Fatal("required envvar DEVICE_ID not set")
	}

	accessToken := os.Getenv("ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("required envvar ACCESS_TOKEN not set")
	}

	client, err := mautrix.NewClient(homeserver, id.UserID(userID), accessToken)
	if err != nil {
		log.Fatalf("couldn't create client: %v", err)
	}
	client.DeviceID = id.DeviceID(deviceID)
	client.Store = store

	return client
}
