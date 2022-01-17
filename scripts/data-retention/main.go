package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type Config struct {
	AppName     string
	MaxRows     int
	DatabaseURL string
	Service     *drive.Service
	Token       oauth2.Token
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) (*http.Client, error) {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokenJson := os.Getenv("TOKEN_JSON")
	if tokenJson != "" {
		r := strings.NewReader(tokenJson)
		tok := &oauth2.Token{}
		err := json.NewDecoder(r).Decode(tok)
		if err != nil {
			return nil, err
		}
		return config.Client(context.Background(), tok), nil
	} else {
		fmt.Println("TOKEN_JSON not set, looking for token.json")
		tokFile := "token.json"
		tok, err := tokenFromFile(tokFile)
		if err != nil {
			tok = getTokenFromWeb(config)
			saveToken(tokFile, tok)
		}
		return config.Client(context.Background(), tok), nil
	}
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
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

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getOrCreateBackupFile(srv *drive.Service, bucketName, backupFileName string) (string, error) {
	localBackupFilePath := fmt.Sprintf("/tmp/%s/%s", bucketName, backupFileName)
	r, err := srv.Files.List().Q(fmt.Sprintf("name = '%s'", bucketName)).Do()
	if err != nil {
		return "", fmt.Errorf("searching for bucket: %s", err)
	}

	if len(r.Files) == 0 {
		fmt.Println("Cold storage bucket does not exist, creating it now")

		bucket, err := srv.Files.Create(&drive.File{
			Name:     bucketName,
			MimeType: "application/vnd.google-apps.folder",
		}).Do()

		if err != nil {
			return "", err
		}

		backupFile, err := srv.Files.Create(&drive.File{
			Name:     backupFileName,
			Parents:  []string{bucket.Id},
			MimeType: "text/csv",
		}).Do()

		if err != nil {
			return "", err
		}
		resp, err := srv.Files.Get(backupFile.Id).Download()
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		err = ioutil.WriteFile(localBackupFilePath, body, 0644)
		return backupFile.Id, nil

	} else if len(r.Files) == 1 {
		bucket := *r.Files[0]
		backupFileRes, _ := srv.Files.List().Q(fmt.Sprintf("'%s' in parents", bucket.Id)).Do()
		backupFile := *&backupFileRes.Files[0]
		if len(backupFileRes.Files) != 1 || *&backupFileRes.Files[0].Name != backupFileName {
			return "", fmt.Errorf("Expected cold storage bucket %s to contain file %s", bucket.Name, backupFileName)
		}

		fmt.Println(fmt.Sprintf("Cold storage bucket %s already exists with backup filename %s, syncing now", bucket.Name, backupFileName))

		resp, err := srv.Files.Get(backupFile.Id).Download()
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		err = ioutil.WriteFile(localBackupFilePath, body, 0644)
		return backupFile.Id, nil
	} else {
		return "", fmt.Errorf("Only one bucket expected but found more than one")
	}
}

func setupDir(syncDir string) {
	err := os.RemoveAll(syncDir)
	if err != nil {
		log.Fatal(err)
	}
	err = os.Mkdir(syncDir, 0755)
	if err != nil {
		log.Fatal(err)
	}
}

func configClient() (*drive.Service, oauth2.Token) {
	ctx := context.Background()

	// try env var, then file on disk
	var b []byte
	credsString := os.Getenv("CREDENTIALS_JSON")
	if credsString != "" {
		b = []byte(credsString)
	} else {
		fmt.Println("CREDENTIALS_JSON not set, looking for credentials.json")
		var err error
		b, err = ioutil.ReadFile("credentials.json")
		if err != nil {
			log.Fatalf("Unable to read client secret file: %v", err)
		}
	}

	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client, err := getClient(config)
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	newToken, err := client.Transport.(*oauth2.Transport).Source.Token()
	if err != nil {
		log.Fatalf("Unalbe to get token from client: %v", err)
	}

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to create new Drive service: %v", err)
	}
	return srv, *newToken
}

func writeBackupFile(messages []Message, filepath string) error {
	file, _ := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	datawriter := bufio.NewWriter(file)
	for _, data := range messages {
		_, err := datawriter.WriteString(fmt.Sprintf("%s,%s,%s\n", data.Source, data.Status, data.Timestamp))
		if err != nil {
			return err
		}
	}
	datawriter.Flush()
	defer file.Close()
	return nil
}

func readMessagesFromBackupFile(filepath string) ([]Message, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	messages := make([]Message, 0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		s := strings.Split(scanner.Text(), ",")
		m := Message{
			Source:    s[0],
			Status:    s[1],
			Timestamp: s[2],
		}
		messages = append(messages, m)
	}

	return messages, nil
}

func ensureBackupMessagesSorted(fullBackupMessages []Message) {
	sort.SliceStable(fullBackupMessages, func(i, j int) bool {
		return fullBackupMessages[i].Timestamp < fullBackupMessages[j].Timestamp
	})
}

func uploadBackupFile(srv *drive.Service, backupFilePath string, backupFileId string) error {
	dstFile := &drive.File{}
	f, err := os.Open(backupFilePath)
	if err != nil {
		return err
	}
	_, err = srv.Files.Update(backupFileId, dstFile).Media(f).Do()
	if err != nil {
		return err
	}
	return nil
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() Config {
	appName := os.Getenv("APP_NAME")
	if appName == "" {
		fmt.Println("APP_NAME env var not set")
		os.Exit(1)
	}

	maxRowsString := os.Getenv("MAX_ROWS")
	if maxRowsString == "" {
		fmt.Println("MAX_ROWS env var not set")
		os.Exit(1)
	}
	maxRows, _ := strconv.Atoi(maxRowsString)

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		fmt.Println("DATABASE_URL env var not set")
		os.Exit(1)
	}

	srv, token := configClient()
	return Config{
		AppName:     appName,
		MaxRows:     maxRows,
		DatabaseURL: dbUrl,
		Service:     srv,
		Token:       token,
	}
}

func main() {
	defaultCommands := []string{"refresh-token", "read-only", "trim"}
	command := flag.String("command", "", fmt.Sprintf("The command to run. Available commands: %s", defaultCommands))
	flag.Parse()

	if *command != "refresh-token" && *command != "read-only" && *command != "trim" {
		fmt.Println(fmt.Sprintf("Command not recognized: %s. Available commands: %s", *command, defaultCommands))
		os.Exit(1)
	}

	config := initConfig()

	if *command == "refresh-token" {
		c := NewHerokuClient(config.AppName, os.Getenv("HEROKU_API_KEY"))
		err := c.UpdateToken(config.Token)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	bucketName := fmt.Sprintf("backup-%s", config.AppName)
	backupFileName := "cold-storage.csv"
	syncDir := fmt.Sprintf("/tmp/%s", bucketName)
	setupDir(syncDir)

	c, err := newPostgresClient(config.DatabaseURL)
	checkErr(err)
	rowsAboveMax, _ := c.getRowsAboveMax(config.MaxRows)
	numberRowsAboveMax := len(rowsAboveMax)
	if numberRowsAboveMax == 0 {
		os.Exit(0)
	}
	if *command == "read-only" {
		fmt.Println("Read only mode")
		os.Exit(0)
	}

	backupFileId, err := getOrCreateBackupFile(config.Service, bucketName, backupFileName)
	checkErr(err)

	localBackupFilePath := fmt.Sprintf("/%s/%s", syncDir, backupFileName)
	backupMessages, err := readMessagesFromBackupFile(localBackupFilePath)
	checkErr(err)
	preUpdateRowCount := len(backupMessages)

	fullBackupMessages := append(backupMessages, rowsAboveMax...)
	ensureBackupMessagesSorted(fullBackupMessages)
	err = writeBackupFile(fullBackupMessages, localBackupFilePath)
	checkErr(err)
	postUpdateRowCount := len(fullBackupMessages)

	numBackupUpdated := postUpdateRowCount - preUpdateRowCount
	if numBackupUpdated != numberRowsAboveMax {
		fmt.Println(fmt.Sprintf("WARN: number of messages updated in backup file '%d' did not match expected number of rows above max '%d'", numBackupUpdated, numberRowsAboveMax))
	}
	err = uploadBackupFile(config.Service, fmt.Sprintf("/tmp/%s/%s", bucketName, backupFileName), backupFileId)
	checkErr(err)
	err = c.deleteRows(rowsAboveMax)
	checkErr(err)
}
