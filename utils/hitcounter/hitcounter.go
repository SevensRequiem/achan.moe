package hitcounter

import (
	"log"
	"sync"
	"time"

	"achan.moe/database"
	"gorm.io/gorm"
)

type HitCounter struct {
	ID    int `gorm:"primaryKey"`
	Hits  int // Exported field
	mutex sync.Mutex
	cache map[string]time.Time
}

var db = database.DB

func init() {
	db.AutoMigrate(&HitCounter{})
	ensureInitialRecord()
}

func ensureInitialRecord() {
	var hitCounter HitCounter
	if err := db.First(&hitCounter, 1).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			hitCounter = HitCounter{ID: 1, Hits: 0}
			if err := db.Create(&hitCounter).Error; err != nil {
				log.Fatalf("Failed to create initial hit counter record: %v", err)
			}
			log.Println("Created initial hit counter record.")
		} else {
			log.Fatalf("Failed to check initial hit counter record: %v", err)
		}
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
	err := db.Model(&HitCounter{}).Where("id = ?", 1).Update("hits", gorm.Expr("hits + ?", 1)).Error
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
	err := db.First(&hit, 1).Error
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Retrieved hit count: %d.\n", hit.Hits)
	return hit.Hits
}
