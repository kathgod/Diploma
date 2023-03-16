package handler

import (
	"context"
	"crypto/hmac"
	cr "crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
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
	case "DATABASE_URI":
	case "ACCRUAL_SYSTEM_ADDRESS":
	}
	log.Println(res)
	return res
}

var ResHandParam struct {
	DataBaseURI          string
	AccrualSystemAddress string
}

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
		return 500, emptcck
	}
	db = CreateRegTable(db)
	boolFlagExistRecord := IfExist(db, segStrInst)
	var affRows int64 = -1
	var cck *http.Cookie
	affRows, cck = AddRecordInRegTable(db, segStrInst)
	if affRows == 0 {
		if boolFlagExistRecord == true {
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
	rows, err2 := res.RowsAffected()
	if err2 != nil {
		log.Println(err2)
	}
	log.Printf("%d rows created CreateSQLTable", rows)
	return db
}

// IfExist проверяет наличие логина\пароля в нашей базе
func IfExist(db *sql.DB, segStrInst RegisterStruct) bool {
	check := new(string)
	row := db.QueryRow("select login from userRegTable where login == $1", segStrInst.Login)
	if err := row.Scan(check); err != sql.ErrNoRows {
		return true
	} else {
		return false
	}
}

func AddRecordInRegTable(db *sql.DB, segStrInst RegisterStruct) (int64, *http.Cookie) {
	query := `INSERT INTO idshortlongurl(login, password, authcoockie, idcoockie, keycoockie) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (login) DO NOTHING`
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
	log.Printf("%d rows created AddRecordInTable", rows)
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
	if boolFlagExistRecord == true {
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
	row := db.QueryRow("select authcoockie from userRegTable where login == $1", segStrInst.Login)
	if err := row.Scan(check); err != sql.ErrNoRows {
		return *check
	} else {
		return ""
	}
}
