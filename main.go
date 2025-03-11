package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"layeh.com/gopus"
)

var (
	token     string
	guildID   string
	voiceID   string
	streamURL string
	volume    float64
	apiPort   string
	stopChan  chan bool
	frameSize int = 960
	channels  int = 2
	
	// Configuration management
	configMutex sync.RWMutex
	isStreaming bool
	currentVC   *discordgo.VoiceConnection
	botSession  *discordgo.Session
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file:", err)
	}

	token = os.Getenv("DISCORD_TOKEN")
	guildID = os.Getenv("GUILD_ID")
	voiceID = os.Getenv("VC_ID")
	streamURL = os.Getenv("STREAM_URL")
	apiPort = os.Getenv("API_PORT")
	
	volumeStr := os.Getenv("VOLUME")
	volume = 0.5
	if volumeStr != "" {
		parsedVolume, err := strconv.ParseFloat(volumeStr, 64)
		if err == nil && parsedVolume > 0 && parsedVolume <= 1.0 {
			volume = parsedVolume
		}
	}

	if streamURL == "" {
		streamURL = "https://listen.moe/stream"
	}
	
	if apiPort == "" {
		apiPort = "8080"
	}

	if token == "" {
		log.Fatal("No Discord token provided. Set DISCORD_TOKEN in .env")
	}
	if guildID == "" {
		log.Fatal("No Guild ID provided. Set GUILD_ID in .env")
	}
	if voiceID == "" {
		log.Fatal("No Voice Channel ID provided. Set VC_ID in .env")
	}

	stopChan = make(chan bool)
	isStreaming = false
}

func main() {
	var err error
	botSession, err = discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Error creating Discord session:", err)
		return
	}

	botSession.AddHandler(ready)

	err = botSession.Open()
	if err != nil {
		log.Fatal("Error opening Discord connection:", err)
		return
	}

	go startAPIServer()
	
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	close(stopChan)
	time.Sleep(1 * time.Second)
	botSession.Close()
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateGameStatus(0, "LISTEN.moe")
	fmt.Println("Bot is ready! Logged in as", event.User.Username)
	go joinAndStream(s)
}

func joinAndStream(s *discordgo.Session) {
	log.Println("Attempting to join voice channel:", voiceID)
	
	configMutex.Lock()
	vc, err := s.ChannelVoiceJoin(guildID, voiceID, false, true)
	if err != nil {
		log.Println("Error joining voice channel:", err)
		configMutex.Unlock()
		time.Sleep(10 * time.Second)
		go joinAndStream(s)
		return
	}
	currentVC = vc
	configMutex.Unlock()
	
	defer func() {
		configMutex.Lock()
		currentVC = nil
		configMutex.Unlock()
		vc.Disconnect()
	}()

	time.Sleep(2 * time.Second)

	for {
		select {
		case <-stopChan:
			log.Println("Received stop signal, disconnecting...")
			return
		default:
			configMutex.Lock()
			isStreaming = true
			currentStreamURL := streamURL
			configMutex.Unlock()
			
			log.Println("Starting to stream from:", currentStreamURL)
			err = streamAudio(vc)
			
			configMutex.Lock()
			isStreaming = false
			configMutex.Unlock()
			
			if err != nil {
				log.Println("Streaming error:", err)
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func streamAudio(vc *discordgo.VoiceConnection) error {
	client := &http.Client{}
	
	configMutex.RLock()
	currentStreamURL := streamURL
	configMutex.RUnlock()
	
	req, err := http.NewRequest("GET", currentStreamURL, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Range", "bytes=0-")
	req.Header.Set("Referer", "https://listen.moe/")
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error requesting audio stream: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned error: %d %s", resp.StatusCode, resp.Status)
	}
	
	log.Println("Connected to audio stream")
	log.Println("Content-Type:", resp.Header.Get("Content-Type"))

	vc.Speaking(true)
	defer vc.Speaking(false)

	configMutex.RLock()
	currentVolume := volume
	configMutex.RUnlock()
	
	ffmpeg := exec.Command(
		"ffmpeg",
		"-i", "pipe:0",
		"-f", "s16le",
		"-ar", "48000",
		"-ac", "2",
		"-af", fmt.Sprintf("volume=%f", currentVolume),
		"-loglevel", "warning",
		"pipe:1",
	)

	ffmpeg.Stdin = resp.Body
	ffmpegout, err := ffmpeg.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating ffmpeg stdout pipe: %v", err)
	}

	ffmpeg.Stderr = os.Stderr

	err = ffmpeg.Start()
	if err != nil {
		return fmt.Errorf("error starting ffmpeg: %v", err)
	}
	defer ffmpeg.Process.Kill()

	opusEncoder, err := gopus.NewEncoder(48000, channels, gopus.Audio)
	if err != nil {
		return fmt.Errorf("error creating opus encoder: %v", err)
	}

	audioBuf := make([]int16, frameSize*channels)
	opusBuffer := make([]byte, 1000)
	for {
		select {
		case <-stopChan:
			return nil
		default:
		}

		err = binary.Read(ffmpegout, binary.LittleEndian, &audioBuf)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return fmt.Errorf("ffmpeg output ended")
			}
			return fmt.Errorf("error reading from ffmpeg: %v", err)
		}

		opusData, err := opusEncoder.Encode(audioBuf, frameSize, len(opusBuffer))
		if err != nil {
			return fmt.Errorf("error encoding to opus: %v", err)
		}

		vc.OpusSend <- opusData
	}
}

// API Server implementation
func startAPIServer() {
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/volume", handleVolume)
	http.HandleFunc("/stream", handleStream)
	http.HandleFunc("/voice", handleVoice)
	http.HandleFunc("/restart", handleRestart)
	
	http.Handle("/", http.FileServer(http.Dir("./frontend")))

	listenAddr := fmt.Sprintf(":%s", apiPort)
	log.Printf("Starting API server on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Printf("API server error: %v", err)
	}
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	configMutex.RLock()
	defer configMutex.RUnlock()
	
	status := map[string]interface{}{
		"streaming": isStreaming,
		"stream_url": streamURL,
		"volume": volume,
		"guild_id": guildID,
		"voice_channel_id": voiceID,
		"connected": currentVC != nil,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func handleVolume(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		configMutex.RLock()
		currentVolume := volume
		configMutex.RUnlock()
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]float64{"volume": currentVolume})
		return
	}
	
	if r.Method == http.MethodPost {
		var req struct {
			Volume float64 `json:"volume"`
		}
		
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		if req.Volume < 0 || req.Volume > 1.0 {
			http.Error(w, "Volume must be between 0 and 1", http.StatusBadRequest)
			return
		}
		
		configMutex.Lock()
		volume = req.Volume
		configMutex.Unlock()
		
		// Restart stream to apply new volume
		stopChan <- true
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Volume updated"})
		return
	}
	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		configMutex.RLock()
		currentURL := streamURL
		configMutex.RUnlock()
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"stream_url": currentURL})
		return
	}
	
	if r.Method == http.MethodPost {
		var req struct {
			URL string `json:"url"`
		}
		
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		if req.URL == "" {
			http.Error(w, "URL cannot be empty", http.StatusBadRequest)
			return
		}
		
		configMutex.Lock()
		streamURL = req.URL
		configMutex.Unlock()
		
		// Restart stream to apply new URL
		stopChan <- true
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Stream URL updated"})
		return
	}
	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleVoice(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		configMutex.RLock()
		currentGuild := guildID
		currentVC := voiceID
		configMutex.RUnlock()
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"guild_id": currentGuild,
			"voice_channel_id": currentVC,
		})
		return
	}
	
	if r.Method == http.MethodPost {
		var req struct {
			GuildID string `json:"guild_id"`
			VoiceID string `json:"voice_channel_id"`
		}
		
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		if req.GuildID == "" || req.VoiceID == "" {
			http.Error(w, "Guild ID and Voice Channel ID cannot be empty", http.StatusBadRequest)
			return
		}
		
		configMutex.Lock()
		guildID = req.GuildID
		voiceID = req.VoiceID
		
		// Disconnect from current voice channel if connected
		if currentVC != nil {
			currentVC.Disconnect()
			currentVC = nil
		}
		configMutex.Unlock()
		
		// Reconnect to new voice channel
		go joinAndStream(botSession)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Voice channel updated"})
		return
	}
	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	configMutex.Lock()
	if currentVC != nil {
		currentVC.Disconnect()
		currentVC = nil
	}
	configMutex.Unlock()
	
	go joinAndStream(botSession)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Streaming restarted"})
}