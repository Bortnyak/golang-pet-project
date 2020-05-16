package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v2"
)

// link pattern for getting image
// const resultLnkPattern = "https://drive.google.com/uc?id="

// link pattern for download image
const resultLnkPattern = "https://docs.google.com/uc?export=download&id="

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
func startAfter(fullString string, startAfterStr string) string {
	pos := strings.LastIndex(fullString, startAfterStr)
	if pos == -1 {
		return ""
	}
	adjustedPos := pos + len(startAfterStr)

	if adjustedPos >= len(fullString) {
		return ""
	}
	return fullString[adjustedPos:len(fullString)]
}

func makeImageLink(childrenLink string, strPattern string) string {
	photoID := startAfter(childrenLink, "/children/")
	return strPattern + photoID
}

func makeDowloadImageLink(childrenLink string, strPattern string) string {
	fmt.Printf("childrenLink %s\n", childrenLink)
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

func downloadFile(url string) string {
	fmt.Println("Downloading file...")

	filename := startAfter(url, "download&id=") + ".jpg"
	f, err := os.Create(filename)
	if err != nil {
		fmt.Printf("create file: %v", err)
		panic(err)
	}
	defer f.Close()

	c := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	r, err := c.Get(url)
	if err != nil {
		fmt.Printf("Error while downloading %q: %v", url, err)
		panic(err)
	}
	defer r.Body.Close()

	fmt.Println("StatusCode: ", r.StatusCode)

	if r.StatusCode != 200 {
		fmt.Println("request throtled")
		return "throtled"
	}

	n, err := io.Copy(f, r.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(n, "bytes downloaded")

	return filename
}

func downloadWithTimeOut(url string) string {
	fileName := downloadFile(url)
	for fileName == "throtled" {
		timeNow := time.Now()
		fmt.Println("After this moment, sleep for 30 minutes", timeNow)
		time.Sleep(30 * time.Minute)

		return downloadWithTimeOut(url)
	}

	return fileName
}

// func downloadWithProxy(url string) string {
// 	fileName := downloadFile(url)
// 	println("fileName: ", fileName)

// 	for fileName == "throtled" {
// 		usedProxiesCounter++
// 		fmt.Println("usedProxiesCounter: ", usedProxiesCounter)

// 		timeNow := time.Now()
// 		fmt.Println("throtled time: ", timeNow)

// 		proxiesList := getProxies()
// 		proxyServer := proxiesList[usedProxiesCounter]
// 		fmt.Println("Next proxy server: ", proxyServer)

// 		os.Setenv("HTTPS_PROXY", "http://"+proxyServer)
// 		os.Setenv("HTTP_PROXY", "http://"+proxyServer)
// 		os.Setenv("http_proxy", "http://"+proxyServer)
// 		os.Setenv("https_proxy", "http://"+proxyServer)

// 		env := os.Getenv("HTTPS_PROXY")
// 		fmt.Println("env: ", env)

// 		downloadWithProxy(url)
// 	}
// 	return fileName
// }

// uploadFile
func uploadFile(filename string, targetURL string) string {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	// this step is very important
	fileWriter, err := bodyWriter.CreateFormFile("media", filename)
	if err != nil {
		fmt.Println("error writing to buffer")
		panic(err)
	}

	// open file handle
	fh, err := os.Open(filename)
	if err != nil {
		fmt.Println("error opening file")
		panic(err)
	}
	defer fh.Close()

	//iocopy
	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		panic(err)
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	resp, err := http.Post(targetURL, contentType, bodyBuf)
	if err != nil {
		panic(err)
	}

	fileLocation := resp.Header.Get("Location")
	defer resp.Body.Close()
	return "https://image001.pixtrek.com" + fileLocation
}

// Creates a new file upload http request with optional extra params
func newfileUploadRequest(uri string, paramName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	file.Close()
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, fi.Name())
	if err != nil {
		return nil, err
	}
	part.Write(fileContents)

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return http.NewRequest("POST", uri, body)
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
	f, err := os.Open("SABADIVA_Inventory_Management_Main _Alena (1).csv")
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
	var counter int32
	for i := 1; i < len(rows); i++ {
		folerdLink := rows[i][9]
		hasPrafixHttP := strings.HasPrefix(folerdLink, "http")

		if hasPrafixHttP != false {
			if counter > 0 && counter%50 == 0 {
				min := 10
				max := 40
				randInt := rand.Intn(max-min) + min
				fmt.Println("randomSeconds: ", randInt)
				time.Sleep(time.Duration(randInt) * time.Second)
			}

			folderID := strings.TrimPrefix(folerdLink, "https://drive.google.com/drive/folders/")
			resp, err := AllChildren(srv, folderID)
			if err != nil {
				fmt.Printf("err: %v", err)
			}

			fmt.Printf("string %+v\n", resp)
			var resultLink = ""

			for _, v := range resp {
				downloadURL := makeDowloadImageLink(v, resultLnkPattern)
				fmt.Printf("downloadURL %s\n\n\n", downloadURL)

				fileName := downloadWithTimeOut(downloadURL)

				counter++
				fmt.Printf("Success dowloads : %d\n", counter)

				path := "./" + fileName
				fmt.Printf("path: %s\n", path)

				var fileURL = ""
				fileURL = uploadFile(fileName, "https://image001.pixtrek.com/")
				resultLink += fileURL + " | "
			}

			rows[i][9] = resultLink
			fmt.Printf("resultLink: %s\n", resultLink)
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
	timeNow := time.Now()
	fmt.Printf("Script started: %s", timeNow)
	editedRows := appendLinks(rows, srv)
	writeChanges(editedRows)
	fmt.Println("Script finished!")
}
