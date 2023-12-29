package service

import (
	"Stream/DB"
	"gorm.io/gorm"
	"sync"
)

type Cache struct {
	data  map[string]*DB.Orders
	mutex sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{data: make(map[string]*DB.Orders)}
}

func (c *Cache) SetCache(key string, order *DB.Orders) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data[key] = order
}

func (c *Cache) GetCache(key string) (*DB.Orders, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	order, exists := c.data[key]
	return order, exists
}

func LoadCache(db *gorm.DB, cache *Cache) error {
	var orders []DB.Orders
	if err := db.Find(&orders).Error; err != nil {
		return err
	}

	for _, order := range orders {
		orderCopy := order
		cache.SetCache(order.OrderUID, &orderCopy)
	}
	return nil
}
