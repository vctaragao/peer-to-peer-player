package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var (
	SongName       = "Daydream - Soobin Hoang SonThaoboy (Hiderway Remix)"
	Duration int64 = 15
)

type (
	Tracks map[string]TrackInfo

	TrackInfo struct {
		os.FileInfo
		Format string `json:"format"`
		Length int64  `json:"length"`
	}
)

func (t *TrackInfo) StrSize() string {
	return strconv.Itoa(int(t.FileInfo.Size()))
}

func (t *TrackInfo) BytesPerSecond() int64 {
	return t.FileInfo.Size() / t.Length
}

func main() {
	port := flag.String("port", "8080", "port to run the server on")

	http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open("tracks/" + SongName + ".mp3")
		if err != nil {
			log.Println("opening file: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer f.Close()

		trackInfo, err := getTrackInfo(SongName)
		if err != nil {
			log.Println("getting track info: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		trackInfo.FileInfo, err = os.Stat("tracks/" + SongName + ".mp3")
		if err != nil {
			log.Println("getting track file info: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var songRead int64

		bytesRange := r.Header.Get("Range")
		log.Println("Range", bytesRange)
		if bytesRange != "" {
			bytesRange = strings.Replace(strings.Replace(bytesRange, "bytes=", "", 1), "-", "", 1)
			bytesRead, err := strconv.Atoi(bytesRange)
			if err != nil {
				log.Println("reading range requested: ", err)
				w.WriteHeader(http.StatusInternalServerError)
			}

			songRead = int64(bytesRead)
		}

		buffer := make([]byte, Duration*trackInfo.BytesPerSecond())

		n, err := f.ReadAt(buffer, songRead)
		if err != nil {
			log.Println("reading 15s of song: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", n))
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", songRead, songRead+int64(n), trackInfo.Size()))

		w.WriteHeader(http.StatusPartialContent)
		if _, err := w.Write(buffer[:n]); err != nil {
			log.Println("respoding 15s of song: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	})

	log.Println("Starting server at port " + *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

func getTrackInfo(songName string) (TrackInfo, error) {
	conf, err := os.ReadFile("./tracks/tracks.json")
	if err != nil {
		log.Println("reading tracks infos: ", err)
		return TrackInfo{}, err
	}

	var tracks Tracks
	if err := json.Unmarshal(conf, &tracks); err != nil {
		log.Println("unmarshaling tracks infos: ", err)
		return TrackInfo{}, err
	}

	track, exists := tracks[songName]
	if !exists {
		log.Println("getting mapped track info: ", errors.New("track not found"))
		return TrackInfo{}, errors.New("track not found")
	}

	return track, nil
}
