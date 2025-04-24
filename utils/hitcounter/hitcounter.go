package hitcounter

import (
	"context"
	"log"
	"sync"
	"time"

	"achan.moe/database"
	"achan.moe/logs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type HitCounter struct {
	ID    int `gorm:"primaryKey"`
	Hits  int
	mutex sync.Mutex
	cache map[string]time.Time
}

var db = database.DB_Main

func NewHitCounter() *HitCounter {
	return &HitCounter{
		cache: make(map[string]time.Time),
	}
}

func (h *HitCounter) Hit(ip string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if hitTime, ok := h.cache[ip]; ok && time.Since(hitTime).Hours() < 2 {
		log.Printf("IP %s hit less than 2 hours ago, not counting this hit.\n", ip)
		return
	}

	options := options.Update().SetUpsert(true)

	_, err := db.Collection("stats").UpdateOne(context.Background(), bson.M{"_id": 1}, bson.M{"$inc": bson.M{"hit_count": 1}}, options)
	if err != nil {
		log.Fatal(err)
	}

	logs.Info("Hit from IP %s recorded.\n", ip)
	h.cache[ip] = time.Now()
	h.Hits++
}

func (h *HitCounter) GetHits() int {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	var hit HitCounter
	err := db.Collection("hits").FindOne(context.Background(), bson.M{}).Decode(&hit)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Retrieved hit count: %d.\n", hit.Hits)
	return hit.Hits
}
