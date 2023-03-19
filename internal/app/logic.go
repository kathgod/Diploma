package handler

import (
	"context"
	"crypto/hmac"
	cr "crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	_ "github.com/lib/pq"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	postBodyError = "Bad Post request body"
	dbOpenError   = "Open DataBase Error"
)

func HandParam(name string, flg *string) string {
	res := ""
	globEnv := os.Getenv(name)
	if globEnv != "" {
		res = globEnv
	} else {
		res = *flg
	}
	switch name {
	case "RUN_ADDRESS":
		log.Println("RUN_ADDRESS:", res)
	case "DATABASE_URI":
		log.Println("DATABASE_URI:", res)
	case "ACCRUAL_SYSTEM_ADDRESS":
		log.Println("ACCRUAL_SYSTEM_ADDRESS", res)
	}
	return res
}

var ResHandParam struct {
	DataBaseURI          string
	AccrualSystemAddress string
}

//---------------------------------------------------------------------------
//---------------------------------------------------------------------------

type RegisterStruct struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func logicPostRegister(r *http.Request) (int, *http.Cookie) {
	var emptcck *http.Cookie

	rawBsp, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(postBodyError)
		return 400, emptcck
	}
	segStrInst := RegisterStruct{}
	if err := json.Unmarshal(rawBsp, &segStrInst); err != nil {
		log.Println(postBodyError)
		return 400, emptcck
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
		log.Println(errDB)
		return 500, emptcck
	}
	db = CreateRegTable(db)
	boolFlagExistRecord := IfExist(db, segStrInst)
	var affRows int64 = -1
	var cck *http.Cookie
	affRows, cck = AddRecordInRegTable(db, segStrInst)
	if affRows == 0 {
		if boolFlagExistRecord {
			return 409, emptcck
		} else {
			return 500, emptcck
		}
	} else {
		return 200, cck
	}
}

func CreateRegTable(db *sql.DB) *sql.DB {
	query := `CREATE TABLE IF NOT EXISTS userRegTable(login text primary key, password text, authcoockie text, idcoockie text, keycoockie text)`
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

func IfExist(db *sql.DB, segStrInst RegisterStruct) bool {
	check := new(string)
	row := db.QueryRow("select login from userRegTable where login = $1", segStrInst.Login)
	if err := row.Scan(check); err != sql.ErrNoRows {
		return true
	} else {
		return false
	}
}

func AddRecordInRegTable(db *sql.DB, segStrInst RegisterStruct) (int64, *http.Cookie) {
	query := `INSERT INTO userRegTable(login, password, authcoockie, idcoockie, keycoockie) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (login) DO NOTHING`
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
	cck := createCoockie()
	rik := resIDKey[cck.Value]
	rikID := rik.id
	rikKey := rik.key
	res, err2 := stmt.ExecContext(ctx, segStrInst.Login, segStrInst.Password, cck.Value, rikID, rikKey)
	if err2 != nil {
		log.Println(err2)
	}
	rows, err3 := res.RowsAffected()
	if err3 != nil {
		log.Println(err3)
	}
	//log.Printf("%d rows created AddRecordInTable", rows)
	return rows, cck
}

type idKey struct {
	id  string
	key string
}

var resIDKey = map[string]idKey{"0": {"0", "0"}}

func createCoockie() *http.Cookie {
	id := make([]byte, 16)
	key := make([]byte, 16)
	_, err1 := cr.Read(id)
	_, err2 := cr.Read(key)

	if err1 != nil || err2 != nil {
		log.Println(err1, err2)
	}
	h := hmac.New(sha256.New, key)
	h.Write(id)
	sgnIDKey := h.Sum(nil)
	cck := &http.Cookie{
		Name:  "userId",
		Value: hex.EncodeToString(sgnIDKey),
	}
	resIDKey[hex.EncodeToString(sgnIDKey)] = idKey{hex.EncodeToString(id), hex.EncodeToString(key)}
	return cck
}

//---------------------------------------------------------------------------
//---------------------------------------------------------------------------

func logicPostLogin(r *http.Request) (int, *http.Cookie) {
	var emptcck *http.Cookie
	rawBsp, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(postBodyError)
		return 400, emptcck
	}
	segStrInst := RegisterStruct{}
	if err := json.Unmarshal(rawBsp, &segStrInst); err != nil {
		log.Println(postBodyError)
		return 400, emptcck
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
		return 500, emptcck
	}
	boolFlagExistRecord := IfExist(db, segStrInst)
	if boolFlagExistRecord {
		cckValue := GetCckValue(db, segStrInst)
		cck := &http.Cookie{
			Name:  "userId",
			Value: cckValue,
		}
		return 200, cck
	} else {
		return 401, emptcck
	}
}

func GetCckValue(db *sql.DB, segStrInst RegisterStruct) string {
	check := new(string)
	row := db.QueryRow("select authcoockie from userRegTable where login = $1", segStrInst.Login)
	if err := row.Scan(check); err != sql.ErrNoRows {
		return *check
	} else {
		return ""
	}
}

//---------------------------------------------------------------------------
//---------------------------------------------------------------------------

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
		return 500
	}

	rawBsp, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(postBodyError)
		return 400
	}
	orderNumber := string(rawBsp)

	log.Println(orderNumber)

	db = CreateOrderTable(db)

	flagAuthUser := authCheck(r, db)
	if !flagAuthUser {
		return 401
	} else {
		var affrow int64 = -1
		affrow = AddRecordInOrderTable(db, r, orderNumber)
		if affrow == 0 {
			userCoockieCheckOrderTable := CheckOrderTable(orderNumber, db)
			cck, err1 := r.Cookie("userId")
			if err1 != nil {
				log.Println(err1)
				return 500
			}
			if userCoockieCheckOrderTable == cck.Value {
				return 200
			} else {
				return 409
			}
		} else {
			return 202
		}

	}

}

func CreateOrderTable(db *sql.DB) *sql.DB {
	query := `CREATE TABLE IF NOT EXISTS orderTable(ordernumber text primary key, authcoockie text, timecreate text, mydateandtime timestamptz)`
	ctx, cancelfunc := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelfunc()
	res, err := db.ExecContext(ctx, query)
	if err != nil {
		log.Println(err)
	}
	rows, err2 := res.RowsAffected()
	if err2 != nil {
		log.Println(err2)
	}
	log.Printf("%d rows created CreateRegTable", rows)
	return db
}

func authCheck(r *http.Request, db *sql.DB) bool {
	cck, err := r.Cookie("userId")
	if err != nil {
		log.Println("Error1 Coockie check", err)
		return false
	}
	check := new(string)
	row := db.QueryRow("select login from userRegTable where authcoockie = $1", cck.Value)
	if err1 := row.Scan(check); err1 != sql.ErrNoRows {
		return true
	} else {
		return false
	}
}

func CheckOrderTable(orderNumber string, db *sql.DB) string {
	var check string
	row := db.QueryRow("select authcoockie from orderTable where ordernumber = $1", orderNumber)
	if err1 := row.Scan(&check); err1 != sql.ErrNoRows {
		log.Println(check)
		return check
	} else {
		return ""
	}
}

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
	log.Printf("%d rows created AddRecordInTable", rows)
	return rows
}

//---------------------------------------------------------------------------
//---------------------------------------------------------------------------

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
		return 500, emptyByte
	}

	flagAuthUser := authCheck(r, db)
	if !flagAuthUser {
		return 401, emptyByte
	}

	orderNumbers := GetAllUsersOrderNumbers(db, r)
	if len(orderNumbers) == 0 {
		return 204, emptyByte
	} else {
		var resOrderNumbers []RespGetOrderNumber
		for i := 0; i < len(orderNumbers); i++ {
			resp := RespGetOrderNumber{}
			accrualBaseAdressReqTxt := ResHandParam.AccrualSystemAddress + "/api/orders/" + orderNumbers[i].Order
			acrualResponse, err := http.Get(accrualBaseAdressReqTxt)
			if err != nil {
				log.Println(err)
			}
			respB, err1 := io.ReadAll(acrualResponse.Body)
			if err1 != nil {
				log.Println(err1)
			}
			if err2 := json.Unmarshal(respB, &resp); err2 != nil {
				log.Println(err2)
			}
			resp.UploadedAt = orderNumbers[i].UploadedAt
			resOrderNumbers = append(resOrderNumbers, resp)
		}
		byteFormatResp, errM := json.Marshal(resOrderNumbers)
		if errM != nil {
			log.Println(errM)
		}
		return 200, byteFormatResp
	}

}

type RespGetOrderNumber struct {
	Order      string `json:"order"`
	Status     string `json:"status"`
	Accrual    string `json:"accrual,omitempty"`
	UploadedAt string `json:"uploaded_at"`
}

func GetAllUsersOrderNumbers(db *sql.DB, r *http.Request) []RespGetOrderNumber {
	cck, err := r.Cookie("userId")
	if err != nil {
		log.Println(err)
	}

	var orderNumbers []RespGetOrderNumber
	rows, err1 := db.Query("select ordernumber, timecreate from orderTable where authcoockie = $1 order by mydateandtime asc", cck.Value)
	if err1 != nil {
		log.Println(err1)
	}
	for rows.Next() {
		var oneNumber RespGetOrderNumber
		errRow := rows.Scan(oneNumber.Order, oneNumber.UploadedAt)
		if errRow != nil {
			log.Println(errRow)
			continue
		}
		orderNumbers = append(orderNumbers, oneNumber)
	}
	return orderNumbers
}
