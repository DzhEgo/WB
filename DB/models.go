package DB

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
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

func DbInit(dsn string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Ошибка к подключению к БД!", err)
	}
	db.AutoMigrate(&Orders{}, &Delivery{}, &Items{}, &Payment{})
	if err != nil {
		log.Fatal("Ошибка миграции БД!", err)
	}
	return db
}
