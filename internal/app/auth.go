package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	cr "crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

// HandParam функция обработки флагов
func HandParam(name string, flg *string) string {
	res := ""
	globEnv := os.Getenv(name)
	if globEnv != "" {
		res = globEnv
	} else if flg != nil {
		res = *flg
	}
	switch name {
	case "RUN_ADDRESS":
		log.Println("RUN_ADDRESS:", res)
	case "DATABASE_URI":
		log.Println("DATABASE_URI:", res)
	case "ACCRUAL_SYSTEM_ADDRESS":
		log.Println("ACCRUAL_SYSTEM_ADDRESS", res)
	default:
		log.Println("NOT ALLOWED FLAG")
	}
	return res
}

// ResHandParam Структура флагов
var ResHandParam struct {
	DataBaseURI          string
	AccrualSystemAddress string
}

// RegisterStruct Структура данных регистрации
type RegisterStruct struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// Функция регистрации пользователя
func logicPostRegister(r *http.Request) (int, *http.Cookie) {
	var emptycookie *http.Cookie

	rawBsp, err := decompress(io.ReadAll(r.Body))
	if err != nil {
		log.Println(postBodyError)
		return http.StatusBadRequest, emptycookie
	}
	segStrInst := RegisterStruct{}
	if err := json.Unmarshal(rawBsp, &segStrInst); err != nil {
		log.Println(postBodyError)
		return http.StatusBadRequest, emptycookie
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
		return http.StatusInternalServerError, emptycookie
	}
	db, err = CreateRegTable(db)
	if err != nil {
		return http.StatusInternalServerError, emptycookie
	}
	boolFlagExistRecord := IfExist(db, segStrInst)
	var affRows int64 = -1
	var cck *http.Cookie
	affRows, cck = AddRecordInRegTable(db, segStrInst)
	if affRows == 0 {
		if boolFlagExistRecord {
			return http.StatusConflict, emptycookie
		}
		return http.StatusInternalServerError, emptycookie
	}
	return http.StatusOK, cck
}

// CreateRegTable Функция создания базы данных
func CreateRegTable(db *sql.DB) (*sql.DB, error) {
	query := `CREATE TABLE IF NOT EXISTS userRegTable(login text primary key, password text, authcoockie text, idcoockie text, keycoockie text)`
	ctx, cancelfunc := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelfunc()
	res, err := db.ExecContext(ctx, query)
	if err != nil {
		log.Println(err)
		return db, err
	}
	_, err = res.RowsAffected()
	if err != nil {
		log.Println(err)
		return db, err
	}
	//log.Printf("%d rows created CreateRegTable", rows)
	return db, nil
}

// IfExist Функция проверки существования логина
func IfExist(db *sql.DB, segStrInst RegisterStruct) bool {
	check := new(string)
	row := db.QueryRow("select login from userRegTable where login = $1", segStrInst.Login)
	if err := row.Scan(check); err != sql.ErrNoRows {
		return true
	}
	return false
}

// AddRecordInRegTable Функция добавления записи в таблицу авторизациии
func AddRecordInRegTable(db *sql.DB, segStrInst RegisterStruct) (int64, *http.Cookie) {
	query := `INSERT INTO userRegTable(login, password, authcoockie, idcoockie, keycoockie) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (login) DO NOTHING`
	ctx, cancelfunc := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelfunc()
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		log.Println(err)
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {
			log.Println(err)
		}
	}(stmt)
	cck, err := createCoockie()
	if err != nil {
		log.Println(err)
		return -1, cck
	}
	rik := resIDKey[cck.Value]
	rikID := rik.id
	rikKey := rik.key
	res, err := stmt.ExecContext(ctx, segStrInst.Login, segStrInst.Password, cck.Value, rikID, rikKey)
	if err != nil {
		log.Println(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Println(err)
	}
	//log.Printf("%d rows created AddRecordInTable", rows)
	return rows, cck
}

// Структурный тип данных для создания куки файла
type idKey struct {
	id  string
	key string
}

var resIDKey = map[string]idKey{"0": {"0", "0"}}

// Функция создания куки файла
func createCoockie() (*http.Cookie, error) {
	var emptycck *http.Cookie
	id := make([]byte, 16)
	key := make([]byte, 16)
	_, err1 := cr.Read(id)
	if err1 != nil {
		log.Println(err1)
		return emptycck, err1
	}
	_, err2 := cr.Read(key)
	if err2 != nil {
		log.Println(err2)
		return emptycck, err2
	}

	h := hmac.New(sha256.New, key)
	h.Write(id)
	sgnIDKey := h.Sum(nil)
	cck := &http.Cookie{
		Name:  "userId",
		Value: hex.EncodeToString(sgnIDKey),
	}
	resIDKey[hex.EncodeToString(sgnIDKey)] = idKey{hex.EncodeToString(id), hex.EncodeToString(key)}
	return cck, nil
}

// Функция авторизации пользователя
func logicPostLogin(r *http.Request) (int, *http.Cookie) {
	var emptcck *http.Cookie
	rawBsp, err := decompress(io.ReadAll(r.Body))
	if err != nil {
		log.Println(postBodyError)
		return http.StatusBadRequest, emptcck
	}
	segStrInst := RegisterStruct{}
	if err := json.Unmarshal(rawBsp, &segStrInst); err != nil {
		log.Println(postBodyError)
		return http.StatusBadRequest, emptcck
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
		return http.StatusInternalServerError, emptcck
	}
	boolFlagExistRecord := IfExist(db, segStrInst)
	if boolFlagExistRecord {
		cckValue := GetCckValue(db, segStrInst)
		cck := &http.Cookie{
			Name:  "userId",
			Value: cckValue,
		}
		return http.StatusOK, cck
	} else {
		return http.StatusUnauthorized, emptcck
	}
}

// UserWithdrawStruct Структурный тип ответа json
type UserWithdrawStruct struct {
	Order       string  `json:"order"`
	Sum         float32 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

// Функция обработких сжатых данных
func decompress(data []byte, err0 error) ([]byte, error) {
	if err0 != nil {
		return nil, fmt.Errorf("error 0 %v", err0)
	}

	r, err1 := gzip.NewReader(bytes.NewReader(data))
	if err1 != nil {
		return data, nil
	}
	defer func(r *gzip.Reader) {
		err := r.Close()
		if err != nil {
			log.Println(err)
		}
	}(r)

	var b bytes.Buffer

	_, err := b.ReadFrom(r)
	if err != nil {
		return data, nil
	}

	return b.Bytes(), nil
}

// Функция проверки авторизации
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
