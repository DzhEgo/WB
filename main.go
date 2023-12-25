package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/nats-io/stan.go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"html/template"
	"log"
	"net/http"
	"sync"
)

type Delivery struct {
	gorm.Model
	OrderUID string `gorm:"uniqueIndex"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Zip      string `json:"zip"`
	City     string `json:"city"`
	Address  string `json:"address"`
	Region   string `json:"region"`
	Email    string `json:"email"`
}

type Payment struct {
	gorm.Model
	OrderUID     string `gorm:"uniqueIndex"`
	Transaction  string `json:"transaction"`
	RequestID    string `json:"request_id"`
	Currency     string `json:"currency"`
	Provider     string `json:"provider"`
	Amount       int    `json:"amount"`
	PaymentDt    int    `json:"payment_dt"`
	Bank         string `json:"bank"`
	DeliveryCost int    `json:"delivery_cost"`
	GoodsTotal   int    `json:"goods_total"`
	CustomFee    int    `json:"custom_fee"`
}

type Items struct {
	gorm.Model
	OrderUID    string `gorm:"uniqueIndex"`
	ChrtID      int    `json:"chrt_id"`
	TrackNumber string `json:"track_number"`
	Price       int    `json:"price"`
	RID         string `json:"rid"`
	Name        string `json:"name"`
	Sale        int    `json:"sale"`
	Size        string `json:"size"`
	TotalPrice  int    `json:"total_price"`
	NmId        int    `json:"nm_id"`
	Brand       string `json:"brand"`
	Status      int    `json:"status"`
}

type Orders struct {
	gorm.Model
	OrderUID          string   `json:"order_uid" gorm:"uniqueIndex"`
	TrackNumber       string   `json:"track_number"`
	Entry             string   `json:"entry"`
	Delivery          Delivery `json:"deliveries" gorm:"foreignKey:OrderUID; references:OrderUID"`
	Payment           Payment  `json:"payments" gorm:"foreignKey:OrderUID; references:OrderUID"`
	Items             []Items  `json:"items" gorm:"foreignKey:OrderUID; references:OrderUID"`
	Locale            string   `json:"locale"`
	InternalSignature string   `json:"internal_signature"`
	CustomerID        string   `json:"customer_id"`
	DeliveryService   string   `json:"delivery_service"`
	ShardKey          string   `json:"shard_key"`
	SmID              int      `json:"sm_id"`
	DateCreated       string   `json:"date_created"`
	OofShard          string   `json:"oof_shard"`
}

type Cache struct {
	data  map[string]*Orders
	mutex sync.RWMutex
}

var db *gorm.DB
var cache *Cache

func ConnectNats() (stan.Conn, error) {
	natsURL := stan.DefaultNatsURL
	sc, err := stan.Connect("nats-Max", "DzhEgo", stan.NatsURL(natsURL))
	if err != nil {
		return nil, err
	}
	return sc, nil
}

func SubdcribeToChan(sc stan.Conn, subject string, cb stan.MsgHandler) (stan.Subscription, error) {
	subscription, err := sc.Subscribe(subject, cb)
	if err != nil {
		return nil, err
	}
	return subscription, nil
}

func StartServer() {
	r := mux.NewRouter()
	r.HandleFunc("/order", AddHandler).Methods("POST")
	r.HandleFunc("/order/{id}", FindByIdHandler).Methods("GET")
	r.HandleFunc("/order", PageIdHandler).Methods("GET")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	var err error

	dsn := "user=postgres password=Lax212212 dbname=test_stream sslmode=disable"
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	db.AutoMigrate(&Orders{}, &Delivery{}, &Items{}, &Payment{})

	cache = NewCache()
	if err := LoadCache(db, cache); err != nil {
		log.Fatalf("Ошибка загрузки кэша из БД: %v", err)
	}

	//sc, err := ConnectNats()
	//if err != nil {
	//	log.Fatalf("Ошибка с подключением к NATS Streaming", err)
	//}
	//defer sc.Close()

	//_, err = SubdcribeToChan(sc, "Test", messageHandler)
	//if err != nil {
	//	log.Fatalf("Ошибка подписки на канал: %v", err)
	//}

	fmt.Println("Запуск сервера...")
	StartServer()
}

func NewCache() *Cache {
	return &Cache{data: make(map[string]*Orders)}
}

func (c *Cache) SetCache(key string, order *Orders) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data[key] = order
}

func (c *Cache) GetCache(key string) (*Orders, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	order, exists := c.data[key]
	return order, exists
}

func LoadCache(db *gorm.DB, cache *Cache) error {
	var orders []Orders
	if err := db.Find(&orders).Error; err != nil {
		return err
	}

	for _, order := range orders {
		cache.SetCache(order.OrderUID, &order)
	}
	return nil
}

func AddHandler(w http.ResponseWriter, r *http.Request) {
	var order Orders

	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	res := db.Create(&order)
	if res.Error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	cacheKey := order.OrderUID
	cache.SetCache(cacheKey, &order)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("Данные записаны!")
}

func FindByIdHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var order Orders

	if id == "" {
		http.Error(w, "Поле ID не должно быть пустым!", http.StatusBadRequest)
		return
	}

	if cachedOrder, exists := cache.GetCache(id); exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cachedOrder)
		return
	}

	res := db.Preload("Delivery").Preload("Payment").Preload("Items").First(&order, "order_uid = ?", id)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "Данные с этим ID не найдены", http.StatusNotFound)
		} else {
			http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		}
		return
	}

	cache.SetCache(id, &order)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(order)
}

func PageIdHandler(w http.ResponseWriter, r *http.Request) {
	page := template.Must(template.ParseFiles("index.html"))

	orderId := r.URL.Query().Get("id")
	var order Orders
	var data = struct {
		Order *Orders
	}{}

	if orderId != "" {
		res := db.Preload("Delivery").Preload("Payment").Preload("Items").First(&order, "order_uid = ?", orderId)
		if res.Error == nil {
			data.Order = &order
		}
	}

	page.Execute(w, data)
}
