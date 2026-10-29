<?php

declare(strict_types=1);
require_once __DIR__ . '/vendor/autoload.php';

use Symfony\Contracts\HttpClient\HttpClientInterface;
use Symfony\Component\HttpClient\HttpClient;

function getClient(): HttpClientInterface {
    $client = HttpClient::create(defaultOptions: [
        'base_uri' => 'https://matrix.example.org',
        'headers' => [
            'Content-Type' => 'application/json'
        ]
    ]);
    
    return $client;
}

function main(string $path) {
    $client = getClient();
    $token = authToMatrixServer($client);
    $spacesCreated = [];
    
    foreach(glob($path. '/*.json') as $filename) {

        logMessage(importance: 'INFO', context: 'READING FILES', content: 'Working on file ' . basename($filename));
        $content = getFileContent($filename);

        if($content['channel']['type'] !== 'GuildTextChat') {
            unset($content);
            continue;
        }

        $roomAliasName = getNormalizedRoomName($content['channel']['category']);

        if(shouldCreateSpace($roomAliasName, $spacesCreated) === true) {
            $spacesCreated[$roomAliasName] = createSpace($content['channel']['category'], $roomAliasName, $client, $token);
            addChildSpaceToMain('', $spacesCreated[$roomAliasName], $client, $token);
        }

        foreach($content['messages'] as $message) {
            sendTextMessageToRoom('', $message, $client, $token);
            
            if(empty($message['stickers']) === false) {
                foreach($message['stickers'] as $sticker) {
                    sendStickerMessageToRoom('', $message, $sticker, $path, $client, $token);
                }
                
                continue;
            }
            
            if(empty($message['attachments']) === false) {
                foreach($message['attachments'] as $attachment) {
                    sendAttachmentMessageToRoom('', $message, $attachment, $path, $client, $token);
                }
                
                continue;
            }
        }
        
        unset($content);

    }
}

function logMessage(string $context, string $content, string $importance = 'INFO'): void {
    $messageTemplate = "[%s][%s] - [%s] : %s";
    $date = new \DateTime();
    echo sprintf($messageTemplate, $importance, $date->format('Y-m-d H:i:s'), $context, $content . "\n");
}

function getFileContent(string $path): array {
    return json_decode(file_get_contents($path), true);
}

function authToMatrixServer(HttpClientInterface $client): string {
    $response = $client->request('POST', '/_matrix/client/v3/login', [
        'json' => [
          "identifier" => [
            "type" => "m.id.user",
            "user" => ""
          ],
          "password" => "",
          "type" => "m.login.password"
        ]
    ]);

    $decodedPayload = $response->toArray();
    
    logMessage(importance: 'INFO', context: 'AUTHENTICATION', content: "Successfully logged in");

    return $decodedPayload['access_token'];
}

function shouldCreateSpace(string $roomAliasName, array $existingSpaces): bool {
    return in_array($roomAliasName, $existingSpaces) ===  false;
}

function getNormalizedRoomName(string $roomName): string {
    
    return preg_replace('/[^a-zA-Z0-9_ -]/s',' ', $roomName)
        |> trim(...)
        |> (fn($x) => str_replace(' ', '-', $x))
        |> strtolower(...);
}

function createSpace(string $roomName, string $roomAliasName, HttpClientInterface $client, string $token): string {

    logMessage(importance: 'INFO', context: 'CREATING SPACE', content: $roomName);
    
    $response = $client->request('PUT', '/_matrix/client/v3/createRoom', [
        'json' => [
          "creation_content" => [
            "m.federate" => false,
            "type" => "m.space"
          ],
          "name" =>  $roomName,
          "preset" =>  "private_chat",
          "room_alias_name" =>  $roomAliasName
        ],
        'auth_bearer' => $token
    ]);

    $decodedPayload = $response->toArray();
    
    logMessage(importance: 'INFO', context: 'CREATING SPACE', content: 'Done : ' . $decodedPayload['room_id']);

    return $decodedPayload['room_id'];
}

function addChildSpaceToMain(string $mainSpaceId, string $childSpaceId, HttpClientInterface $client, string $token): void {
    logMessage(importance: 'INFO', context: 'CHILD SPACE', content: '...');

    $response = $client->request('PUT', '/_matrix/client/v3/rooms/' . $mainSpaceId . '/state/m.space.child/' . $childSpaceId, [
        'json' => [
          "via" => [
            "matrix.example.org"
          ],
          "suggested" => true
        ],
        'auth_bearer' => $token
    ]);

    $decodedPayload = $response->toArray();
    
    logMessage(importance: 'INFO', context: 'CHILD SPACE', content: 'Done');
}

function sendTextMessageToRoom(string $roomId, array $message, HttpClientInterface $client, string $token): void {
    logMessage(importance: 'INFO', context: 'POSTING TEXT MESSAGE', content: '...');

    $contentTemplate = "[%s][%s][%s] : %s";
    $content = sprintf($contentTemplate, $message['timestamp'], $message['author']['name'], $message['author']['nickname'], $message['content']);

    $response = $client->request('PUT', '/_matrix/client/v3/rooms/' . $roomId. '/state/m.room.message', [
        'json' => [
            'body' => $content,
            'msgtype' => 'm.text'
        ],
        'auth_bearer' => $token
    ]);

    $decodedPayload = $response->toArray();
    
    logMessage(importance: 'INFO', context: 'POSTING MESSAGE', content: 'Done');
}

function sendStickerMessageToRoom(string $roomId, array $message, array $sticker, string $path, HttpClientInterface $client, string $token): void {
    logMessage(importance: 'INFO', context: 'POSTING STICKER MESSAGE', content: 'posting message');

    $fileInfos = getFileInfos($path . '/' . $sticker['sourceUrl'], filename: $sticker['name']);

    $contentUri = uploadFile($fileInfos, $client, $token);
    
    sendFileMessageToRoom($roomId, $fileInfos, $contentUri, $client, $token);
    
}

function sendAttachmentMessageToRoom(string $roomId, array $message, array $attachment, string $path, HttpClientInterface $client, string $token): void {
    logMessage(importance: 'INFO', context: 'POSTING ATTACHMENT MESSAGE', content: 'posting message');

    $fileInfos = getFileInfos($path . '/' . $attachment['url'], $attachment);

    $contentUri = uploadFile($fileInfos, $client, $token);

    sendFileMessageToRoom($roomId, $fileInfos, $contentUri, $client, $token);
}

function sendFileMessageToRoom(string $roomId, array $fileInfos, string $contentUri, HttpClientInterface $client, string $token): void {
    $msgtype = 'm.file';

    if(str_contains($fileInfos['mimetype'], 'image') === true) {
        $msgtype = 'm.image';
    }

    if(str_contains($fileInfos['mimetype'], 'audio') === true) {
        $msgtype = 'm.audio';
    }
    
    if(str_contains($fileInfos['mimetype'], 'video') === true) {
        $msgtype = 'm.video';
    }

    $body = [
            'body' => $fileInfos['filename'],
            'url' => $contentUri,
            'msgtype' => $msgtype,
            'info' => [
                'mimetype' => $fileInfos['mimetype'],
                'size' => $fileInfos['size']
            ]
        ];

        var_dump($body);
    
    $response = $client->request('PUT', '/_matrix/client/v3/rooms/' . $roomId. '/state/m.room.message', [
        'json' => $body,
        'auth_bearer' => $token
    ]);
    
}

function getFileInfos(string $filepath, array | null $attachment = null, string | null $filename = null): array {

    $fileInfos = [
        'filepath' => $filepath,
        'filename' => '',
        'mimetype' => '',
        'size' => 0
    ];

    $fileInfos['mimetype'] = mime_content_type($filepath);

    if(empty($filename) === false) {
        $fileInfos['filename'] = $filename;
    }

    if(empty($attachment) === false) {
        $fileInfos['filename'] = $attachment['fileName'];
        $fileInfos['size'] = $attachment['fileSizeBytes'];

        return $fileInfos;
    }

    $fileInfos['filename'] = basename(filepath);
    $fileInfos['size'] = filesize($filepath);
    
    
    return $fileInfos;
}

function uploadFile(array $fileInfos, HttpClientInterface $client, string $token): string {
    logMessage(importance: 'INFO', context: 'POSTING FILE MESSAGE', content: 'uploading file ' . $fileInfos['filepath']);
    
    $response = $client->request('POST', '/_matrix/media/v3/upload?filename='.basename($fileInfos['filename']), [
        'body' => fopen($fileInfos['filepath'], 'r'),
        'auth_bearer' => $token,
        'headers' => [
            'Content-Type' => $fileInfos['mimetype'],
        ]
    ]);

    $decodedPayload = $response->toArray();
    
    logMessage(importance: 'INFO', context: 'POSTING FILE MESSAGE', content: 'file uploaded at ' . $decodedPayload['content_uri']);

    return $decodedPayload['content_uri'];
}

$dir = __DIR__ . '/discord';
main($dir);

?>
