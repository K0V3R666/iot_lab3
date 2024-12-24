package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// PaymentRequest представляет запрос на обработку оплаты.
type PaymentRequest struct {
	ServiceID string    `json:"service_id"` // Идентификатор сервиса
	Method    string    `json:"method"`     // Метод сервиса
	From      time.Time `json:"from"`       // Начало периода оплаты
	To        time.Time `json:"to"`         // Конец периода оплаты
}

// PaymentResponse представляет ответ после обработки оплаты.
type PaymentResponse struct {
	Token  string    `json:"token"`  // Уникальный токен оплаты
	From   time.Time `json:"from"`   // Начало периода оплаты
	To     time.Time `json:"to"`     // Конец периода оплаты
	Method string    `json:"method"` // Метод сервиса
}

// ServiceRegistry хранит доступные сервисы и их методы.
type ServiceRegistry struct {
	services map[string]map[string]bool // Карта сервисов и их методов
	sync.RWMutex                        // Мьютекс для безопасного доступа к данным
}

// Создаем глобальный экземпляр регистра сервисов.
var registry = &ServiceRegistry{
	services: make(map[string]map[string]bool),
}

// RegisterService регистрирует сервис и его метод в регистре.
func (r *ServiceRegistry) RegisterService(serviceID, method string) {
	r.Lock() // Блокируем запись в регистр
	defer r.Unlock()

	// Если сервис еще не зарегистрирован, создаем для него пустую карту методов.
	if _, exists := r.services[serviceID]; !exists {
		r.services[serviceID] = make(map[string]bool)
	}

	// Регистрируем метод для сервиса.
	r.services[serviceID][method] = true
}

// IsServiceAvailable проверяет, доступен ли запрашиваемый сервис и метод.
func (r *ServiceRegistry) IsServiceAvailable(serviceID, method string) bool {
	r.RLock() // Блокируем чтение из регистра
	defer r.RUnlock()

	// Проверяем, существует ли сервис и его метод.
	if methods, exists := r.services[serviceID]; exists {
		return methods[method]
	}
	return false
}

// generateToken генерирует уникальный токен для оплаты.
func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b) // Генерируем случайные байты
	return base64.URLEncoding.EncodeToString(b) // Кодируем в Base64
}

// handlePayment обрабатывает запрос на оплату.
func handlePayment(w http.ResponseWriter, r *http.Request) {
	// Декодируем тело запроса в структуру PaymentRequest.
	var req PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный запрос", http.StatusBadRequest)
		return
	}
	defer r.Body.Close() // Закрываем тело запроса после обработки

	// Проверяем, доступен ли запрашиваемый сервис и метод.
	if !registry.IsServiceAvailable(req.ServiceID, req.Method) {
		http.Error(w, "Сервис или метод не найден", http.StatusNotFound)
		return
	}

	// Генерируем уникальный токен для оплаты.
	token := generateToken()

	// Формируем ответ с токеном и данными об оплате.
	response := PaymentResponse{
		Token:  token,
		From:   req.From,
		To:     req.To,
		Method: req.Method,
	}

	// Устанавливаем заголовок Content-Type и отправляем ответ в формате JSON.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Регистрируем несколько сервисов и методов для демонстрации.
	registry.RegisterService("service1", "method1")
	registry.RegisterService("service1", "method2")
	registry.RegisterService("service2", "method1")

	// Регистрируем обработчик для маршрута /payment.
	http.HandleFunc("/payment", handlePayment)

	// Запускаем HTTP-сервер на порту 8080.
	fmt.Println("Запуск сервиса оплаты на :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}