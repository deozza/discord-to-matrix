package main

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type channel struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Topic    string `json:"topic"`
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
	Topic     string
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
	RoomId    string `json:"room_id"`
	Error     string `json:"error"`
	ErrorCode string `json:"errcode"`
}

type creationContent struct {
	MFederate bool   `json:"m.federate"`
	Type      string `json:"type"`
}

type createRoomPayload struct {
	CreationContent creationContent `json:"creation_content"`
	Name            string          `json:"name"`
	Preset          string          `json:"preset"`
	RoomAliasName   string          `json:"room_alias_name"`
	Topic           string          `json:"topic"`
}

type getRoomIdResponse struct {
	RoomId    string `json:"room_id"`
	Error     string `json:"error"`
	ErrorCode string `json:"errcode"`
}

type addChildSpaceToParentPayload struct {
	Via       []string `json:"via"`
	Suggested bool     `json:"sugggested"`
}

type addChildSpaceToParentResponse struct {
	Error     string `json:"error"`
	ErrorCode string `json:"errcode"`
}

const (
	DISCORD_BACKUP_DIR    = "DISCORD_BACKUP_DIR"
	MATRIX_SERVER_ADDRESS = "MATRIX_SERVER_ADDRESS"
	MATRIX_TOKEN          = "MATRIX_TOKEN"
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

	client.Token = os.Getenv(MATRIX_TOKEN)

	if client.Token == "" {
		loginResponse, err := client.getMatrixAuthToken()

		if err != nil {
			logger.Error(err.Error())
		}

		client.Token = loginResponse.AccessToken
		logger.Info("Authenticated with provided username and password")
	} else {
		logger.Info("Authenticated with provided token")
	}

	files := getFilesFromBackupDir()

	for _, file := range files {
		_, structure := fileHandler(file)
		prepareStructureOnServer(structure)
	}
}

func checkEnv() error {
	var envs = [2]string{DISCORD_BACKUP_DIR, MATRIX_SERVER_ADDRESS}
	for _, env := range envs {

		if os.Getenv(env) == "" {
			return errors.New(fmt.Sprintf("Missing env %s. Your docker-compose.yaml file must contains the following environment variable %s", env, envs))
		}
	}

	if os.Getenv(MATRIX_TOKEN) == "" {
		if os.Getenv(MATRIX_USERNAME) == "" && os.Getenv(MATRIX_PASSWORD) == "" {
			return errors.New("You must provide either a valid matrix token or your credentials to login to your server.")
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

func fileHandler(file string) (fileContent channelContent, structure roomRelations) {
	logger.Info(fmt.Sprintf("Working on file %s", file))
	fileContent = getDecodedFileContent(file)

	if fileContent.Channel.Type != "GuildTextChat" {
		return
	}

	structure = roomRelations{
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

	return
}

func getNormalizedRoomName(category string) (normalized string) {
	normalized = strings.Replace(category, "/[^a-zA-Z0-9_ -]/s", " ", -1)
	normalized = strings.TrimSpace(normalized)
	normalized = strings.Replace(normalized, " ", "-", -1)

	return strings.ToLower(normalized)
}

func prepareStructureOnServer(structure roomRelations) {
	structure.Guild = client.prepareRoomOnServer(structure.Guild, true)

	structure.Space = client.prepareRoomOnServer(structure.Space, true)

	err := client.addChildSpaceToParent(structure.Guild, structure.Space)
	if err != nil {
		panic(err)
	}

	structure.Room = client.prepareRoomOnServer(structure.Room, false)
	err = client.addChildSpaceToParent(structure.Space, structure.Room)
	if err != nil {
		panic(err)
	}
}

func (c *apiClient) prepareRoomOnServer(room roomStructure, isASpace bool) roomStructure {
	getRoomIdResponse, err := client.getRoomId(room.AliasName)

	if getRoomIdResponse != nil {
		room.RoomId = getRoomIdResponse.RoomId

		return room
	}

	createRoomResponse, err := client.createRoom(room.Name, room.AliasName, isASpace)
	if err != nil {
		panic(err)
	}

	room.RoomId = createRoomResponse.RoomId

	return room
}

func (c *apiClient) getRoomId(aliasName string) (*getRoomIdResponse, error) {
	logger.Info(fmt.Sprintf("Checking if room %s already exists", aliasName))

	endpoint := "/_matrix/client/v3/directory/room/"
	queryParameter := url.QueryEscape(fmt.Sprintf("#%s:%s", aliasName, os.Getenv(MATRIX_SERVER_ADDRESS)))
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%s%s", c.BaseURL, endpoint, queryParameter), nil)

	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	res, err := c.HTTPClient.Do(req)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	getRoomIdResponse := getRoomIdResponse{}

	if err = json.Unmarshal(body, &getRoomIdResponse); err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	if res.StatusCode != 200 {
		logger.Info(fmt.Sprintf("Room %s does not exist", aliasName))
		return nil, errors.New(getRoomIdResponse.Error)
	}

	logger.Info(fmt.Sprintf("Room %s already exist with id %s", aliasName, getRoomIdResponse.RoomId))

	createdRooms = append(createdRooms, aliasName)

	return &getRoomIdResponse, nil
}

func (c *apiClient) createRoom(name string, aliasName string, isASpace bool) (*createRoomResponse, error) {
	logger.Info(fmt.Sprintf("Creating room %s", aliasName))

	roomType := ""
	if isASpace == true {
		roomType = "m.space"
	}

	payload := createRoomPayload{
		CreationContent: creationContent{
			MFederate: false,
			Type:      roomType,
		},
		Name:          name,
		Preset:        "private_chat",
		RoomAliasName: aliasName,
	}

	jsonData, err := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", c.BaseURL, "/_matrix/client/v3/createRoom"), bytes.NewBuffer(jsonData))

	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+c.Token)

	res, err := c.HTTPClient.Do(req)
	defer res.Body.Close()
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	createRoomResponse := createRoomResponse{}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	if err = json.Unmarshal(body, &createRoomResponse); err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, errors.New(createRoomResponse.Error)
	}

	logger.Info(fmt.Sprintf("Room %s created with id %s", aliasName, createRoomResponse.RoomId))
	createdRooms = append(createdRooms, aliasName)

	return &createRoomResponse, nil
}

func (c *apiClient) addChildSpaceToParent(parentRoom roomStructure, childRoom roomStructure) error {
	logger.Info(fmt.Sprintf("Linking parent room %s to %s", parentRoom.Name, childRoom.Name))

	payload := addChildSpaceToParentPayload{
		Via:       []string{os.Getenv(MATRIX_SERVER_ADDRESS)},
		Suggested: true,
	}

	jsonData, err := json.Marshal(payload)

	url := fmt.Sprintf("%s%s%s%s%s", c.BaseURL, "/_matrix/client/v3/rooms/", url.QueryEscape(parentRoom.RoomId), "/state/m.space.child/", url.QueryEscape(childRoom.RoomId))

	fmt.Println(url)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonData))

	if err != nil {
		logger.Error(err.Error())
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+c.Token)

	res, err := c.HTTPClient.Do(req)
	defer res.Body.Close()
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	addaddChildSpaceToParentResponse := addChildSpaceToParentResponse{}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	if err = json.Unmarshal(body, &addaddChildSpaceToParentResponse); err != nil {
		logger.Error(err.Error())
		return err
	}
	if res.StatusCode != 200 {
		return errors.New(addaddChildSpaceToParentResponse.Error)
	}

	logger.Info("Rooms are linked")

	return nil
}

func (c *apiClient) postMessages(roomName string, messages []message) {
}

func getApiClient() *apiClient {
	return &apiClient{
		BaseURL:    "https://" + os.Getenv(MATRIX_SERVER_ADDRESS),
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
