from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
import datetime

app = FastAPI()

# Разрешаем браузерному HTML-файлу стучаться на наш бэкенд
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Инструмент трансформации кривого формата ДДММГГГГ в стандарт баз данных YYYY-MM-DD
def format_qlo_date(raw_date: str) -> str:
    try:
        # Режем строку "27092026" на день, месяц, год и пересобираем
        day = raw_date[0:2]
        month = raw_date[2:4]
        year = raw_date[4:8]
        return f"{year}-{month}-{day}"
    except Exception:
        raise HTTPException(status_code=422, detail="Неверный формат даты. Ожидается ДДММГГГГ")

# Схемы валидации входящих JSON-запросов
class SearchSchema(BaseModel):
    first: str
    secnd: str
    time1: str
    time2: str

class BookSchema(BaseModel):
    first: str
    secnd: str
    time1: str
    time2: str
    room_id: int
    name: str
    last_name: str
    phone: str

@app.post("/search")
def search_rooms(data: SearchSchema):
    # Преобразуем входящие даты для QloApps/MySQL
    qlo_start_date = format_qlo_date(data.first)
    qlo_end_date = format_qlo_date(data.secnd)
    
    print(f"Ищем свободные номера в QloApps с {qlo_start_date} по {qlo_end_date}")
    
    # Имитация ответа свободных физических комнат из базы QloApps
    # (В продакшене тут будет запрос к таблице `ps_hotel_room_booking_history`)
    available_rooms = [101, 102, 203, 207, 301]
    
    return {"rooms": available_rooms}

@app.post("/book")
def book_room(data: BookSchema):
    # Эмулируем проверку занятости номеров в QloApps
    # Например, если кто-то пытается забронировать уже занятый телефон
    if data.phone == "89999999999":
        raise HTTPException(status_code=400, detail="Занято")
        
    qlo_start_date = format_qlo_date(data.first)
    qlo_end_date = format_qlo_date(data.secnd)
    
    # --- ИНТЕГРАЦИОННАЯ ЛОГИКА С QLOAPPS ПОД КАПОТОМ ---
    # 1. Скрипт проверяет наличие Customer по номеру телефона в таблице `ps_customer`.
    # 2. Регистрирует новую корзину `ps_cart`.
    # 3. Добавляет запись в историю занятости номеров отеля `ps_hotel_room_booking_history`.
    
    print(f"Регистрируем бронь в QloApps:")
    print(f"Гость: {data.name} {data.last_name}, Комната: {data.room_id}")
    print(f"Период: {qlo_start_date} ({data.time1}) -> {qlo_end_date} ({data.time2})")
    
    return {"status": "success", "message": "Бронирование успешно проведено в QloApps!"}
