package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gorilla/websocket"
)

// We'll need to define an Upgrader
// this will require a Read and Write buffer size
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	// We'll need to check the origin of our connection
	// this will allow us to make requests from our React
	// development server to here.
	// For now, we'll do no checking and just allow any connection
	CheckOrigin: func(r *http.Request) bool { return true },
}

// define a reader which will listen for
// new messages being sent to our WebSocket
// endpoint
type Msg struct {
	Type    string
	Message string
}

func reader(conn *websocket.Conn) {
	for {
		// read in a message
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		// print out that message for clarity
		fmt.Println(string(p))
		var msg Msg
		json.Unmarshal([]byte(p), &msg)
		// fmt.Printf("Type: %s, Message: %s", msg.Type, msg.Message)

		if msg.Message == "frontend connected" {
			if err := conn.WriteMessage(messageType, []byte(`{"type":"success","message":"BOOOM" }`)); err != nil {
				log.Println(err)
				return
			}
		}
	}
}

// define our WebSocket endpoint
func serveWs(w http.ResponseWriter, r *http.Request) {
	// fmt.Println(r.Host)

	// upgrade this connection to a WebSocket
	// connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}
	// listen indefinitely for new messages coming
	// through on our WebSocket connection
	reader(ws)
}

func unzip() {
	dst := "dest"
	archive, err := zip.OpenReader("new_frontend.zip")
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(dst, f.Name)
		// fmt.Println("unzipping file ", filePath)

		if !strings.HasPrefix(filePath, filepath.Clean(dst)+string(os.PathSeparator)) {
			fmt.Println("invalid file path")
			return
		}
		if f.FileInfo().IsDir() {
			// fmt.Println("creating directory...")
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			panic(err)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		dstFile.Close()
		fileInArchive.Close()
	}
	os.Rename("./dest/ledfx_frontend_v2", "./frontend")

	// cleanup
	if _, err := os.Stat("dest"); err == nil {
		os.RemoveAll("dest")
	}

	defer os.RemoveAll("new_frontend.zip")

}

func serveFrontend() {
	serveFrontend := http.FileServer(http.Dir("frontend"))
	fileMatcher := regexp.MustCompile(`\.[a-zA-Z]*$`)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !fileMatcher.MatchString(r.URL.Path) {
			http.ServeFile(w, r, "frontend/index.html")
		} else {
			serveFrontend.ServeHTTP(w, r)
		}
	})
}

func getFrontend() {
	log.Println("Getting latest Frontend")
	resp, err := http.Get("https://github.com/YeonV/LedFx-Frontend-v2/releases/latest/download/ledfx_frontend_v2.zip")
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return
	}
	// Delete old files
	if _, err := os.Stat("frontend"); err == nil {
		os.RemoveAll("frontend")
	}
	defer os.RemoveAll("new_frontend.zip")

	// Create the file
	out, err := os.Create("new_frontend.zip")
	if err != nil {
		fmt.Printf("err: %s", err)
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Printf("err: %s", err)
	}
	// Extract frontend
	unzip()
	log.Println("Got latest Frontend")
}

func setupRoutes() {
	getFrontend()
	serveFrontend()
	// map our `/ws` endpoint to the `serveWs` function
	http.HandleFunc("/ws", serveWs)
}

func main() {
	fmt.Println("===========================================")
	fmt.Println("            LedFx v0.01 by Blade")
	fmt.Println("    [CTRL]+Click: http://localhost:8080")
	fmt.Println("===========================================")
	setupRoutes()
	http.ListenAndServe(":8080", nil)
}
