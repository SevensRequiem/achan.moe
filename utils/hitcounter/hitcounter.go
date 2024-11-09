package hitcounter

import (
	"context"
	"log"
	"sync"
	"time"

	"achan.moe/database"
	"go.mongodb.org/mongo-driver/bson"
)

type HitCounter struct {
	ID    int `gorm:"primaryKey"`
	Hits  int // Exported field
	mutex sync.Mutex
	cache map[string]time.Time
}

var db = database.DB_Main

func init() {
	initialdocument()

}
func initialdocument() {
	_, err := db.Collection("hits").InsertOne(context.Background(), bson.M{"hits": 0})
	if err != nil {
		log.Fatal(err)
	}
}
func NewHitCounter() *HitCounter {
	log.Println("Creating a new HitCounter...")
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

	// Increment the count in the first row
	_, err := db.Collection("hits").UpdateOne(context.Background(), bson.M{}, bson.M{"$inc": bson.M{"hits": 1}})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Counted a hit from IP %s.\n", ip)
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
