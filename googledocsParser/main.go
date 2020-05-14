package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v2"
)

const resultLnkPattern = "https://drive.google.com/uc?id="

// Retrieves a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Requests a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	defer f.Close()
	if err != nil {
		log.Fatalf("Unable to cache OAuth token: %v", err)
	}
	json.NewEncoder(f).Encode(token)
}

// Get substring after a string.
func startAfter(value string, a string) string {
	pos := strings.LastIndex(value, a)
	if pos == -1 {
		return ""
	}
	adjustedPos := pos + len(a)
	if adjustedPos >= len(value) {
		return ""
	}
	return value[adjustedPos:len(value)]
}

func makeImageLink(childrenLink string, strPattern string) string {
	photoID := startAfter(childrenLink, "/children/")
	return strPattern + photoID
}

func makeResultString(linkMap map[int]string) string {
	resultString := ""
	for _, v := range linkMap {
		resultString += makeImageLink(v, resultLnkPattern) + " | "
	}
	return resultString
}

// AllChildren fetches all the children of a given folder
// func AllChildren(d *drive.Service, folderID string) ([]*drive.ChildReference, error) {
func AllChildren(d *drive.Service, folderID string) (map[int]string, error) {
	cs := make(map[int]string)
	pageToken := ""
	for {
		q := d.Children.List(folderID)
		if pageToken != "" {
			q = q.PageToken(pageToken)
		}

		r, err := q.Do()
		if err != nil {
			fmt.Printf("An error occurred: %v\n", err)
			return nil, err
		}

		for i, v := range r.Items {
			cs[i] = v.SelfLink
		}

		pageToken = r.NextPageToken
		if pageToken == "" {
			break
		}
	}
	return cs, nil
}

func readSample() [][]string {
	f, err := os.Open("SABADIVA_Inventory_Management Main _Julia_(1).csv")
	if err != nil {
		log.Fatal(err)
	}
	rows, err := csv.NewReader(f).ReadAll()
	f.Close()
	if err != nil {
		log.Fatal(err)
	}
	return rows
}

func appendLinks(rows [][]string, srv *drive.Service) [][]string {
	for i := 1; i < len(rows); i++ {
		folerdLink := rows[i][9]
		hasPrafixHttP := strings.HasPrefix(folerdLink, "http")
		if hasPrafixHttP != false {
			folderID := strings.TrimPrefix(folerdLink, "https://drive.google.com/drive/folders/")

			resp, err := AllChildren(srv, folderID)
			if err != nil {
				fmt.Printf("err: %v", err)
			}

			resultLink := makeResultString(resp)
			fmt.Printf("resultLink %s\n\n", resultLink)
			rows[i][9] = resultLink
		}
	}

	return rows
}

func writeChanges(rows [][]string) {
	f, err := os.Create("OUTPUT_SABADIVA_Inventory_Management Main _Julia_(1).csv")
	if err != nil {
		log.Fatal(err)
	}
	err = csv.NewWriter(f).WriteAll(rows)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/drive https://www.googleapis.com/auth/drive.file https://www.googleapis.com/auth/drive.photos.readonly")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := drive.New(client)

	if err != nil {
		log.Fatalf("Unable to retrieve Docs client: %v", err)
	}

	rows := readSample()
	editedRows := appendLinks(rows, srv)
	writeChanges(editedRows)
}
