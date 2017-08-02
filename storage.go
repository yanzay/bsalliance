package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/yanzay/log"
)

var (
	bucketName   = []byte("bsalliance")
	conquerorKey = []byte("conqueror")
	immunesKey   = []byte("immunes")
	usersKey     = []byte("users")
)

type GameStore struct {
	sync.Mutex
	db        *bolt.DB
	immunes   map[string]*Immune
	conqueror *Player
	users     map[string]bool
}

func NewGameStore(file string) *GameStore {
	db, err := bolt.Open(file, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		log.Fatalf("Can't open database: %q", err)
	}
	db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists(bucketName)
		return nil
	})
	immunes := make(map[string]*Immune)
	users := make(map[string]bool)
	var conquerorBytes []byte
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		immunesBytes := b.Get(immunesKey)
		err := json.Unmarshal(immunesBytes, &immunes)
		if err != nil {
			log.Errorf("can't unmarshal immunes %s: %q", string(immunesBytes), err)
			return err
		}
		conquerorBytes = b.Get(conquerorKey)
		usersBytes := b.Get(usersKey)
		err = json.Unmarshal(usersBytes, &users)
		if err != nil {
			log.Errorf("can't unmarshal users %s: %q", string(immunesBytes), err)
			return err
		}
		return nil
	})
	return &GameStore{
		db:        db,
		immunes:   immunes,
		conqueror: &Player{Name: string(conquerorBytes)},
		users:     users,
	}
}

func (gs *GameStore) SetConqueror(player *Player) {
	gs.Lock()
	gs.conqueror = player
	gs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		return b.Put(conquerorKey, []byte(player.Name))
	})
	gs.Unlock()
}

func (gs *GameStore) GetConqueror() *Player {
	gs.Lock()
	conqueror := gs.conqueror
	gs.Unlock()
	return conqueror
}

func (gs *GameStore) GetImmunes() map[string]*Immune {
	gs.Lock()
	defer gs.Unlock()
	return gs.immunes
}

func (gs *GameStore) GetImmune(name string) *Immune {
	gs.Lock()
	defer gs.Unlock()
	return gs.immunes[name]
}

func (gs *GameStore) AddImmune(player *Player, start time.Time) (*Immune, bool) {
	gs.Lock()
	defer gs.Unlock()
	immune := &Immune{
		Player: player,
		End:    start.Add(gs.immuneDuration(player)),
	}
	if gs.existingImmune(immune) {
		return immune, false
	}
	gs.immunes[player.Name] = immune
	gs.saveImmunes()
	return immune, true
}

func (gs *GameStore) DeleteImmune(name string) {
	gs.Lock()
	defer gs.Unlock()
	delete(gs.immunes, name)
	gs.saveImmunes()
}

func (gs *GameStore) AddUser(name string) {
	gs.Lock()
	gs.users[name] = true
	gs.saveUsers()
	gs.Unlock()
}

func (gs *GameStore) DelUser(name string) {
	gs.Lock()
	delete(gs.users, name)
	gs.saveUsers()
	gs.Unlock()
}

func (gs *GameStore) IsUser(name string) bool {
	gs.Lock()
	_, ok := gs.users[name]
	gs.Unlock()
	return ok
}

func (gs *GameStore) GetUsers() []string {
	users := make([]string, 0)
	gs.Lock()
	for user := range gs.users {
		users = append(users, user)
	}
	gs.Unlock()
	return users
}

func (gs *GameStore) saveImmunes() error {
	gs.expireImmunes()
	immunesBytes, err := json.Marshal(gs.immunes)
	if err != nil {
		log.Errorf("failed to marshal immunes: %q", err)
		return err
	}
	log.Infof("Marshalled immunes: %s", string(immunesBytes))
	return gs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		return b.Put(immunesKey, immunesBytes)
	})
}

func (gs *GameStore) saveUsers() error {
	usersBytes, err := json.Marshal(gs.users)
	if err != nil {
		log.Errorf("failed to marshal users: %q", err)
		return err
	}
	log.Infof("Marshalled users: %s", string(usersBytes))
	return gs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		return b.Put(usersKey, usersBytes)
	})
}

func (gs *GameStore) existingImmune(immune *Immune) bool {
	existing := gs.immunes[immune.Player.Name]
	if existing == nil {
		return false
	}
	return existing.End.Add(gs.immuneDuration(immune.Player)).After(immune.End)
}

func (gs *GameStore) immuneDuration(player *Player) time.Duration {
	if gs.conqueror != nil && gs.conqueror.Name == player.Name {
		return immuneConquerorDuration
	}
	return immuneStandardDuration
}

func (gs *GameStore) expireImmunes() {
	newImmunes := make(map[string]*Immune)
	for name, immune := range gs.immunes {
		if immune.End.After(time.Now().Add(-6 * time.Hour)) {
			newImmunes[name] = immune
		}
	}
	gs.immunes = newImmunes
}

func (gs *GameStore) runWaiters() {
	gs.Lock()
	defer gs.Unlock()
	for _, immune := range gs.immunes {
		if immune.End.After(time.Now()) {
			go waiter(immune, fmt.Sprintf(MessageEndOfImmune, immune.Player.Name), 0)
		}
	}
}
