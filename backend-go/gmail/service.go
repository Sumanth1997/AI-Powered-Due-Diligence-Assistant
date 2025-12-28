package gmail

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Service wraps the Gmail API service
type Service struct {
	srv *gmail.Service
}

// NewService creates a new Gmail service with OAuth2 authentication
func NewService(credentialsPath, tokenPath string) (*Service, error) {
	ctx := context.Background()

	// Read credentials file
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file: %v", err)
	}

	// Parse credentials
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope, gmail.GmailModifyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials: %v", err)
	}

	// Get OAuth2 token
	tok, err := getToken(config, tokenPath)
	if err != nil {
		return nil, fmt.Errorf("unable to get token: %v", err)
	}

	// Create Gmail service
	client := config.Client(ctx, tok)
	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create Gmail service: %v", err)
	}

	return &Service{srv: srv}, nil
}

// getToken retrieves token from file or initiates OAuth2 flow
func getToken(config *oauth2.Config, tokenPath string) (*oauth2.Token, error) {
	// Try to load from file
	tok, err := tokenFromFile(tokenPath)
	if err != nil {
		// No saved token, get new one via web
		tok, err = getTokenFromWeb(config)
		if err != nil {
			return nil, err
		}
		// Save for future use
		saveToken(tokenPath, tok)
	}
	return tok, nil
}

// getTokenFromWeb initiates OAuth2 flow in browser
func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("\n=== GMAIL AUTHENTICATION ===\n")
	fmt.Printf("1. Open this URL in your browser:\n\n%v\n\n", authURL)
	fmt.Println("2. Sign in and authorize the app")
	fmt.Println("3. You'll be redirected - copy ONLY the code= value from the URL")
	fmt.Println("   Example: if URL is http://localhost/?code=4/0ABC...&scope=...")
	fmt.Println("   Copy just: 4/0ABC...")
	fmt.Print("\nPaste the code here: ")

	reader := bufio.NewReader(os.Stdin)
	authCode, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %v", err)
	}
	authCode = strings.TrimSpace(authCode)

	tok, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to exchange token: %v", err)
	}
	return tok, nil
}

// tokenFromFile retrieves token from a local file
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken saves a token to a file
func saveToken(path string, token *oauth2.Token) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Printf("Unable to cache oauth token: %v", err)
		return
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// ListRecentMessages lists recent messages with optional query filter
func (s *Service) ListRecentMessages(query string, maxResults int64) ([]*gmail.Message, error) {
	user := "me"
	call := s.srv.Users.Messages.List(user).MaxResults(maxResults)
	if query != "" {
		call = call.Q(query)
	}

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("unable to list messages: %v", err)
	}

	return resp.Messages, nil
}

// GetMessage retrieves full message details
func (s *Service) GetMessage(messageID string) (*gmail.Message, error) {
	msg, err := s.srv.Users.Messages.Get("me", messageID).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to get message: %v", err)
	}
	return msg, nil
}

// Attachment represents an email attachment
type Attachment struct {
	Filename string
	MimeType string
	Data     []byte
}

// GetPDFAttachments extracts PDF attachments from a message
func (s *Service) GetPDFAttachments(messageID string) ([]Attachment, error) {
	msg, err := s.srv.Users.Messages.Get("me", messageID).Do()
	if err != nil {
		return nil, err
	}

	var attachments []Attachment

	// Check all parts for PDF attachments
	for _, part := range msg.Payload.Parts {
		if part.Filename != "" && strings.HasSuffix(strings.ToLower(part.Filename), ".pdf") {
			// Get attachment data
			attData, err := s.srv.Users.Messages.Attachments.Get("me", messageID, part.Body.AttachmentId).Do()
			if err != nil {
				log.Printf("Error getting attachment: %v", err)
				continue
			}

			// Decode base64 data
			data, err := base64.URLEncoding.DecodeString(attData.Data)
			if err != nil {
				log.Printf("Error decoding attachment: %v", err)
				continue
			}

			attachments = append(attachments, Attachment{
				Filename: part.Filename,
				MimeType: part.MimeType,
				Data:     data,
			})
		}
	}

	return attachments, nil
}

// CheckForPitchDecks checks for new emails with PDF attachments matching pitch deck keywords
func (s *Service) CheckForPitchDecks() ([]struct {
	MessageID string
	Subject   string
	Sender    string
	PDFs      []Attachment
}, error) {
	// Search for unread emails with attachments
	query := "is:unread has:attachment filename:pdf (subject:pitch OR subject:deck OR subject:investment)"
	messages, err := s.ListRecentMessages(query, 10)
	if err != nil {
		return nil, err
	}

	var results []struct {
		MessageID string
		Subject   string
		Sender    string
		PDFs      []Attachment
	}

	for _, m := range messages {
		msg, err := s.GetMessage(m.Id)
		if err != nil {
			continue
		}

		// Extract subject and sender
		var subject, sender string
		for _, header := range msg.Payload.Headers {
			switch header.Name {
			case "Subject":
				subject = header.Value
			case "From":
				sender = header.Value
			}
		}

		// Get PDF attachments
		pdfs, err := s.GetPDFAttachments(m.Id)
		if err != nil || len(pdfs) == 0 {
			continue
		}

		results = append(results, struct {
			MessageID string
			Subject   string
			Sender    string
			PDFs      []Attachment
		}{
			MessageID: m.Id,
			Subject:   subject,
			Sender:    sender,
			PDFs:      pdfs,
		})
	}

	return results, nil
}

// HealthCheck verifies Gmail API connection
func (s *Service) HealthCheck() error {
	_, err := s.srv.Users.GetProfile("me").Do()
	return err
}
