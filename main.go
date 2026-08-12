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
	StatusFree   RoomStatus = "free"   // Светло-серый
	StatusBooked RoomStatus = "booked" // Темно-серый
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

// ---- НАСТРОЙКА SMTP ДЛЯ ОТПРАВКИ EMAIL ----
const (
	smtpHost     = "smtp.yandex.ru"            // Адрес SMTP вашего провайдера
	smtpPort     = "587"                       // Порт (обычно 587 или 465)
	senderEmail  = "your-hostel@yandex.ru"     // Ваша рабочая почта хостела
	senderPass   = "your-app-password-here"    // Пароль приложения (не обычный пароль!)
	managerEmail = "manager-hostel@yandex.ru"  // Почта управляющего для копий уведомлений
)

// Функция отправки письма
func sendEmailNotification(userEmail, roomNumber, roomType string) error {
	auth := smtp.PlainAuth("", senderEmail, senderPass, smtpHost)

	// Формируем красивое текстовое письмо в стиле минимализма
	subject := fmt.Sprintf("Subject: Успешное бронирование номера %s\n", roomNumber)
	mime := "MIME-version: 1.0;\nContent-Type: text/plain; charset=\"UTF-8\";\n\n"
	
	var body bytes.Buffer
	body.WriteString("Отель «Silence»\n")
	body.WriteString("-------------------------------------------\n")
	body.WriteString("Здравствуйте!\n\n")
	body.WriteString(fmt.Sprintf("Вы успешно забронировали номер %s (%s).\n", roomNumber, roomType))
	body.WriteString(fmt.Sprintf("Дата оформления: %s\n\n", time.Now().Format("02.01.2006 15:04")))
	body.WriteString("Ждем вас по адресу: ул. Мира, д. 10.\n")
	body.WriteString("-------------------------------------------\n")
	body.WriteString("Это автоматическое уведомление.")

	msg := []byte(subject + mime + body.String())
	recipients := []string{userEmail, managerEmail}

	// Отправляем асинхронно, чтобы не тормозить HTTP-ответ пользователю
	go func() {
		err := smtp.SendMail(smtpHost+":"+smtpPort, auth, senderEmail, recipients, msg)
		if err != nil {
			log.Printf("[Ошибка SMTP]: Не удалось отправить email: %v", err)
		} else {
			log.Printf("[Успех SMTP]: Письмо о брони номера %s отправлено на %s", roomNumber, userEmail)
		}
	}()

	return nil
}

func setupCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
// Handler 1: Отдача комнат для фронтенда
func GetRoomsHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w)
	if r.Method == "OPTIONS" { return }

	db.RLock()
	defer db.RUnlock()

	var list []Room
	for _, room := range db.Rooms {
		list = append(list, *room)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// Handler 2: Мгновенное бронирование и инициация отправки Email
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

	if req.UserEmail == "" {
		http.Error(w, "Email требуется для уведомления", http.StatusBadRequest)
		return
	}

	db.Lock()
	room, exists := db.Rooms[req.RoomID]

	// Защита от Race Conditions
	if !exists || room.Status != StatusFree {
		db.Unlock()
		http.Error(w, "Номер уже забронирован кем-то другим", http.StatusConflict)
		return
	}

	// Мгновенно переводим в статус занят (темно-серый)
	room.Status = StatusBooked
	
	newBooking := Booking{
		ID:        len(db.Bookings) + 1,
		RoomID:    req.RoomID,
		UserEmail: req.UserEmail,
		BookedAt:  time.Now(),
	}
	db.Bookings = append(db.Bookings, newBooking)
	db.Unlock()

	// Триггерим отправку Email
	sendEmailNotification(req.UserEmail, room.Number, room.Type)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "confirmed", "message": "Бронирование оформлено, чек отправлен на email"})
}

func main() {
	http.HandleFunc("/api/rooms", GetRoomsHandler)
	http.HandleFunc("/api/book", BookRoomHandler)

	fmt.Println("Сервер отеля «Silence» запущен на http://localhost:8000")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err)
	}
	
	// Маршруты для фронтенда (совпадают с index.html на 100%)
	http.HandleFunc("/api/rooms", GetRoomsHandler)
	http.HandleFunc("/api/book", BookRoomHandler) // Теперь адрес строго /api/book

	fmt.Println("Сервер отеля «Silence» запущен на http://localhost:8000")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err)
	}
}

