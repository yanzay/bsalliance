package main

import (
	"encoding/json"
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
		json.Unmarshal(immunesBytes, immunes)
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

func (gs *GameStore) AddImmune(player *Player, start time.Time) *Immune {
	gs.Lock()
	defer gs.Unlock()
	var end time.Time
	if gs.conqueror != nil && gs.conqueror.Name == player.Name {
		end = start.Add(immuneConquerorDuration)
	} else {
		end = start.Add(immuneStandardDuration)
	}
	immune := &Immune{Player: player, End: end}
	gs.immunes[player.Name] = immune

	immunesBytes, err := json.Marshal(gs.immunes)
	if err != nil {
		log.Errorf("failed to marshal immunes: %q", err)
		return immune
	}
	log.Infof("Marshalled immunes: %s", string(immunesBytes))
	gs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		return b.Put(immunesKey, immunesBytes)
	})
	return immune
}
