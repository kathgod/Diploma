package handler

import (
	"log"
	"net/http"
)

// PostRegister функция регистрации пользователя
func PostRegister() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resLF, cck := logicPostRegister(r)
		//log.Println("func PostRegister, resLF:", resLF)
		switch {
		case resLF == 200:
			http.SetCookie(w, cck)
			w.Header().Set("Authorization", cck.Value)
			w.WriteHeader(http.StatusOK)
		case resLF == 400:
			w.WriteHeader(http.StatusBadRequest)
		case resLF == 409:
			w.WriteHeader(http.StatusConflict)
		case resLF == 500:
			w.WriteHeader(http.StatusInternalServerError)
		default:
			log.Println("Invalid value of variable resLF")
		}
	}
}

// PostLogin функция аутентификации пользователя
func PostLogin() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resLF, cck := logicPostLogin(r)
		//log.Println("func PostLogin, resLF:", resLF)
		switch {
		case resLF == 200:
			http.SetCookie(w, cck)
			w.Header().Set("Authorization", cck.Value)
			w.WriteHeader(http.StatusOK)
		case resLF == 400:
			w.WriteHeader(http.StatusBadRequest)
		case resLF == 401:
			w.WriteHeader(http.StatusUnauthorized)
		case resLF == 500:
			w.WriteHeader(http.StatusInternalServerError)
		default:
			log.Println("Invalid value of variable resLF")
		}
	}
}

// PostOrders функция загрузки пользователем номера заказа для расчёта
func PostOrders() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resLF := logicPostOrders(r)
		//log.Println(resLF)
		switch {
		case resLF == 200:
			w.WriteHeader(http.StatusOK)
		case resLF == 202:
			w.WriteHeader(http.StatusAccepted)
		case resLF == 400:
			w.WriteHeader(http.StatusBadRequest)
		case resLF == 401:
			w.WriteHeader(http.StatusUnauthorized)
		case resLF == 409:
			w.WriteHeader(http.StatusConflict)
		case resLF == 422:
			w.WriteHeader(http.StatusUnprocessableEntity)
		case resLF == 500:
			w.WriteHeader(http.StatusInternalServerError)
		default:
			log.Println("Invalid value of variable resLF")
		}
	}
}

// GetOrders получение списка загруженных пользователем номеров заказов, статусов их обработки и информации о начислениях
func GetOrders() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resLF, byteResp := logicGetOrders(r)
		switch {
		case resLF == 200:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(byteResp)
		case resLF == 204:
			w.WriteHeader(http.StatusNoContent)
		case resLF == 401:
			w.WriteHeader(http.StatusUnauthorized)
		case resLF == 500:
			w.WriteHeader(http.StatusInternalServerError)
		default:
			log.Println("Invalid value of variable resLF")
		}
	}
}

// GetBalance функция  получения текущего баланса счёта баллов лояльности пользователя
func GetBalance() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resLF, byteResp := logicGetBalance(r)
		switch {
		case resLF == 200:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(byteResp)
		case resLF == 401:
			w.WriteHeader(http.StatusUnauthorized)
		case resLF == 500:
			w.WriteHeader(http.StatusInternalServerError)
		default:
			log.Println("Invalid value of variable resLF")
		}
	}
}

// PostBalanceWithdraw функция запроса на списание баллов с накопительного счёта в счёт оплаты нового заказа
func PostBalanceWithdraw() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resLF := logicPostBalanceWithdraw(r)
		switch {
		case resLF == 200:
			w.WriteHeader(http.StatusOK)
		case resLF == 401:
			w.WriteHeader(http.StatusUnauthorized)
		case resLF == 402:
			w.WriteHeader(http.StatusPaymentRequired)
		case resLF == 422:
			w.WriteHeader(http.StatusUnprocessableEntity)
		case resLF == 500:
			w.WriteHeader(http.StatusInternalServerError)
		default:
			log.Println("Invalid value of variable resLF")
		}
	}
}

// GetUserWithdraw функция получения информации о выводе средств с накопительного счёта пользователем.
func GetUserWithdraw() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resLF, byteResp := logicGetUserWithdraw(r)
		switch {
		case resLF == 200:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(byteResp)
		case resLF == 204:
			w.WriteHeader(http.StatusNoContent)
		case resLF == 401:
			w.WriteHeader(http.StatusUnauthorized)
		case resLF == 500:
			w.WriteHeader(http.StatusInternalServerError)
		default:
			log.Println("Invalid value of variable resLF")
		}
	}
}
