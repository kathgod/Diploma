package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// Функция получения текущего баланса пользователя
func logicGetBalance(r *http.Request) (int, []byte) {
	var emtyByte []byte
	db, errDB := sql.Open("postgres", ResHandParam.DataBaseURI)
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Println(err)
		}
	}(db)
	if errDB != nil {
		log.Println(dbOpenError)
		return http.StatusInternalServerError, emtyByte
	}

	db = createBalanceTable(db)

	flagAuthUser := authCheck(r, db)
	if !flagAuthUser {
		return http.StatusUnauthorized, emtyByte
	}

	orderNumbers := GetAllUsersOrderNumbers(db, r)
	var balanceStruct Balance
	for i := 0; i < len(orderNumbers); i++ {
		resp := RespGetOrderNumber{}
		accrualBaseAdressReqTxt := ResHandParam.AccrualSystemAddress + "/api/orders/" + orderNumbers[i].Order
		acrualResponse, err := http.Get(accrualBaseAdressReqTxt)
		if err != nil {
			log.Println(err)
		}
		if acrualResponse.StatusCode == http.StatusNoContent {
			resp.Status = "NEW"
			resp.Number = orderNumbers[i].Order
		}
		if acrualResponse.StatusCode == http.StatusNoContent {
			respB, err1 := io.ReadAll(acrualResponse.Body)
			if err1 != nil {
				log.Println(err1)
			}
			if err2 := json.Unmarshal(respB, &resp); err2 != nil {
				log.Println(err2)
			}
		}
		//resp.Order = ""
		resp.UploadedAt = orderNumbers[i].UploadedAt
		insertInToBalanceTable(db, r, resp)
		balanceStruct.Current = balanceStruct.Current + resp.Accrual
		//resOrderNumbers = append(resOrderNumbers, resp)
		errBC := acrualResponse.Body.Close()
		if errBC != nil {
			log.Println(errBC)
		}
	}
	withdraw := getAllWithdraw(db, r)
	balanceStruct.Withdrawn = withdraw
	balanceStruct.Current = balanceStruct.Current - withdraw
	byteFormatResp, errM := json.Marshal(balanceStruct)
	if errM != nil {
		log.Println(errM)
	}
	return http.StatusOK, byteFormatResp

}

// Функция создания таблицы баланса
func createBalanceTable(db *sql.DB) *sql.DB {
	query := `CREATE TABLE IF NOT EXISTS balancetable(coockie text, accrual float(2) default 0, withdrawn float(2) default 0, ordernumber text primary key, gotimewithdrawn text, sqltimewithdrawn timestamptz)`
	ctx, cancelfunc := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelfunc()
	res, err := db.ExecContext(ctx, query)
	if err != nil {
		log.Println(err)
	}
	_, err2 := res.RowsAffected()
	if err2 != nil {
		log.Println(err2)
	}
	//log.Printf("%d rows created CreateRegTable", rows)
	return db
}

// Функция записи в таблицу баланса
func insertInToBalanceTable(db *sql.DB, r *http.Request, resp RespGetOrderNumber) {
	query := `INSERT INTO balancetable(coockie, accrual, ordernumber) VALUES ($1, $2, $3) ON CONFLICT (ordernumber) DO NOTHING`
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
	res, err2 := stmt.ExecContext(ctx, cck.Value, resp.Accrual, resp.Order)
	if err2 != nil {
		log.Println(err2)
	}
	_, err3 := res.RowsAffected()
	if err3 != nil {
		log.Println(err3)
	}
}

// Balance Структурный тип для ответа json
type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn,omitempty"`
}
