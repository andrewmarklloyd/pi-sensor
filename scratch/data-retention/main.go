package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/prasmussen/gdrive/auth"
	gdrive "github.com/prasmussen/gdrive/drive"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const ClientId = ""
const ClientSecret = ""

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

func GetClientWithBucket(bucketName, syncDir string) (*drive.Service, drive.File, *gdrive.Drive) {
	ctx := context.Background()
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	r, err := srv.Files.List().Q(fmt.Sprintf("name = '%s'", bucketName)).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}

	tokenPath := "token.json"
	oauth, err := auth.NewFileSourceClient(ClientId, ClientSecret, tokenPath, authCodePrompt)

	if err != nil {
		fmt.Println("Failed getting oauth client:", err.Error())
		os.Exit(1)
	}
	dr, err := gdrive.New(oauth)
	if err != nil {
		fmt.Println("Failed getting drive:", err.Error())
		os.Exit(1)
	}

	var bucket drive.File
	if len(r.Files) == 0 {
		fmt.Println("Cold storage bucket does not exist, creating it now")

		dr.Upload(gdrive.UploadArgs{
			Out:       os.Stdout,
			Recursive: true,
			Path:      syncDir,
		})

		if err != nil {
			log.Fatal("Could not create dir: " + err.Error())
		}

		r, err = srv.Files.List().Q(fmt.Sprintf("name = '%s'", bucketName)).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve files: %v", err)
		}

		dr.UploadSync(gdrive.UploadSyncArgs{
			Out:    os.Stdout,
			Path:   syncDir,
			RootId: r.Files[0].Id,
		})
	} else if len(r.Files) == 1 {
		fmt.Println("Cold storage bucket already exists, syncing now")
		bucket = *r.Files[0]
		dr.DownloadSync(gdrive.DownloadSyncArgs{
			Out:    os.Stdout,
			Path:   syncDir,
			RootId: bucket.Id,
		})
	} else {
		log.Fatal("Only one bucket expected but found more than one")
	}
	return srv, bucket, dr
}

func authCodePrompt(url string) func() string {
	return func() string {
		fmt.Println("Authentication needed")
		fmt.Println("Go to the following url in your browser:")
		fmt.Printf("%s\n\n", url)
		fmt.Print("Enter verification code: ")

		var code string
		if _, err := fmt.Scan(&code); err != nil {
			fmt.Printf("Failed reading code: %s", err.Error())
		}
		return code
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

func main() {
	app := "pi-sensor-staging"
	bucketName := fmt.Sprintf("backup-%s", app)
	syncDir := fmt.Sprintf("/tmp/%s", bucketName)
	setupDir(syncDir)
	tmpWorkDir := "/tmp/data"
	setupDir(tmpWorkDir)
	_, bucket, dr := GetClientWithBucket(bucketName, syncDir)
	// GetClientWithBucket(bucketName, syncDir)
	keepFile := []byte("testing-abc")
	filename := fmt.Sprintf("%s/testing-abc", syncDir)
	os.WriteFile(filename, keepFile, 0644)
	dr.UploadSync(gdrive.UploadSyncArgs{
		Out:    os.Stdout,
		Path:   syncDir,
		RootId: bucket.Id,
	})
}
