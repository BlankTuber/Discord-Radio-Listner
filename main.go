package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
)

func main() {
	// URL of the audio stream
	streamURL := "https://listen.moe/stream"
	
	fmt.Println("Connecting to stream:", streamURL)
	
	// Create an HTTP client
	client := &http.Client{}
	
	// Create a request with necessary headers
	req, err := http.NewRequest("GET", streamURL, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	
	// Set headers similar to what a browser would send
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Range", "bytes=0-")
	req.Header.Set("Referer", "https://listen.moe/")
	
	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error requesting audio stream:", err)
		return
	}
	defer resp.Body.Close()
	
	// Check if we got a successful response
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Server returned error: %d %s\n", resp.StatusCode, resp.Status)
		return
	}
	
	// Print stream information
	fmt.Println("Connected to audio stream")
	fmt.Println("Content-Type:", resp.Header.Get("Content-Type"))
	
	// Look for ICY metadata headers
	for name, values := range resp.Header {
		if strings.HasPrefix(strings.ToLower(name), "icy-") {
			fmt.Printf("%s: %s\n", name, values[0])
		}
	}

	// Setup signal catching for clean exit
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	
	// Start the appropriate audio player based on platform
	var cmd *exec.Cmd
	var stdin io.WriteCloser
	
	switch runtime.GOOS {
	case "linux":
		// Try to use ffplay on Linux
		cmd = exec.Command("ffplay", "-nodisp", "-autoexit", "-loglevel", "quiet", "-i", "pipe:0")
	case "darwin":
		// Try to use ffplay on macOS
		cmd = exec.Command("ffplay", "-nodisp", "-autoexit", "-loglevel", "quiet", "-i", "pipe:0")
	case "windows":
		// Try to use ffplay on Windows
		cmd = exec.Command("ffplay", "-nodisp", "-autoexit", "-loglevel", "quiet", "-i", "pipe:0")
	default:
		fmt.Println("Unsupported platform")
		return
	}
	
	// Get stdin pipe to write audio data
	stdin, err = cmd.StdinPipe()
	if err != nil {
		fmt.Println("Error getting stdin pipe:", err)
		return
	}
	
	// Redirect stdout/stderr to null to avoid ffplay output
	cmd.Stdout = nil
	cmd.Stderr = nil
	
	// Start the command
	err = cmd.Start()
	if err != nil {
		fmt.Println("Error starting player:", err)
		fmt.Println("Make sure ffplay (part of ffmpeg) is installed on your system")
		return
	}
	
	fmt.Println("\nPlaying stream. Press Ctrl+C to stop")
	
	// Start a goroutine to copy audio data from the response to ffplay
	go func() {
		defer stdin.Close()
		
		buffer := make([]byte, 8192)
		for {
			n, err := resp.Body.Read(buffer)
			if err != nil && err != io.EOF {
				fmt.Println("Error reading from stream:", err)
				break
			}
			
			if n > 0 {
				_, err := stdin.Write(buffer[:n])
				if err != nil {
					fmt.Println("Error writing to player:", err)
					break
				}
			}
			
			if err == io.EOF {
				break
			}
		}
		
		done <- true
	}()
	
	// Wait for either Ctrl+C or streaming error
	select {
	case <-sigs:
		fmt.Println("\nReceived interrupt signal")
	case <-done:
		fmt.Println("Stream ended")
	}
	
	// Kill the player process if it's still running
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
	
	fmt.Println("Playback stopped")
}