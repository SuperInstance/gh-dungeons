package game

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// PLATO room structure - mirrors what the room server returns
type PLATOTile struct {
	ID       string `json:"id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type PLATORoom struct {
	Name      string      `json:"name"`
	Tiles     []PLATOTile `json:"tiles"`
	CreatedAt string      `json:"created_at"`
}

// PLATOSource replaces CodeFile scanner with PLATO room data
type PLATOSource struct {
	Rooms   []PLATORoom
	URL     string
}

// RoomListResponse is the JSON returned by /rooms
type RoomListResponse struct {
	Rooms []string `json:"rooms"`
}

// FetchPLATOSource connects to a PLATO server and fetches room data
func FetchPLATOSource(baseURL string, roomName string) (*PLATOSource, error) {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	source := &PLATOSource{URL: baseURL}

	client := &http.Client{Timeout: 10 * time.Second}

	// If specific room requested, only fetch that one
	if roomName != "" {
		room, err := fetchRoom(client, baseURL, roomName)
		if err != nil {
			return nil, fmt.Errorf("fetching room %s: %w", roomName, err)
		}
		source.Rooms = append(source.Rooms, *room)
		return source, nil
	}

	// Otherwise fetch all rooms
	rooms, err := fetchRoomList(client, baseURL)
	if err != nil {
		return nil, fmt.Errorf("fetching room list: %w", err)
	}
	source.Rooms = rooms
	return source, nil
}

func fetchRoomList(client *http.Client, baseURL string) ([]PLATORoom, error) {
	resp, err := client.Get(baseURL + "rooms")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("room list returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var list RoomListResponse
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("parsing room list: %w", err)
	}

	var rooms []PLATORoom
	for _, name := range list.Rooms {
		room, err := fetchRoom(client, baseURL, name)
		if err != nil {
			continue // Skip rooms that fail to load
		}
		rooms = append(rooms, *room)
	}

	return rooms, nil
}

func fetchRoom(client *http.Client, baseURL, name string) (*PLATORoom, error) {
	resp, err := client.Get(baseURL + "room/" + name)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("room %s returned status %d", name, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var room PLATORoom
	if err := json.Unmarshal(body, &room); err != nil {
		return nil, fmt.Errorf("parsing room %s: %w", name, err)
	}

	room.Name = name
	return &room, nil
}

// PLATOToCodeFiles converts PLATO rooms to the CodeFile format used by dungeon generation.
// Each room becomes a CodeFile with its name as the path and tile Q&A concatenated as "lines".
func (ps *PLATOSource) PLATOToCodeFiles() []CodeFile {
	var files []CodeFile
	for _, room := range ps.Rooms {
		lines := make([]string, 0, len(room.Tiles)*2)
		for _, tile := range room.Tiles {
			// Use Q&A as content lines
			if tile.Question != "" {
				lines = append(lines, tile.Question)
			}
			if tile.Answer != "" {
				lines = append(lines, tile.Answer)
			}
		}

		// If no Q&A content, use room name repeated as filler
		if len(lines) == 0 {
			lines = append(lines, room.Name)
		}

		content := strings.Join(lines, "\n")
		hash := sha256.Sum256([]byte(content))

		files = append(files, CodeFile{
			Path:  room.Name,
			Lines: lines,
			SHA:   string(hash[:]),
		})
	}
	return files
}

// ComputeSeed generates a deterministic seed from PLATO room data.
// Uses room name + tile count + first tile Q&A for variety.
func (ps *PLATOSource) ComputeSeed() int64 {
	h := sha256.New()

	for _, room := range ps.Rooms {
		h.Write([]byte(room.Name))
		h.Write([]byte(fmt.Sprintf("%d", len(room.Tiles))))
		if len(room.Tiles) > 0 {
			if room.Tiles[0].Question != "" {
				h.Write([]byte(room.Tiles[0].Question))
			}
		}
	}

	sum := h.Sum(nil)
	return int64(binary.BigEndian.Uint64(sum[:8]))
}

// GetRoomNames returns all room names in the source.
func (ps *PLATOSource) GetRoomNames() []string {
	names := make([]string, len(ps.Rooms))
	for i, room := range ps.Rooms {
		names[i] = room.Name
	}
	return names
}