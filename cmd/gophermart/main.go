package main

import (
	MyHandler "diploma/internal/app"
	"flag"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
)

const (
	srError = "Server Listen Error"
)

var (
	runAddress           *string
	dataBaseURI          *string
	accrualSystemAddress *string
)

func init() {
	runAddress = flag.String("a", "", "RUN_ADDRESS")
	dataBaseURI = flag.String("d", "", "DATABASE_URI")
	accrualSystemAddress = flag.String("r", "", "ACCRUAL_SYSTEM_ADDRESS")
}

func main() {
	flag.Parse()
	portNumber := MyHandler.HandParam("RUN_ADDRESS", runAddress)
	MyHandler.ResHandParam.DataBaseURI = MyHandler.HandParam("DATABASE_URI", dataBaseURI)
	MyHandler.ResHandParam.AccrualSystemAddress = MyHandler.HandParam("ACCRUAL_SYSTEM_ADDRESS", accrualSystemAddress)

	resPostRegister := MyHandler.PostRegister()
	resPostLogin := MyHandler.PostLogin()
	resPostOrders := MyHandler.PostOrders()
	resGetOrders := MyHandler.GetOrders()
	resGetBalance := MyHandler.GetBalance()
	resPostBalanceWithdraw := MyHandler.PostBalanceWithdraw()
	resGetUserWithdraw := MyHandler.GetUserWithdraw()

	rtr := chi.NewRouter()
	rtr.Post("/api/user/register", resPostRegister)
	rtr.Post("/api/user/login", resPostLogin)
	rtr.Post("/api/user/orders", resPostOrders)
	rtr.Get("/api/user/orders", resGetOrders)
	rtr.Get("/api/user/balance", resGetBalance)
	rtr.Post("/api/user/balance/withdraw", resPostBalanceWithdraw)
	rtr.Get("/api/user/withdrawals", resGetUserWithdraw)

	err2 := http.ListenAndServe(portNumber, rtr)
	if err2 != nil {
		log.Println(srError)
	}

}
