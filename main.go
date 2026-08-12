package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"sync"
	"time"
)

type RoomStatus string

const (
	StatusFree   RoomStatus = "free"
	StatusBooked RoomStatus = "booked"
)

type Room struct {
	ID     int        `json:"id"`
	Number string     `json:"number"`
	Type   string     `json:"type"`
	Status RoomStatus `json:"status"`
}

type Booking struct {
	ID        int       `json:"id"`
	RoomID    int       `json:"room_id"`
	UserEmail string    `json:"user_email"`
	BookedAt  time.Time `json:"booked_at"`
}

type Store struct {
	sync.RWMutex
	Rooms    map[int]*Room
	Bookings []Booking
}

var db = &Store{
	Rooms: map[int]*Room{
		201: {ID: 201, Number: "201", Type: "Standard", Status: StatusFree},
		202: {ID: 202, Number: "202", Type: "Standard", Status: StatusBooked},
		203: {ID: 203, Number: "203", Type: "Studio Suite", Status: StatusFree},
		204: {ID: 204, Number: "204", Type: "Deluxe Room", Status: StatusFree},
		205: {ID: 205, Number: "205", Type: "Standard", Status: StatusBooked},
		206: {ID: 206, Number: "206", Type: "Studio Suite", Status: StatusFree},
		207: {ID: 207, Number: "207", Type: "Deluxe Room", Status: StatusBooked},
		208: {ID: 208, Number: "208", Type: "Penthouse", Status: StatusFree},
	},
}

func sendEmailNotification(userEmail, roomNumber, roomType string) {
	// Демо-заглушка. Письмо логируется в терминал сервера, чтобы не вызывать зависаний
	log.Printf("[Имитация SMTP]: Отправка письма на %s. Номер %s забронирован.", userEmail, roomNumber)
	
	// Если захотите включить настоящую отправку, раскомментируйте код ниже:
	/*
	auth := smtp.PlainAuth("", "user@yandex.ru", "password", "smtp.yandex.ru")
	msg := []byte("Subject: Бронь\n\nНомер забронирован.")
	smtp.SendMail("smtp.yandex.ru:587", auth, "user@yandex.ru", []string{userEmail}, msg)
	*/
}

func setupCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
func GetRoomsHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w)
	if r.Method == "OPTIONS" { return }

	db.RLock()
	defer db.RUnlock()

	var list []Room = make([]Room, 0)
	for _, room := range db.Rooms {
		list = append(list, *room)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func BookRoomHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w)
	if r.Method == "OPTIONS" { return }
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RoomID    int    `json:"room_id"`
		UserEmail string `json:"user_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	db.Lock()
	room, exists := db.Rooms[req.RoomID]
	if !exists || room.Status != StatusFree {
		db.Unlock()
		http.Error(w, "Номер занят", http.StatusConflict)
		return
	}

	room.Status = StatusBooked
	db.Bookings = append(db.Bookings, Booking{
		ID:        len(db.Bookings) + 1,
		RoomID:    req.RoomID,
		UserEmail: req.UserEmail,
		BookedAt:  time.Now(),
	})
	db.Unlock()

	sendEmailNotification(req.UserEmail, room.Number, room.Type)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status":"confirmed"}`))
}

// Хендлер для главной страницы, чтобы убрать ошибку 404 page not found
func RootHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<h1>Сервер отеля «Silence» запущен успешно!</h1><p>Для просмотра комнат перейдите на <a href="/api/rooms">/api/rooms</a></p>`))
}

func main() {
	http.HandleFunc("/", RootHandler) // Теперь на главной странице будет красивое уведомление вместо 404
	http.HandleFunc("/api/rooms", GetRoomsHandler)
	http.HandleFunc("/api/book", BookRoomHandler)

	log.Println("Сервер запущен на порту :8000")
	if err := http.ListenAndServe("0.0.0.0:8000", nil); err != nil {
		log.Fatal(err)
	}
}

