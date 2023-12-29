package handlers

import (
	"Stream/DB"
	"Stream/service"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/nats-io/stan.go"
	"gorm.io/gorm"
	"html/template"
	"net/http"
)

func AddHandler(db *gorm.DB, cache *service.Cache, sc stan.Conn, w http.ResponseWriter, r *http.Request) {
	var order DB.Orders

	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		service.Publish(sc, "test", "Ошибка при декодировании!")
		return
	}
	defer r.Body.Close()

	res := db.Create(&order)
	if res.Error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		service.Publish(sc, "test", "Ошибка при добавлении данных в ДБ!")
		return
	}

	service.Publish(sc, "test", fmt.Sprintf("Данные с ID %s добавлены в БД!", order.OrderUID))

	cacheKey := order.OrderUID
	cache.SetCache(cacheKey, &order)

	service.Publish(sc, "test", fmt.Sprintf("Данные с ID %s добавлены в кэш!", order.OrderUID))

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("Данные записаны!")
}

func FindByIdHandler(db *gorm.DB, cache *service.Cache, sc stan.Conn, w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var order DB.Orders

	if id == "" {
		http.Error(w, "Поле ID не должно быть пустым!", http.StatusBadRequest)
		service.Publish(sc, "test", "Ошибка! Поле ID не должно быть пустым!")
		return
	}

	if cachedOrder, exists := cache.GetCache(id); exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cachedOrder)
		service.Publish(sc, "test", fmt.Sprintf("Данные с ID %s получены из кэша!", id))
		return
	}

	res := db.Preload("Delivery").Preload("Payment").Preload("Items").First(&order, "order_uid = ?", id)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "Данные с этим ID не найдены", http.StatusNotFound)
			service.Publish(sc, "test", fmt.Sprintf("Данные с ID %s не существуют в БД!", id))
		} else {
			http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
			service.Publish(sc, "test", "Ошибка сервера при попытке получить данные!")
		}
		return
	}

	cache.SetCache(id, &order)
	service.Publish(sc, "cache_add", fmt.Sprintf("Данные с ID %s добавлены в кэш!", id))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(order)
	service.Publish(sc, "test", fmt.Sprintf("Данные с ID %s получен из БД!", id))
}

func PageIdHandler(db *gorm.DB, cache *service.Cache, sc stan.Conn, w http.ResponseWriter, r *http.Request) {
	page := template.Must(template.ParseFiles("../../web/index.html"))
	orderId := r.URL.Query().Get("id")
	var data = struct {
		Order *DB.Orders
	}{}

	if orderId == "" {
		http.Error(w, "Необходимо предоставить ID!", http.StatusBadRequest)
		service.Publish(sc, "test", "ID не предоставлен!")
		return
	}

	if cachedOrder, exists := cache.GetCache(orderId); exists {
		data.Order = cachedOrder
		service.Publish(sc, "test", fmt.Sprintf("Данные с ID %s получены из кэша!", orderId))
	} else {
		var order DB.Orders
		res := db.Preload("Delivery").Preload("Payment").Preload("Items").First(&order, "order_uid = ?", orderId)
		if res.Error != nil {
			http.Error(w, "Ошибка при получении данных из БД!", http.StatusInternalServerError)
			service.Publish(sc, "test", fmt.Sprintf("Ошибка при получении данных c ID %s из БД!", orderId))
			return
		}
		data.Order = &order
		cache.SetCache(orderId, &order)
		service.Publish(sc, "test", fmt.Sprintf("Данные с ID %s получены из кэша!", orderId))
	}

	if err := page.Execute(w, data); err != nil {
		http.Error(w, "Ошибка при генерации страницы!", http.StatusInternalServerError)
		service.Publish(sc, "test", fmt.Sprintf("Ошибка при генерации страницы для данных c ID %s!", orderId))
		return
	}
}
