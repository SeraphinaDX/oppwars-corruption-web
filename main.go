// OPP Wars: Corruption — Web Edition (GPL-3.0-only)
package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed static/*
var embeddedAssets embed.FS

type entry struct {
	mu         sync.Mutex
	game       *Game
	lastActive time.Time
}
type server struct {
	mu       sync.Mutex
	sessions map[string]*entry
	saveDir  string
}

func newServer(saveDir string) (*server, error) {
	if err := os.MkdirAll(saveDir, 0700); err != nil {
		return nil, err
	}
	return &server{saveDir: saveDir, sessions: make(map[string]*entry)}, nil
}
func randomID() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
func validID(v string) bool {
	if len(v) != 64 {
		return false
	}
	for _, c := range v {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func (s *server) getSession(w http.ResponseWriter, r *http.Request) (string, *entry, error) {
	cookie, err := r.Cookie("oppwars_session")
	id := ""
	if err == nil && validID(cookie.Value) {
		id = cookie.Value
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	// Evict inactive in-memory sessions; their saves remain on disk.
	for key, item := range s.sessions {
		if now.Sub(item.lastActive) > 24*time.Hour {
			delete(s.sessions, key)
		}
	}
	if id != "" {
		if e, ok := s.sessions[id]; ok {
			e.lastActive = now
			return id, e, nil
		}
		data, readErr := os.ReadFile(filepath.Join(s.saveDir, id+".json"))
		if readErr == nil {
			var g Game
			if json.Unmarshal(data, &g) == nil && validGame(&g) {
				g.Player.normalizeLegacyClassResources()
				g.Version = gameVersion // Old save files should display the current release.
				e := &entry{game: &g, lastActive: now}
				s.sessions[id] = e
				return id, e, nil
			}
		} else if !errors.Is(readErr, os.ErrNotExist) {
			return "", nil, readErr
		}
	}
	id, err = randomID()
	if err != nil {
		return "", nil, err
	}
	e := &entry{game: newGame(), lastActive: now}
	if err := s.save(id, e.game); err != nil {
		return "", nil, err
	}
	s.sessions[id] = e
	http.SetCookie(w, &http.Cookie{Name: "oppwars_session", Value: id, Path: "/", MaxAge: 30 * 24 * 60 * 60, Secure: r.TLS != nil, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	return id, e, nil
}
func validGame(g *Game) bool {
	switch g.Phase {
	case "setup", "hq", "event", "combat", "gameover":
	default:
		return false
	}
	if g.Phase == "combat" && g.Enemy == nil {
		return false
	}
	if g.Phase == "event" && g.Event == nil {
		return false
	}
	return true
}
func (s *server) save(id string, g *Game) error {
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.saveDir, ".save-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(data)
	}
	if err == nil {
		err = tmp.Sync()
	}
	closeErr := tmp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, filepath.Join(s.saveDir, id+".json"))
}
func jsonReply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && u.Host == r.Host && (u.Scheme == "http" || u.Scheme == "https")
}
func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/state", func(w http.ResponseWriter, r *http.Request) {
		_, e, err := s.getSession(w, r)
		if err != nil {
			jsonReply(w, 500, map[string]string{"error": "could not open save"})
			return
		}
		e.mu.Lock()
		defer e.mu.Unlock()
		jsonReply(w, 200, e.game)
	})
	mux.HandleFunc("POST /api/action", func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			jsonReply(w, 403, map[string]string{"error": "invalid origin"})
			return
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			jsonReply(w, 415, map[string]string{"error": "expected JSON"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var req struct {
			Action string `json:"action"`
			Name   string `json:"name"`
			Class  string `json:"class"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			jsonReply(w, 400, map[string]string{"error": "invalid request"})
			return
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			jsonReply(w, 400, map[string]string{"error": "unexpected JSON after request"})
			return
		}
		id, e, err := s.getSession(w, r)
		if err != nil {
			jsonReply(w, 500, map[string]string{"error": "could not open save"})
			return
		}
		e.mu.Lock()
		defer e.mu.Unlock()
		if err := e.game.act(req.Action, req.Name, req.Class); err != nil {
			jsonReply(w, 400, map[string]string{"error": err.Error()})
			return
		}
		if err := s.save(id, e.game); err != nil {
			log.Printf("save failed: %v", err)
			jsonReply(w, 500, map[string]string{"error": "could not save game"})
			return
		}
		jsonReply(w, 200, e.game)
	})
	assets, _ := fs.Sub(embeddedAssets, "static")
	mux.Handle("GET /", http.FileServer(http.FS(assets)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; base-uri 'none'; object-src 'none'; frame-ancestors 'none'")
		mux.ServeHTTP(w, r)
	})
}
func main() {
	addr := flag.String("listen", "127.0.0.1:8080", "HTTP listen address (localhost by default)")
	saveDir := flag.String("save-dir", "", "directory for browser save games")
	flag.Parse()
	if *saveDir == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			log.Fatal(err)
		}
		*saveDir = filepath.Join(dir, "oppwars-web", "saves")
	}
	app, err := newServer(*saveDir)
	if err != nil {
		log.Fatal(err)
	}
	httpServer := &http.Server{Addr: *addr, Handler: app.handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	fmt.Printf("OPP Wars Web Edition: http://%s\nSave games: %s\n", *addr, *saveDir)
	log.Fatal(httpServer.ListenAndServe())
}
