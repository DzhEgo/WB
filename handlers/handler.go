package handlers

import (
	"Stream/DB"
	"Stream/service"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"net/http"
)

func AddHandler(db *gorm.DB, cache * ,w http.ResponseWriter, r *http.Request) {
	var order DB.Orders

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
	page := template.Must(template.ParseFiles("../../web/index.html"))
	orderId := r.URL.Query().Get("id")
	var data = struct {
		Order *Orders
	}{}

	if orderId != "" {
		if cachedOrder, exists := cache.GetCache(orderId); exists {
			data.Order = cachedOrder
		} else {
			var order Orders
			res := db.Preload("Delivery").Preload("Payment").Preload("Items").First(&order, "order_uid = ?", orderId)
			if res.Error == nil {
				data.Order = &order
				cache.SetCache(orderId, &order)
			}
		}
	}

	page.Execute(w, data)
}
