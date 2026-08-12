package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type RoomStatus string

const (
	StatusFree   RoomStatus = "free"   // Светло-серый цвет на фронтенде
	StatusBooked RoomStatus = "booked" // Темно-серый цвет на фронтенде
)

// Структура номера отеля
type Room struct {
	ID     int        `json:"id"`
	Number string     `json:"number"`
	Type   string     `json:"type"`
	Status RoomStatus `json:"status"`
}

// Структура оформленной брони
type Booking struct {
	ID        int       `json:"id"`
	RoomID    int       `json:"room_id"`
	UserEmail string    `json:"user_email"`
	BookedAt  time.Time `json:"booked_at"`
}

// Потокобезопасная база данных в оперативной памяти (Защита от Race Conditions)
type Store struct {
	sync.RWMutex
	Rooms    map[int]*Room
	Bookings []Booking
}

// Инициализируем пример этажа (номера 202, 205, 207 заняты по умолчанию)
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

// Асинхронный триггер для Email-нотификации
func sendEmailNotification(userEmail, roomNumber string) {
	// Демо-заглушка: выводит лог отправки прямо в терминал вашего localhost
	log.Printf("[SMTP LOCALHOST]: Отправка подтверждения на %s. Номер %s успешно забронирован.", userEmail, roomNumber)
}

// Включение CORS-заголовков, чтобы локальный файл index.html мог общаться с бэкендом
func setupCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// Получение комнат для сетки фронтенда
func GetRoomsHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w)
	if r.Method == "OPTIONS" {
		return
	}

	db.RLock()
	defer db.RUnlock()

	var list []Room = make([]Room, 0)
	for _, room := range db.Rooms {
		list = append(list, *room)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// Обработка кнопки «Забронировать»
func BookRoomHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w)
	if r.Method == "OPTIONS" {
		return
	}
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
		http.Error(w, "Номер уже занят", http.StatusConflict)
		return
	}

	// Переводим статус в занят (комната сразу станет темно-серой для всех)
	room.Status = StatusBooked
	db.Bookings = append(db.Bookings, Booking{
		ID:        len(db.Bookings) + 1,
		RoomID:    req.RoomID,
		UserEmail: req.UserEmail,
		BookedAt:  time.Now(),
	})
	db.Unlock()

	// Запуск отправки письма
	sendEmailNotification(req.UserEmail, room.Number)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status":"confirmed"}`))
}

// Заглушка для главной страницы localhost:8000
func RootHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<h2>Локальный сервер отеля запущен успешно.</h2><p>Данные доступны по адресу: <a href="/api/rooms">/api/rooms</a></p>`))
}

func main() {
	// Маршруты и хвосты ссылок для локальной работы
	http.HandleFunc("/", RootHandler)
	http.HandleFunc("/api/rooms", GetRoomsHandler)
	http.HandleFunc("/api/book", BookRoomHandler)

	log.Println("Локальный сервер запущен на http://localhost:8000")
	// На localhost слушаем внутренний интерфейс 127.0.0.1
	if err := http.ListenAndServe("127.0.0.1:8000", nil); err != nil {
		log.Fatal(err)
	}
}
