package main

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type channel struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Type     string `json:"type"`
}

type guild struct {
	Name string `json:"name"`
}

type author struct {
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
}

type sticker struct {
	SourceUrl string `json:"sourceUrl"`
	Name      string `json:"name"`
}

type attachment struct {
	Url           string `json:"url"`
	FileName      string `json:"fileName"`
	FileSizeBytes int    `json:"fileSizeBytes"`
}

type message struct {
	Timestamp       string       `json:"timestamp"`
	TimestampEdited string       `json:"timestampEdited"`
	Author          author       `json:"author"`
	Content         string       `json:"content"`
	Stickers        []sticker    `json:"stickers"`
	Attachments     []attachment `json:"attachments"`
}

type channelContent struct {
	Guild    guild     `json:"guild"`
	Channel  channel   `json:"channel"`
	Messages []message `json:"messages"`
}

type roomStructure struct {
	Name      string
	AliasName string
	RoomId    string
}

type roomRelations struct {
	Guild roomStructure
	Space roomStructure
	Room  roomStructure
}

type apiClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
}

type identifier struct {
	Type string `json:"type"`
	User string `json:"user"`
}

type loginPayload struct {
	Identifier identifier `json:"identifier"`
	Password   string     `json:"password"`
	Type       string     `json:"type"`
}

type createRoomResponse struct {
}

type creationContent struct {
	MFederate string `json:"m.federate"`
	Type      string `json:"type"`
}

type createRoomPayload struct {
	CreationContent creationContent `json:"creation_content"`
	Name            string          `json:"name"`
	Preset          string          `json:"preset"`
	RoomAliasName   string          `json:"room_alias_name"`
}

const (
	DISCORD_BACKUP_DIR    = "DISCORD_BACKUP_DIR"
	MATRIX_SERVER_ADDRESS = "MATRIX_SERVER_ADDRESS"
	MATRIX_USERNAME       = "MATRIX_USERNAME"
	MATRIX_PASSWORD       = "MATRIX_PASSWORD"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
var createdRooms []string
var client = getApiClient()

func main() {
	err := checkEnv()
	if err != nil {
		logger.Error(err.Error())
	}

	loginResponse, err := client.getMatrixAuthToken()

	if err != nil {
		logger.Error(err.Error())
	}

	client.Token = loginResponse.AccessToken

	logger.Info(client.Token)

	files := getFilesFromBackupDir()

	for _, file := range files {
		fileHandler(file)
	}
}

func checkEnv() error {
	var envs = [4]string{DISCORD_BACKUP_DIR, MATRIX_SERVER_ADDRESS, MATRIX_USERNAME, MATRIX_PASSWORD}
	for _, env := range envs {

		if os.Getenv(env) == "" {
			return errors.New(fmt.Sprintf("Missing env %s. Your docker-compose.yaml file must contains the following environment variable %s", env, envs))
		}
	}

	return nil
}

func getFilesFromBackupDir() (files []string) {

	var backupFilePath strings.Builder
	backupFilePath.WriteString(os.Getenv(DISCORD_BACKUP_DIR))
	backupFilePath.WriteString("/*.json")

	files, err := filepath.Glob(backupFilePath.String())
	if err != nil {
		panic(err)
	}

	return
}

func getDecodedFileContent(filePath string) (decodedFileContent channelContent) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	if err := json.Unmarshal(fileContent, &decodedFileContent); err != nil {
		panic(err)
	}

	return
}

func fileHandler(file string) {
	logger.Info(fmt.Sprintf("Working on file %s", file))
	fileContent := getDecodedFileContent(file)

	if fileContent.Channel.Type != "GuildTextChat" {
		return
	}

	structure := roomRelations{
		Guild: roomStructure{
			Name:      fileContent.Guild.Name,
			AliasName: getNormalizedRoomName(fileContent.Guild.Name),
		},
		Space: roomStructure{
			Name:      fileContent.Channel.Category,
			AliasName: getNormalizedRoomName(fileContent.Channel.Category),
		},
		Room: roomStructure{
			Name:      fileContent.Channel.Name,
			AliasName: getNormalizedRoomName(fileContent.Channel.Name),
		},
	}

	prepareStructureOnServer(structure)

	// postMessages(roomName, fileContent.Messages)

}

func getNormalizedRoomName(category string) (normalized string) {
	normalized = strings.Replace(category, "/[^a-zA-Z0-9_ -]/s", " ", -1)
	normalized = strings.TrimSpace(normalized)
	normalized = strings.Replace(normalized, " ", "-", -1)

	return strings.ToLower(normalized)
}

func prepareStructureOnServer(structure roomRelations) {
	structure.Guild.RoomId = client.createRoom(structure.Guild.Name, structure.Guild.AliasName)

	structure.Space.RoomId = client.createRoom(structure.Space.Name, structure.Space.AliasName)
	client.addChildSpaceToParent(structure.Guild, structure.Space)

	structure.Room.RoomId = client.createRoom(structure.Room.Name, structure.Room.AliasName)
	client.addChildSpaceToParent(structure.Space, structure.Room)
}

func (c *apiClient) createRoom(name string, aliasName string) string {
	logger.Info(fmt.Sprintf("Creating room %s", name))

	createdRooms = append(createdRooms, aliasName)

	return "roomId"
}

func (c *apiClient) addChildSpaceToParent(parentRoom roomStructure, childRoom roomStructure) {
	logger.Info(fmt.Sprintf("Linking parent room %s to %s", parentRoom.Name, childRoom.Name))

}

func (c *apiClient) postMessages(roomName string, messages []message) {
}

func getApiClient() *apiClient {
	return &apiClient{
		BaseURL:    os.Getenv(MATRIX_SERVER_ADDRESS),
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *apiClient) getMatrixAuthToken() (*loginResponse, error) {
	payload := loginPayload{
		Identifier: identifier{
			Type: "m.id.user",
			User: os.Getenv(MATRIX_USERNAME),
		},
		Password: os.Getenv(MATRIX_PASSWORD),
		Type:     "m.login.password",
	}

	jsonData, err := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", c.BaseURL, "/_matrix/client/v3/login"), bytes.NewBuffer(jsonData))

	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	defer res.Body.Close()

	loginResponse := loginResponse{}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	if err = json.Unmarshal(body, &loginResponse); err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	return &loginResponse, nil
}

func postTextMessage() {

}

func postStickerMessage() {
	getFileInfos()
	uploadFile()
	postFileMessage()
}

func postAttachmentMessage() {
	getFileInfos()
	uploadFile()
	postFileMessage()
}

func getFileInfos() {

}

func uploadFile() {

}

func postFileMessage() {

}
