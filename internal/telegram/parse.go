package telegram

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// DefaultFileName is the file Telegram Desktop writes into an export folder.
const DefaultFileName = "result.json"

// ErrNoChats is returned when the input is valid JSON but contains no chats.
var ErrNoChats = errors.New("no chats found: is this a Telegram Desktop JSON export?")

// document covers both export flavours: a single chat ("Export chat history")
// and a full account export ("Export Telegram data") with chats.list.
type document struct {
	Chat
	Messages *[]Message `json:"messages"`
	Chats    *struct {
		List []Chat `json:"list"`
	} `json:"chats"`
	LeftChats *struct {
		List []Chat `json:"list"`
	} `json:"left_chats"`
}

// Parse decodes an export from r and returns every chat it contains.
func Parse(r io.Reader) ([]Chat, error) {
	var doc document
	if err := json.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode export: %w", err)
	}

	var chats []Chat
	if doc.Messages != nil {
		c := doc.Chat
		c.Messages = *doc.Messages
		chats = append(chats, c)
	}
	if doc.Chats != nil {
		chats = append(chats, doc.Chats.List...)
	}
	if doc.LeftChats != nil {
		chats = append(chats, doc.LeftChats.List...)
	}
	if len(chats) == 0 {
		return nil, ErrNoChats
	}
	return chats, nil
}

// ResolvePath accepts either a JSON file or an export directory and returns
// the path of the JSON file to read.
func ResolvePath(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return path, nil
	}
	p := filepath.Join(path, DefaultFileName)
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("%s is a directory without %s", path, DefaultFileName)
	}
	return p, nil
}

// Load resolves path (file or export folder) and parses it.
func Load(path string) ([]Chat, error) {
	p, err := ResolvePath(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	chats, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	return chats, nil
}
