package main

import (
	"Stream/DB"
	"Stream/handlers"
	"Stream/service"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/nats-io/stan.go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"net/http"
)

func StartServer(db *gorm.DB, cache *service.Cache, sc stan.Conn) {
	r := mux.NewRouter()
	r.HandleFunc("/order", func(w http.ResponseWriter, r *http.Request) {
		handlers.AddHandler(db, cache, sc, w, r)
	}).Methods("POST")
	r.HandleFunc("/order/{id}", func(w http.ResponseWriter, r *http.Request) {
		handlers.FindByIdHandler(db, cache, sc, w, r)
	}).Methods("GET")
	r.HandleFunc("/order", func(w http.ResponseWriter, r *http.Request) {
		handlers.PageIdHandler(db, cache, sc, w, r)
	}).Methods("GET")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	dsn := "user=postgres password=Lax212212 dbname=test_stream sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	err = db.AutoMigrate(&DB.Orders{}, &DB.Delivery{}, &DB.Items{}, &DB.Payment{})
	if err != nil {
		log.Fatal("Ошибка при миграции БД: ", err)
	}

	cache := service.NewCache()
	if err := service.LoadCache(db, cache); err != nil {
		log.Fatalf("Ошибка загрузки кэша из БД: %v", err)
	}

	sc, err := service.ConnectNATS("test-cluster", "DzhEgo", "nats://localhost:4223")
	if err != nil {
		log.Fatalf("Ошибка подключения к NATS Streaming: %v", err)
	}
	defer sc.Close()

	sub, err := service.SubscribeToNATS(sc, "test", func(m *stan.Msg) {
		log.Printf("Получено сообщение: %s\n", string(m.Data))
	})
	if err != nil {
		log.Fatal("Ошибка подписки на канал: ", err)
	}
	defer sub.Unsubscribe()

	fmt.Println("Запуск сервера...")
	StartServer(db, cache, sc)
}
