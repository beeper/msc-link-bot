package main

import (
	"sync"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type MSCBotStore struct {
	sync.RWMutex
	FilterIDs map[id.UserID]string
	NextBatches map[id.UserID]string
	Rooms map[id.RoomID]*mautrix.Room
}

func NewMSCBotStore() *MSCBotStore {
	return &MSCBotStore{
		FilterIDs: make(map[id.UserID]string),
		NextBatches: make(map[id.UserID]string),
		Rooms: make(map[id.RoomID]*mautrix.Room),
	}
}

// mautrix.Storer interface implemented below

func (s *MSCBotStore) SaveFilterID(userID id.UserID, filterID string) {
	s.Lock()
	defer s.Unlock()
	s.FilterIDs[userID] = filterID
}

func (s *MSCBotStore) LoadFilterID(userID id.UserID) string {
	s.RLock()
	defer s.RUnlock()
	return s.FilterIDs[userID]
}

func (s *MSCBotStore) SaveNextBatch(userID id.UserID, nextBatchToken string) {
	s.Lock()
	defer s.Unlock()
	s.NextBatches[userID] = nextBatchToken
}

func (s *MSCBotStore) LoadNextBatch(userID id.UserID) string {
	s.RLock()
	defer s.RUnlock()
	return s.NextBatches[userID]
}

func (s *MSCBotStore) SaveRoom(room *mautrix.Room) {
	s.Lock()
	defer s.Unlock()
	s.Rooms[room.ID] = room
}

func (s *MSCBotStore) LoadRoom(roomID id.RoomID) *mautrix.Room {
	s.RLock()
	defer s.RUnlock()
	return s.Rooms[roomID]
}

func (s *MSCBotStore) UpdateState(_ mautrix.EventSource, evt *event.Event) {
	if !evt.Type.IsState() {
		return
	}
	room := s.LoadRoom(evt.RoomID)
	if room == nil {
		room = mautrix.NewRoom(evt.RoomID)
		s.SaveRoom(room)
	}
	room.UpdateState(evt)
}

// crypto.StateStore interface implemented below

// IsEncrypted returns whether a room is encrypted.
func (s *MSCBotStore) IsEncrypted(roomID id.RoomID) bool {
	s.RLock()
	defer s.RUnlock()
	if room, exists := s.Rooms[roomID]; exists {
		return room.GetStateEvent(event.StateEncryption, "") != nil
	}
	return false
}

// GetEncryptionEvent returns the encryption event's content for an encrypted room.
func (s *MSCBotStore) GetEncryptionEvent(roomID id.RoomID) *event.EncryptionEventContent {
	s.RLock()
	defer s.RUnlock()
	room, exists := s.Rooms[roomID]
	if !exists {
		return nil
	}
	evt := room.GetStateEvent(event.StateEncryption, "")
	content, ok := evt.Content.Parsed.(*event.EncryptionEventContent)
	if !ok {
		return nil
	}
	return content
}

// FindSharedRooms returns the encrypted rooms that another user is also in for a user ID.
func (s *MSCBotStore) FindSharedRooms(userID id.UserID) []id.RoomID {
	s.RLock()
	defer s.RUnlock()
	var sharedRooms []id.RoomID
	for roomID, room := range s.Rooms {
		// if room isn't encrypted, skip
		if room.GetStateEvent(event.StateEncryption, "") == nil {
			continue
		}
		if room.GetMembershipState(userID) == event.MembershipJoin {
			sharedRooms = append(sharedRooms, roomID)
		}
	}
	return sharedRooms
}

func (s *MSCBotStore) GetRoomMembers(roomID id.RoomID) []id.UserID {
	var members []id.UserID
	for userID, evt := range s.Rooms[roomID].State[event.StateMember] {
		if evt.Content.Parsed.(*event.MemberEventContent).Membership.IsInviteOrJoin() {
			members = append(members, id.UserID(userID))
		}
	}
	return members
}
