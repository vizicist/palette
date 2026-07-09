package kit

// Uploading recordings to YouTube, using only the standard library.
//
// One-time setup:
//  1. In Google Cloud Console, create an OAuth client of type
//     "TVs and Limited Input devices" and enable the YouTube Data API v3.
//  2. palette env set YOUTUBE_CLIENT_ID {client id}
//     palette env set YOUTUBE_CLIENT_SECRET {client secret}
//  3. palette youtube auth   (visit the printed URL, enter the code)
// That stores YOUTUBE_REFRESH_TOKEN in the env file; uploads then work
// from the web UI's recordings page or "palette record upload {name}".
//
// Optional env values: YOUTUBE_PRIVACY (default "unlisted"),
// YOUTUBE_DESCRIPTION (default empty).

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	json "github.com/goccy/go-json"
	"github.com/joho/godotenv"
)

const (
	youtubeTokenURL      = "https://oauth2.googleapis.com/token"
	youtubeDeviceCodeURL = "https://oauth2.googleapis.com/device/code"
	youtubeUploadURL     = "https://www.googleapis.com/upload/youtube/v3/videos?uploadType=resumable&part=snippet,status"
	youtubeUploadScope   = "https://www.googleapis.com/auth/youtube.upload"
)

// YouTubeUploadState is the upload progress pushed to the web UI as part
// of the obsrecord snapshot.
type YouTubeUploadState struct {
	File  string `json:"file"`
	State string `json:"state"` // "uploading", "done", or "error"
	URL   string `json:"url,omitempty"`
	Error string `json:"error,omitempty"`
}

var (
	youtubeUploadMu    sync.Mutex
	youtubeUploadState *YouTubeUploadState
)

// YouTubeUploadSnapshot returns a copy of the current upload state,
// or nil if no upload has been started.
func YouTubeUploadSnapshot() *YouTubeUploadState {
	youtubeUploadMu.Lock()
	defer youtubeUploadMu.Unlock()
	if youtubeUploadState == nil {
		return nil
	}
	state := *youtubeUploadState
	return &state
}

func setYouTubeUploadState(state *YouTubeUploadState) {
	youtubeUploadMu.Lock()
	youtubeUploadState = state
	youtubeUploadMu.Unlock()
	NotifyOBSRecordChanged()
}

// YouTubeConfigured reports whether upload credentials are present.
func YouTubeConfigured() bool {
	return EnvLookup("YOUTUBE_CLIENT_ID") != "" &&
		EnvLookup("YOUTUBE_CLIENT_SECRET") != "" &&
		EnvLookup("YOUTUBE_REFRESH_TOKEN") != ""
}

// StartYouTubeUpload begins uploading a recording in the background.
// Progress is pushed to the web UI via the obsrecord snapshot.
func StartYouTubeUpload(filename string) error {
	if !YouTubeConfigured() {
		return fmt.Errorf("YouTube is not configured, see 'palette youtube auth'")
	}
	path, err := recordingPath(filename)
	if err != nil {
		return err
	}

	youtubeUploadMu.Lock()
	if youtubeUploadState != nil && youtubeUploadState.State == "uploading" {
		inProgress := youtubeUploadState.File
		youtubeUploadMu.Unlock()
		return fmt.Errorf("already uploading %s", inProgress)
	}
	youtubeUploadState = &YouTubeUploadState{File: filename, State: "uploading"}
	youtubeUploadMu.Unlock()
	NotifyOBSRecordChanged()

	go func() {
		title := NameTitle(strings.TrimSuffix(filename, filepath.Ext(filename)))
		videoURL, err := youtubeUpload(path, title)
		if err != nil {
			LogError(fmt.Errorf("YouTube upload failed: %w", err), "file", filename)
			setYouTubeUploadState(&YouTubeUploadState{File: filename, State: "error", Error: err.Error()})
			return
		}
		LogInfo("YouTube upload complete", "file", filename, "url", videoURL)
		setYouTubeUploadState(&YouTubeUploadState{File: filename, State: "done", URL: videoURL})
	}()
	return nil
}

// youtubeUpload performs a resumable upload and returns the video URL.
func youtubeUpload(path string, title string) (string, error) {

	accessToken, err := youtubeAccessToken()
	if err != nil {
		return "", err
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}

	privacy := EnvLookup("YOUTUBE_PRIVACY")
	if privacy == "" {
		privacy = "unlisted"
	}
	metadata := map[string]any{
		"snippet": map[string]any{
			"title":       title,
			"description": EnvLookup("YOUTUBE_DESCRIPTION"),
		},
		"status": map[string]any{
			"privacyStatus":           privacy,
			"selfDeclaredMadeForKids": false,
		},
	}
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}

	// Step 1: request an upload session; its URL comes back in Location.
	req, err := http.NewRequest("POST", youtubeUploadURL, strings.NewReader(string(metadataBytes)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("X-Upload-Content-Type", "video/mp4")
	req.Header.Set("X-Upload-Content-Length", fmt.Sprintf("%d", info.Size()))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("YouTube upload session request failed: status=%d body=%s", resp.StatusCode, string(body))
	}
	sessionURL := resp.Header.Get("Location")
	if sessionURL == "" {
		return "", fmt.Errorf("YouTube upload session response has no Location header")
	}

	// Step 2: send the file bytes. No client timeout — a long recording
	// on a slow uplink can legitimately take many minutes.
	req, err = http.NewRequest("PUT", sessionURL, file)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "video/mp4")
	req.ContentLength = info.Size()

	resp, err = (&http.Client{}).Do(req)
	if err != nil {
		return "", err
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("YouTube upload failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var video struct {
		Id string `json:"id"`
	}
	if err := json.Unmarshal(body, &video); err != nil || video.Id == "" {
		return "", fmt.Errorf("YouTube upload succeeded but response has no video id: %s", string(body))
	}
	return "https://youtu.be/" + video.Id, nil
}

// youtubeAccessToken exchanges the stored refresh token for an access token.
func youtubeAccessToken() (string, error) {
	form := url.Values{
		"client_id":     {EnvLookup("YOUTUBE_CLIENT_ID")},
		"client_secret": {EnvLookup("YOUTUBE_CLIENT_SECRET")},
		"refresh_token": {EnvLookup("YOUTUBE_REFRESH_TOKEN")},
		"grant_type":    {"refresh_token"},
	}
	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := youtubePostForm(youtubeTokenURL, form, &result); err != nil {
		return "", err
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("YouTube token refresh failed: %s %s", result.Error, result.ErrorDesc)
	}
	return result.AccessToken, nil
}

// YouTubeDeviceAuth runs the one-time OAuth device flow.  The prompt
// callback receives the URL and code the user must enter; the function
// then blocks polling until the user approves (or the code expires),
// and stores the refresh token in the env file.
func YouTubeDeviceAuth(prompt func(verificationURL, userCode string)) error {

	clientID := EnvLookup("YOUTUBE_CLIENT_ID")
	clientSecret := EnvLookup("YOUTUBE_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("YOUTUBE_CLIENT_ID and YOUTUBE_CLIENT_SECRET must be set first, use 'palette env set'")
	}

	var device struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURL string `json:"verification_url"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
		Error           string `json:"error"`
	}
	form := url.Values{"client_id": {clientID}, "scope": {youtubeUploadScope}}
	if err := youtubePostForm(youtubeDeviceCodeURL, form, &device); err != nil {
		return err
	}
	if device.DeviceCode == "" {
		return fmt.Errorf("device code request failed: %s", device.Error)
	}
	prompt(device.VerificationURL, device.UserCode)

	interval := time.Duration(device.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(device.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		time.Sleep(interval)

		var token struct {
			RefreshToken string `json:"refresh_token"`
			Error        string `json:"error"`
		}
		form := url.Values{
			"client_id":     {clientID},
			"client_secret": {clientSecret},
			"device_code":   {device.DeviceCode},
			"grant_type":    {"urn:ietf:params:oauth:grant-type:device_code"},
		}
		if err := youtubePostForm(youtubeTokenURL, form, &token); err != nil {
			return err
		}
		switch {
		case token.RefreshToken != "":
			return saveEnvValue("YOUTUBE_REFRESH_TOKEN", token.RefreshToken)
		case token.Error == "authorization_pending":
			// keep polling
		case token.Error == "slow_down":
			interval += 2 * time.Second
		default:
			return fmt.Errorf("authorization failed: %s", token.Error)
		}
	}
	return fmt.Errorf("authorization code expired before it was entered")
}

func youtubePostForm(postURL string, form url.Values, result any) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.PostForm(postURL, form)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

// saveEnvValue writes one value to the env file (.palette/.env),
// preserving the other values in it.
func saveEnvValue(key, value string) error {
	path := EnvFilePath()
	myenv, err := godotenv.Read(path)
	if err != nil {
		myenv = map[string]string{}
	}
	myenv[key] = value
	return godotenv.Write(myenv, path)
}
