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
)

type GameStore struct {
	sync.Mutex
	db        *bolt.DB
	immunes   map[string]*Immune
	conqueror *Player
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
	var conquerorBytes []byte
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		immunesBytes := b.Get(immunesKey)
		err := json.Unmarshal(immunesBytes, &immunes)
		if err != nil {
			log.Errorf("can't unmarshal immunes %s: %q", string(immunesBytes), err)
		}
		conquerorBytes = b.Get(conquerorKey)
		return nil
	})
	return &GameStore{
		db:        db,
		immunes:   immunes,
		conqueror: &Player{Name: string(conquerorBytes)},
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

	immunesBytes, err := json.Marshal(gs.immunes)
	if err != nil {
		log.Errorf("failed to marshal immunes: %q", err)
		return immune, false
	}
	log.Infof("Marshalled immunes: %s", string(immunesBytes))
	gs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		return b.Put(immunesKey, immunesBytes)
	})
	return immune, true
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

func (gs *GameStore) runWaiters() {
	gs.Lock()
	defer gs.Unlock()
	for _, immune := range gs.immunes {
		if immune.End.After(time.Now()) {
			go waiter(immune, fmt.Sprintf("Имун закончился: %s", immune.Player.Name))
		}
	}
}
