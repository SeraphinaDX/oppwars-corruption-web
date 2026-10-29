// OPP Wars: Corruption Web — multiplayer accounts and the shared precinct.
// SPDX-License-Identifier: GPL-3.0-only
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
	_ "modernc.org/sqlite"
)

const (
	accountSessionAge = 30 * 24 * time.Hour
	onlineWindow = 5 * time.Minute
	maxLoginAttempts = 8
	loginWindowDuration = 10 * time.Minute
)

var validUsername = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{2,23}$`)

type loginWindow struct { count int; until time.Time }

type accountPublic struct {
	Username string `json:"username"`
	OfficerName string `json:"officerName"`
	Class string `json:"class"`
	Level int `json:"level"`
	Experience int `json:"experience"`
	Paragon int `json:"paragon"`
	Renegade int `json:"renegade"`
	Online bool `json:"online"`
}

type precinctSnapshot struct {
	Roster []accountPublic `json:"roster"`
	Leaderboard []accountPublic `json:"leaderboard"`
}

func initAccounts(saveDir string) (*sql.DB,error) {
	db,err:=sql.Open("sqlite",filepath.Join(saveDir,"precinct.sqlite"))
	if err!=nil {return nil,err}
	db.SetMaxOpenConns(1) // Small single-process game; protects from excess sqlite connections.
	for _,statement:=range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		`CREATE TABLE IF NOT EXISTS accounts (
			id TEXT PRIMARY KEY, username TEXT NOT NULL COLLATE NOCASE UNIQUE,
			password_hash BLOB NOT NULL, joined_at INTEGER NOT NULL,
			last_seen INTEGER NOT NULL DEFAULT 0,
			officer_name TEXT NOT NULL DEFAULT '', class TEXT NOT NULL DEFAULT '',
			level INTEGER NOT NULL DEFAULT 0, experience INTEGER NOT NULL DEFAULT 0,
			paragon INTEGER NOT NULL DEFAULT 0, renegade INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS auth_sessions (
			token_hash TEXT PRIMARY KEY, account_id TEXT NOT NULL,
			expires_at INTEGER NOT NULL,
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
		)`,
		"CREATE INDEX IF NOT EXISTS idx_account_recent ON accounts(last_seen DESC)",
		"CREATE INDEX IF NOT EXISTS idx_account_xp ON accounts(experience DESC)",
	} {
		if _,err=db.Exec(statement);err!=nil {db.Close();return nil,fmt.Errorf("initialize precinct database: %w",err)}
	}
	if err=os.Chmod(filepath.Join(saveDir,"precinct.sqlite"),0600);err!=nil {db.Close();return nil,err}
	return db,nil
}

func passwordDigest(password string) ([]byte,error) {
	salt:=make([]byte,16)
	if _,err:=rand.Read(salt);err!=nil {return nil,err}
	digest:=argon2.IDKey([]byte(password),salt,2,64*1024,1,32)
	return append(salt,digest...),nil
}
func passwordMatches(password string, stored []byte) bool {
	if len(stored)!=48 {return false}
	digest:=argon2.IDKey([]byte(password),stored[:16],2,64*1024,1,32)
	return subtle.ConstantTimeCompare(digest,stored[16:])==1
}

func sessionHash(token string) string {
	hash:=sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
func (s *server) identity(r *http.Request) (string,string) {
	cookie,err:=r.Cookie("oppwars_session")
	if err!=nil || !validID(cookie.Value) {return "",""}
	var id,username string
	err=s.db.QueryRow(`SELECT a.id,a.username FROM auth_sessions sess
		JOIN accounts a ON sess.account_id=a.id
		WHERE sess.token_hash=? AND sess.expires_at>?`,sessionHash(cookie.Value),time.Now().Unix()).Scan(&id,&username)
	if err!=nil {return "",""}
	_,_ = s.db.Exec("UPDATE accounts SET last_seen=? WHERE id=?",time.Now().Unix(),id)
	return id,username
}
func (s *server) setAuthCookie(w http.ResponseWriter,r *http.Request,token string,maxAge int) {
	http.SetCookie(w,&http.Cookie{
		Name:"oppwars_session",Value:token,Path:"/",
		MaxAge:maxAge,Secure:r.TLS!=nil||s.secureCookies,
		HttpOnly:true,SameSite:http.SameSiteStrictMode,
	})
}
func parseCredentials(w http.ResponseWriter,r *http.Request) (string,string,error) {
	if !sameOrigin(r) {return "","",errors.New("invalid origin")}
	if !strings.HasPrefix(r.Header.Get("Content-Type"),"application/json") {
		return "","",errors.New("expected application/json")
	}
	r.Body=http.MaxBytesReader(w,r.Body,4096)
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	decoder:=json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err:=decoder.Decode(&input);err!=nil {return "","",errors.New("invalid credentials request")}
	if err:=decoder.Decode(new(any));err==nil {return "","",errors.New("extra JSON values")}
	return strings.TrimSpace(input.Username),input.Password,nil
}
func validPassword(password string) bool {return len(password)>=12 && len(password)<=128}

func (s *server) makeAuthSession(w http.ResponseWriter,r *http.Request,accountID string) error {
	token,err:=randomID();if err!=nil {return err}
	_,err=s.db.Exec("INSERT INTO auth_sessions(token_hash,account_id,expires_at) VALUES(?,?,?)",
		sessionHash(token),accountID,time.Now().Add(accountSessionAge).Unix())
	if err!=nil {return err}
	s.setAuthCookie(w,r,token,int(accountSessionAge.Seconds()))
	return nil
}
func (s *server) updateOfficerProfile(accountID string,game *Game) error {
	p:=game.Player
	if game.Phase=="setup" {p=Player{}}
	_,err:=s.db.Exec(`UPDATE accounts
		SET officer_name=?,class=?,level=?,experience=?,paragon=?,renegade=?
		WHERE id=?`,
		p.Name,p.Class,p.Level,p.Experience,p.ParagonRank,p.RenegadeRank,accountID)
	return err
}
func (s *server) handleRegister(w http.ResponseWriter,r *http.Request) {
	username,password,err:=parseCredentials(w,r)
	if err!=nil {jsonReply(w,400,map[string]string{"error":err.Error()});return}
	if !validUsername.MatchString(username) || !validPassword(password) {
		jsonReply(w,400,map[string]string{"error":"username must be 3–24 letters/digits/underscores; password must be 12–128 bytes"});return
	}
	if id,_:=s.identity(r);id!="" {
		jsonReply(w,409,map[string]string{"error":"sign out before creating another account"});return
	}
	guestID,e,err:=s.getSession(w,r)
	if err!=nil {jsonReply(w,500,map[string]string{"error":"could not load current game"});return}
	e.mu.Lock()
	defer e.mu.Unlock()
	hash,err:=passwordDigest(password)
	if err!=nil {jsonReply(w,500,map[string]string{"error":"could not create account"});return}
	accountID,err:=randomID()
	if err!=nil {jsonReply(w,500,map[string]string{"error":"could not create account"});return}
	tx,err:=s.db.Begin()
	if err!=nil {jsonReply(w,500,map[string]string{"error":"database unavailable"});return}
	defer tx.Rollback()
	now:=time.Now().Unix()
	_,err=tx.Exec("INSERT INTO accounts(id,username,password_hash,joined_at,last_seen) VALUES(?,?,?,?,?)",
		accountID,username,hash,now,now)
	if err!=nil {
		jsonReply(w,409,map[string]string{"error":"username is already taken"});return
	}
	key:="account-"+accountID
	if err=s.save(key,e.game);err!=nil {jsonReply(w,500,map[string]string{"error":"could not preserve officer"});return}
	if err=tx.Commit();err!=nil {os.Remove(filepath.Join(s.saveDir,key+".json"));jsonReply(w,500,map[string]string{"error":"database unavailable"});return}
	if err=s.updateOfficerProfile(accountID,e.game);err!=nil {log.Printf("profile update failed: %v",err)}
	// Keep the guest save intact until the authenticated login succeeds.
	if err=s.makeAuthSession(w,r,accountID);err!=nil {jsonReply(w,500,map[string]string{"error":"account created; please log in"});return}
	s.mu.Lock()
	s.sessions[key]=e
	delete(s.sessions,guestID)
	s.mu.Unlock()
	if !strings.HasPrefix(guestID,"account-") { _=os.Remove(filepath.Join(s.saveDir,guestID+".json")) }
	jsonReply(w,200,map[string]any{"authenticated":true,"username":username})
}
func (s *server) limitedLogin(username string) bool {
	s.attemptMu.Lock();defer s.attemptMu.Unlock()
	now:=time.Now()
	key:=strings.ToLower(username)
	entry:=s.attempts[key]
	if now.After(entry.until) {entry=loginWindow{until:now.Add(loginWindowDuration)}}
	entry.count++
	s.attempts[key]=entry
	// Clear stale entries periodically and keep the map bounded.
	if len(s.attempts)>2048 {
		for k,v:=range s.attempts {if now.After(v.until) {delete(s.attempts,k)}}
	}
	return entry.count>maxLoginAttempts
}
func (s *server) handleLogin(w http.ResponseWriter,r *http.Request) {
	username,password,err:=parseCredentials(w,r)
	if err!=nil {jsonReply(w,400,map[string]string{"error":err.Error()});return}
	if !validUsername.MatchString(username) || len(password)>128 || s.limitedLogin(username) {
		jsonReply(w,429,map[string]string{"error":"invalid login or too many attempts; try again later"});return
	}
	var id,storedName string
	var stored []byte
	err=s.db.QueryRow("SELECT id,username,password_hash FROM accounts WHERE username=?",username).Scan(&id,&storedName,&stored)
	if err!=nil || !passwordMatches(password,stored) {
		jsonReply(w,401,map[string]string{"error":"invalid username or password"});return
	}
	if err=s.makeAuthSession(w,r,id);err!=nil {jsonReply(w,500,map[string]string{"error":"could not start session"});return}
	s.attemptMu.Lock()
	delete(s.attempts,strings.ToLower(username))
	s.attemptMu.Unlock()
	jsonReply(w,200,map[string]any{"authenticated":true,"username":storedName})
}
func (s *server) handleLogout(w http.ResponseWriter,r *http.Request) {
	if !sameOrigin(r) {jsonReply(w,403,map[string]string{"error":"invalid origin"});return}
	if cookie,err:=r.Cookie("oppwars_session");err==nil&&validID(cookie.Value) {
		_,_ = s.db.Exec("DELETE FROM auth_sessions WHERE token_hash=?",sessionHash(cookie.Value))
	}
	s.setAuthCookie(w,r,"",-1)
	jsonReply(w,200,map[string]bool{"authenticated":false})
}
func (s *server) queryOfficers(query string,args ...any) ([]accountPublic,error) {
	rows,err:=s.db.Query(query,args...)
	if err!=nil {return nil,err}
	defer rows.Close()
	out:=make([]accountPublic,0)
	for rows.Next() {
		var p accountPublic
		var last int64
		if err=rows.Scan(&p.Username,&p.OfficerName,&p.Class,&p.Level,&p.Experience,&p.Paragon,&p.Renegade,&last);err!=nil {return nil,err}
		p.Online=last>=time.Now().Add(-onlineWindow).Unix()
		out=append(out,p)
	}
	return out,rows.Err()
}
const officerFields="username,officer_name,class,level,experience,paragon,renegade,last_seen"
func (s *server) handlePrecinct(w http.ResponseWriter,r *http.Request) {
	if id,_:=s.identity(r);id=="" {jsonReply(w,401,map[string]string{"error":"create an account or sign in to visit the online precinct"});return}
	roster,err:=s.queryOfficers("SELECT "+officerFields+" FROM accounts WHERE last_seen>=? ORDER BY last_seen DESC LIMIT 60",
		time.Now().Add(-onlineWindow).Unix())
	if err!=nil {jsonReply(w,500,map[string]string{"error":"precinct unavailable"});return}
	leaders,err:=s.queryOfficers("SELECT "+officerFields+" FROM accounts WHERE level>0 ORDER BY experience DESC,username LIMIT 25")
	if err!=nil {jsonReply(w,500,map[string]string{"error":"precinct unavailable"});return}
	jsonReply(w,200,precinctSnapshot{Roster:roster,Leaderboard:leaders})
}
func (s *server) handleOfficer(w http.ResponseWriter,r *http.Request) {
	if id,_:=s.identity(r);id=="" {jsonReply(w,401,map[string]string{"error":"sign in to view officer records"});return}
	username:=r.PathValue("username")
	if !validUsername.MatchString(username) {jsonReply(w,404,map[string]string{"error":"officer not found"});return}
	officers,err:=s.queryOfficers("SELECT "+officerFields+" FROM accounts WHERE username=? LIMIT 1",username)
	if err!=nil {jsonReply(w,500,map[string]string{"error":"database unavailable"});return}
	if len(officers)==0 {jsonReply(w,404,map[string]string{"error":"officer not found"});return}
	jsonReply(w,200,officers[0])
}
func (s *server) authRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/auth",func(w http.ResponseWriter,r *http.Request){
		_,username:=s.identity(r)
		jsonReply(w,200,map[string]any{"authenticated":username!="","username":username})
	})
	mux.HandleFunc("POST /api/auth/register",s.handleRegister)
	mux.HandleFunc("POST /api/auth/login",s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout",s.handleLogout)
	mux.HandleFunc("GET /api/precinct",s.handlePrecinct)
	mux.HandleFunc("GET /api/officers/{username}",s.handleOfficer)
}

// Ensure only database-backed accounts are visible in the online precinct.
var _ = sync.Mutex{}
