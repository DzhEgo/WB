package main

import (
	"fmt"
	"github.com/nats-io/nats.go"
	"log"
)

func main() {
	url := "localhost"
	log.Println("Подключение...")

	nc, err := nats.Connect(url)
	if err != nil {
		log.Fatal(err)

	}
	defer nc.Close()

	_, err = nc.Subscribe("app", func(msg *nats.Msg) {
		fmt.Println("Сообщение NATS: %s\n", string(msg.Data))
	})
	if err != nil {
		log.Fatal(err)
		return
	}

	select {}
}
