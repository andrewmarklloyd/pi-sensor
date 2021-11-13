package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
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
		return "", err
	}

	// var bucket drive.File
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

		fmt.Println(fmt.Sprintf("Cold storage bucket %s already exists with backup file %s, syncing now", bucket.Name, backupFileName))

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

func configClient() *drive.Service {
	ctx := context.Background()
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}
	return srv
}

func main() {
	app := "pi-sensor-staging"
	bucketName := fmt.Sprintf("backup-%s", app)
	backupFileName := "cold-storage.csv"
	syncDir := fmt.Sprintf("/tmp/%s", bucketName)
	setupDir(syncDir)
	tmpWorkDir := "/tmp/data"
	setupDir(tmpWorkDir)

	srv := configClient()
	backupFileId, err := getOrCreateBackupFile(srv, bucketName, backupFileName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(backupFileId)

	time.Sleep(time.Second * 5)
	dstFile := &drive.File{}
	f, err := os.Open(fmt.Sprintf("/tmp/%s/%s", bucketName, backupFileName))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	_, err = srv.Files.Update(backupFileId, dstFile).Media(f).Do()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
