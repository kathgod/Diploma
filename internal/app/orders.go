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

// GetCckValue Функция извлечения куки пользователя из БД
func GetCckValue(db *sql.DB, segStrInst RegisterStruct) string {
	check := new(string)
	row := db.QueryRow("select authcoockie from userRegTable where login = $1", segStrInst.Login)
	if err := row.Scan(check); err != sql.ErrNoRows {
		return *check
	} else {
		return ""
	}
}

// Функция записи заказа в систему
func logicPostOrders(r *http.Request) int {
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

	rawBsp, err := decompress(io.ReadAll(r.Body))
	if err != nil {
		log.Println(postBodyError)
		return http.StatusBadRequest
	}
	orderNumber := string(rawBsp)
	buff, errB := strconv.Atoi(orderNumber)
	if errB != nil {
		log.Println(errB)
	}
	flagFormatOrder := Valid(buff)
	if !flagFormatOrder {
		return http.StatusUnprocessableEntity
	}

	db = CreateOrderTable(db)

	flagAuthUser := authCheck(r, db)
	if !flagAuthUser {
		return http.StatusUnauthorized
	} else {
		var affrow int64 = -1
		affrow = AddRecordInOrderTable(db, r, orderNumber)
		if affrow == 0 {
			userCoockieCheckOrderTable := CheckOrderTable(orderNumber, db)
			cck, err1 := r.Cookie("userId")
			if err1 != nil {
				log.Println(err1)
				return http.StatusInternalServerError
			}
			if userCoockieCheckOrderTable == cck.Value {
				return http.StatusOK
			} else {
				return http.StatusConflict
			}
		} else {
			return http.StatusAccepted
		}
	}
}

// Valid Функция проверка корректности заказа
func Valid(number int) bool {
	return (number%10+checksum(number/10))%10 == 0
}

// Функция подсчета
func checksum(number int) int {
	var luhn int

	for i := 0; number > 0; i++ {
		cur := number % 10

		if i%2 == 0 { // even
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
		number = number / 10
	}
	return luhn % 10
}

// CreateOrderTable Функция создания таблицы заказов
func CreateOrderTable(db *sql.DB) *sql.DB {
	query := `CREATE TABLE IF NOT EXISTS orderTable(ordernumber text primary key, authcoockie text, timecreate text, mydateandtime timestamptz)`
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

// CheckOrderTable Функция проверки наличия заказа в таблице
func CheckOrderTable(orderNumber string, db *sql.DB) string {
	var check string
	row := db.QueryRow("select authcoockie from orderTable where ordernumber = $1", orderNumber)
	if err1 := row.Scan(&check); err1 != sql.ErrNoRows {
		//log.Println(check)
		return check
	} else {
		return ""
	}
}

// AddRecordInOrderTable Функция добавления записи в таблицу заказов
func AddRecordInOrderTable(db *sql.DB, r *http.Request, orderNumber string) int64 {
	query := `INSERT INTO orderTable(ordernumber, authcoockie, timecreate, mydateandtime) VALUES ($1, $2, $3, now()) ON CONFLICT (ordernumber) DO NOTHING`
	ctx, cancelfunc := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelfunc()
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

	cck, err := r.Cookie("userId")
	if err != nil {
		log.Println("Error1 Coockie check", err)
	}

	now := time.Now()
	timeStr := now.Format("2006-01-02T15:04:05Z07:00")

	res, err2 := stmt.ExecContext(ctx, orderNumber, cck.Value, timeStr)
	if err2 != nil {
		log.Println(err2)
	}
	rows, err3 := res.RowsAffected()
	if err3 != nil {
		log.Println(err3)
	}
	//log.Printf("%d rows created AddRecordInTable", rows)
	return rows
}

// Функция получения списка заказов
func logicGetOrders(r *http.Request) (int, []byte) {
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

	orderNumbers := GetAllUsersOrderNumbers(db, r)
	if len(orderNumbers) == 0 {
		return http.StatusNoContent, emptyByte
	} else {
		var resOrderNumbers []RespGetOrderNumber
		for i := 0; i < len(orderNumbers); i++ {
			resp := RespGetOrderNumber{}
			accrualBaseAdressReqTxt := ResHandParam.AccrualSystemAddress + "/api/orders/" + orderNumbers[i].Order
			acrualResponse, err := http.Get(accrualBaseAdressReqTxt)
			if err != nil {
				log.Println(err)
			}

			if acrualResponse.StatusCode == http.StatusOK {
				respB, err1 := io.ReadAll(acrualResponse.Body)
				if err1 != nil {
					log.Println(err1)
				}
				if err2 := json.Unmarshal(respB, &resp); err2 != nil {
					log.Println(err2)
				}
				resp.Number = orderNumbers[i].Order
			} else if acrualResponse.StatusCode == http.StatusNoContent {
				resp.Status = "NEW"
				resp.Number = orderNumbers[i].Order
			}
			resp.Order = ""
			resp.UploadedAt = orderNumbers[i].UploadedAt
			resOrderNumbers = append(resOrderNumbers, resp)
			errBC := acrualResponse.Body.Close()
			if errBC != nil {
				log.Println(errBC)
			}

		}
		byteFormatResp, errM := json.Marshal(resOrderNumbers)
		if errM != nil {
			log.Println(errM)
		}

		return http.StatusOK, byteFormatResp
	}

}

// RespGetOrderNumber Структурный тип для вывода ответа json
type RespGetOrderNumber struct {
	Number     string  `json:"number,omitempty"`
	Order      string  `json:"order,omitempty"`
	Status     string  `json:"status,omitempty"`
	Accrual    float64 `json:"accrual,omitempty"`
	UploadedAt string  `json:"uploaded_at"`
}

// GetAllUsersOrderNumbers Функция получения всех заказов пользователя
func GetAllUsersOrderNumbers(db *sql.DB, r *http.Request) []RespGetOrderNumber {
	cck, err := r.Cookie("userId")
	if err != nil {
		log.Println(err)
	}

	var orderNumbers []RespGetOrderNumber
	q := `select ordernumber, timecreate from orderTable where authcoockie = $1 order by mydateandtime asc`
	rows, err1 := db.Query(q, cck.Value)
	if err1 != nil {
		log.Println(err1)
	}
	if rows.Err() != nil {
		log.Println(rows.Err())
	}
	for rows.Next() {
		var oneNumber RespGetOrderNumber
		errRow := rows.Scan(&oneNumber.Order, &oneNumber.UploadedAt)
		if errRow != nil {
			log.Println(errRow)
			continue
		}

		orderNumbers = append(orderNumbers, oneNumber)
	}
	return orderNumbers
}
