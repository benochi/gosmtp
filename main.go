package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"strings"
)

const (
	httpAddr = ":8080"                          // Address for the API server
	apiKey   = "DansSuperSecretKEY1142*&&51123" // Your secret API key for authentication
)

// Email represents the basic parts of an email message
type Email struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}

func main() {
	http.HandleFunc("/send", apiAuthMiddleware(sendEmailHandler))
	log.Printf("HTTP server starting on %s\n", httpAddr)
	if err := http.ListenAndServe(httpAddr, nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func sendEmailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("X-API-Key") != apiKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var email Email
	if err := json.NewDecoder(r.Body).Decode(&email); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	for _, recipient := range email.To {
		if err := sendDirectEmail(email, recipient); err != nil {
			log.Printf("Failed to send email to %s: %v", recipient, err)
			http.Error(w, fmt.Sprintf("Failed to send email to %s", recipient), http.StatusInternalServerError)
			return
		}
	}

	log.Printf("Email sent successfully to all recipients")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Email sent successfully"))
}

func sendDirectEmail(email Email, recipient string) error {
	// Extract recipient's domain
	domain := strings.Split(recipient, "@")[1]
	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		return err
	}
	if len(mxRecords) == 0 {
		return fmt.Errorf("no MX records found for domain: %s", domain)
	}

	// Connect to the recipient's SMTP server
	smtpServer := mxRecords[0].Host
	conn, err := smtp.Dial(smtpServer + ":25")
	if err != nil {
		return err
	}
	defer conn.Close()

	// Set the sender and recipient
	if err := conn.Mail(email.From); err != nil {
		return err
	}
	if err := conn.Rcpt(recipient); err != nil {
		return err
	}

	// Send the email body
	wc, err := conn.Data()
	if err != nil {
		return err
	}
	defer wc.Close()

	buf := bufio.NewWriter(wc)
	if _, err = buf.WriteString(fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: %s\r\n\r\n%s", recipient, email.From, email.Subject, email.Body)); err != nil {
		return err
	}
	if err = buf.Flush(); err != nil {
		return err
	}

	return nil
}

func apiAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next(w, r)
	}
}
