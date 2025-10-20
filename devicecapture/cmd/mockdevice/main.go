// mockdevice Starts an http server that listens on 8080
// routes:
// /ping - Returns 200 Response
// /stream - MJPEG streaming response
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func getTestImage() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 250, 250))
	timestring := time.Now().Format("15:04:05")
	addLabel(img, 10, 10, timestring)
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, nil)
	if err != nil {
		return nil
	}
	return buf.Bytes()
}

func addLabel(img *image.RGBA, x, y int, label string) {
	col := color.RGBA{200, 100, 0, 255}
	point := fixed.Point26_6{fixed.I(x), fixed.I(y)}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
}

func mjpegHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")

	boundary := "\r\n--frame\r\nContent-Type: image/jpeg\r\n\r\n"
	//img := getImageFrame()
	for {

		n, err := io.WriteString(w, boundary)
		if err != nil || n != len(boundary) {
			log.Printf("TestServer -> Error writing boundary: %v", err)
			return
		}
		img := getTestImage()
		_, err = w.Write(img)
		if err != nil {
			log.Printf("TestServer -> Error writing image: %v", err)
			return
		}

		n, err = io.WriteString(w, "\r\n")
		if err != nil || n != 2 {
			log.Printf("TestServer -> Error writing boundary: %v", err)
			return
		}
		// Optional: control frame rate
		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	http.HandleFunc("/stream", mjpegHandler)
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	log.Print("Server started on port 8080. Press Ctrl+C to stop the server.")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
