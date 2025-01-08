package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Map to store the recepit ID and the points for the recepit.
var pointsMap = make(map[string]int)

// Response for get points response.
type get_points_response struct {
	Points int `json:"points"`
}

// Response for the process response request
type process_recepits_reponse struct {
	Id     string `json:"id"`
	Points int    `json:"points"`
}

// Recepit items data in the process recepits request object
type recepit_items struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

// Process recepits request object
type process_recepits_request struct {
	Retailer     string          `json:"retailer"`
	PurchaseDate string          `json:"purchaseDate"`
	PurchaseTime string          `json:"purchaseTime"`
	Items        []recepit_items `json:"items"`
	Total        string          `json:"total"`
}

// Process a recepit and calculate the number of points that the recepit will receive
// The implementation here allows for bad data in and will not include them in the total point received.
func ProcessRecepitsHandler(w http.ResponseWriter, r *http.Request) {
	var recepit process_recepits_request
	if err := json.NewDecoder(r.Body).Decode(&recepit); err != nil {
		http.Error(w, "Invalid JSON"+err.Error(), http.StatusBadRequest)
		return
	}

	//create ID
	id := uuid.New().String()
	points := 0

	// One point for every alphanumeric character in the retailer name.
	retailer_alphanum_count := 0
	for _, char := range recepit.Retailer {
		if unicode.IsLetter(char) || unicode.IsNumber(char) {
			retailer_alphanum_count++
		}
	}
	points += retailer_alphanum_count

	// 50 points if the total is a round dollar amount with no cents.
	var totalSpend float64
	totalSpend, err := strconv.ParseFloat(recepit.Total, 64)
	if err == nil && totalSpend-float64(int64(totalSpend)) == 0 {
		points += 50
	}

	// Try to find if the decimal is divisible by 25. Step 1: isolate the decimal points through using casting to
	// remote the whole number portion from the float. Step 2: multiple 100 to make it integer and thus utilzie the % operator.
	if totalSpend != 0 && int(100*(totalSpend-float64(int64(totalSpend))))%25 == 0 {
		points += 25
	}

	// 5 points for every two items on the receipt.
	points += len(recepit.Items) / 2 * 5

	//f the trimmed length of the item description is a multiple of 3, multiply the price by 0.2
	//and round up to the nearest integer. The result is the number of points earned.
	for _, item := range recepit.Items {
		if len(strings.TrimSpace(item.ShortDescription))%3 == 0 {
			price, err := strconv.ParseFloat(item.Price, 64)
			if err != nil {
				continue
			}

			points += int(math.Ceil(price * 0.2))
		}
	}

	// 6 points if the day in the purchase date is odd.
	purchaseDate, err := time.Parse("2006-01-02", recepit.PurchaseDate)
	if err == nil && purchaseDate.Day()%2 == 1 {
		points += 6
	}

	// 10 points if the time of purchase is after 2:00pm and before 4:00pm.
	purchaseTime, err := time.Parse("15:04", recepit.PurchaseTime)
	if err == nil {
		twoPM, _ := time.Parse("15:04", "14:00")
		fourPM, _ := time.Parse("15:04", "16:00")
		if purchaseTime.After(twoPM) && purchaseTime.Before(fourPM) {
			points += 10
		}
	}

	// Assign points
	pointsMap[id] = points

	// Create response object
	var response = process_recepits_reponse{
		Id:     id,
		Points: points,
	}

	// Write response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Look up points for a given recepit id
func GetPointsHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	points := pointsMap[id]

	var response = get_points_response{
		Points: points,
	}

	json.NewEncoder(w).Encode(response)
}

func handleRequests() {

	r := mux.NewRouter()
	s := r.PathPrefix("/receipts").Subrouter()
	// "/receipts/process"
	s.HandleFunc("/process", ProcessRecepitsHandler)
	// "/receipts/{id}/points"
	s.HandleFunc("/{id}/points", GetPointsHandler)
	http.Handle("/", r)

	srv := &http.Server{
		Handler: r,
		Addr:    "127.0.0.1:8000",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func main() {
	handleRequests()
}
