package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Функция получения всех списаний
func getAllWithdraw(db *sql.DB, r *http.Request) float64 {
	cck, errCck := r.Cookie("userId")
	if errCck != nil {
		log.Println(errCck)
	}
	q := `select withdrawn from balancetable where coockie=$1`
	rows, err1 := db.Query(q, cck.Value)
	if err1 != nil {
		log.Println(err1)
	}
	if rows.Err() != nil {
		log.Println(rows.Err())
	}
	var withdraw float64
	for rows.Next() {
		var buff float64
		errRow := rows.Scan(&buff)
		if errRow != nil {
			log.Println(errRow)

			continue
		}
		withdraw = withdraw + buff
	}
	return withdraw
}

// Функция запроса на списание баллов
func logicPostBalanceWithdraw(r *http.Request) int {
	rawBsp, err := decompress(io.ReadAll(r.Body))
	if err != nil {
		log.Println(postBodyError)
		return http.StatusBadRequest
	}

	balanceWithdrawInst := BalanceWithdraw{}
	if err := json.Unmarshal(rawBsp, &balanceWithdrawInst); err != nil {
		log.Println(postBodyError)
		return http.StatusBadRequest
	}

	db, errDB := sql.Open("postgres", ResHandParam.DataBaseURI)
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Println(err)
		}
	}(db)
	if errDB != nil {
		log.Println(dbOpenError)
		return http.StatusInternalServerError
	}

	flagAuthUser := authCheck(r, db)
	if !flagAuthUser {
		return http.StatusUnauthorized
	}

	buff, errConv := strconv.Atoi(balanceWithdrawInst.Order)
	if errConv != nil {
		log.Println(errConv)
	}
	flagFormatOrder := Valid(buff)
	if !flagFormatOrder {
		return http.StatusUnprocessableEntity
	}

	balance, err := getBalance(db, r)
	if err != nil {
		return http.StatusInternalServerError
	}
	if balanceWithdrawInst.Sum > balance {
		return http.StatusPaymentRequired
	}

	insertWithdrawInToBalanceTable(db, balanceWithdrawInst, r)
	return http.StatusOK
}

// BalanceWithdraw Структурный тип запроса json
type BalanceWithdraw struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

// Функция изменения таблицы баланса
func insertWithdrawInToBalanceTable(db *sql.DB, balanceWithdrawInst BalanceWithdraw, r *http.Request) {
	query := `INSERT INTO balancetable(coockie, ordernumber, withdrawn, gotimewithdrawn, sqltimewithdrawn) VALUES ($1, $2, $3, $4, now()) ON CONFLICT (ordernumber) DO NOTHING`
	ctx, cancelfunc := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelfunc()

	cck, errCck := r.Cookie("userId")
	if errCck != nil {
		log.Println(errCck)
	}

	stmt, err0 := db.PrepareContext(ctx, query)
	if err0 != nil {
		log.Println(err0)
	}
	defer func(stmt *sql.Stmt) {
		err1 := stmt.Close()
		if err1 != nil {
			log.Println(err1)
		}
	}(stmt)

	now := time.Now()
	timeStr := now.Format("2006-01-02T15:04:05Z07:00")

	res, err2 := stmt.ExecContext(ctx, cck.Value, balanceWithdrawInst.Order, balanceWithdrawInst.Sum, timeStr)
	if err2 != nil {
		log.Println(err2)
	}
	_, err3 := res.RowsAffected()
	if err3 != nil {
		log.Println(err3)
	}
}

// Функция получения баланса
func getBalance(db *sql.DB, r *http.Request) (float64, error) {

	orderNumbers := GetAllUsersOrderNumbers(db, r)
	var balanceStruct Balance
	for i := 0; i < len(orderNumbers); i++ {
		resp := RespGetOrderNumber{}
		accrualBaseAdressReqTxt := ResHandParam.AccrualSystemAddress + "/api/orders/" + orderNumbers[i].Order
		accrualResponse, err := http.Get(accrualBaseAdressReqTxt)
		if err != nil {
			log.Println(err)
		}

		if accrualResponse.StatusCode == 204 {
			resp.Status = "NEW"
			resp.Number = orderNumbers[i].Order
		}

		if accrualResponse.StatusCode == 200 {
			respB, err := io.ReadAll(accrualResponse.Body)
			if err != nil {
				log.Println(err)
				return 0, err
			}
			if err := json.Unmarshal(respB, &resp); err != nil {
				log.Println(err)
				return 0, err
			}
		}

		balanceStruct.Current = balanceStruct.Current + resp.Accrual
		//resOrderNumbers = append(resOrderNumbers, resp)
		errBC := accrualResponse.Body.Close()
		if errBC != nil {
			log.Println(errBC)
			return 0, err
		}
	}

	return balanceStruct.Current, nil
}

// Функция получения списаний пользователя
func logicGetUserWithdraw(r *http.Request) (int, []byte) {
	var emptyByte []byte
	db, errDB := sql.Open("postgres", ResHandParam.DataBaseURI)
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Println(err)
		}
	}(db)
	if errDB != nil {
		log.Println(dbOpenError)
		return http.StatusInternalServerError, emptyByte
	}

	flagAuthUser := authCheck(r, db)
	if !flagAuthUser {
		return http.StatusUnauthorized, emptyByte
	}

	massUserWithdrawStruct := selectAllUserWithdraw(db, r)
	if len(massUserWithdrawStruct) == 0 {
		return http.StatusNoContent, emptyByte
	} else {
		byteFormatResp, errM := json.Marshal(massUserWithdrawStruct)
		if errM != nil {
			log.Println(errM)
		}
		return http.StatusOK, byteFormatResp
	}

}

// Функция выбора всех списаний из таблицы
func selectAllUserWithdraw(db *sql.DB, r *http.Request) []UserWithdrawStruct {
	cck, errCck := r.Cookie("userId")
	if errCck != nil {
		log.Println(errCck)
	}
	q := `select ordernumber, withdrawn, gotimewithdrawn from balancetable where coockie=$1 order by sqltimewithdrawn asc`
	rows, err1 := db.Query(q, cck.Value)
	if err1 != nil {
		log.Println(err1)
	}
	if rows.Err() != nil {
		log.Println(rows.Err())
	}
	var massUserWithdrawStruct []UserWithdrawStruct
	for rows.Next() {
		buff := UserWithdrawStruct{}
		errRow := rows.Scan(&buff.Order, &buff.Sum, &buff.ProcessedAt)
		if errRow != nil {
			log.Println(errRow)
			continue
		}
		massUserWithdrawStruct = append(massUserWithdrawStruct, buff)
	}
	return massUserWithdrawStruct
}
