package main

import (
	api "kitex/kitex_gen/api/bankservice"
	"log"
)

func main() {
	svr := api.NewServer(new(BankServiceImpl))

	err := svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
