package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
)

var (
	trackSize     int
	contentLength int
	bytesLoaded   int = 0
	Duration      int = 15
)

type TrackReader struct {
	trackBuffer []byte
	i           int64 // current reading index
}

func NewTrackReader(bufferSize int) TrackReader {
	return TrackReader{
		trackBuffer: make([]byte, 0, bufferSize),
	}
}

func (r *TrackReader) Append(b []byte) {
	r.trackBuffer = append(r.trackBuffer, b...)
}

func (r *TrackReader) Read(b []byte) (n int, err error) {
	if r.i >= int64(len(r.trackBuffer)) {
		return 0, io.EOF
	}
	n = copy(b, r.trackBuffer[r.i:])
	r.i += int64(n)
	return
}

func main() {
	fetchTrackInfo()

	trackReader := NewTrackReader(trackSize)

	nextTrackPortion(&trackReader)

	decodedTrack, err := mp3.NewDecoder(&trackReader)
	if err != nil {
		panic("mp3.NewDecoder failed: " + err.Error())
	}

	log.Println("Initiating speaker with 44100 sample rate")
	otoCtx, readyChan, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   44100,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
	})
	if err != nil {
		log.Fatal(err)
	}

	<-readyChan

	player := otoCtx.NewPlayer(decodedTrack)
	player.Play()

	go func() {
		tickerChan := time.Tick(time.Second * 5)
		for {
			select {
			case <-tickerChan:
				nextTrackPortion(&trackReader)
			}
		}
	}()

	for player.IsPlaying() {
		time.Sleep(time.Millisecond)
	}

	if err := player.Close(); err != nil {
		log.Fatal("closing player", err)
	}
}

func nextTrackPortion(trackReader *TrackReader) {
	partialTrack := getTrackPartialContent()

	log.Println("append 15s to trackBuffer")
	trackReader.Append(partialTrack)
}

func getTrackPartialContent() []byte {
	log.Println("fetching song data from byte: ", bytesLoaded)

	req, err := http.NewRequest("GET", "http://localhost:8080/", nil)
	if err != nil {
		log.Fatalln("creating get request: ", err)
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", bytesLoaded))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalln("unable to get song from server: ", err)
	} else if resp.StatusCode > 400 {
		log.Fatalln("response with error: ", resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln("reading body: ", err)
	}
	defer resp.Body.Close()

	bytesLoaded += contentLength

	return body
}

func fetchTrackInfo() {
	resp, err := http.Head("http://localhost:8080/")
	if err != nil {
		log.Fatalln("unable to get song head information: ", err)
	}

	log.Println("Headers", resp.Header)
	contentLength, err = strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		log.Fatalln("getting content-length: ", err)
	}

	trackSize, err = strconv.Atoi(strings.Split(resp.Header.Get("Content-Range"), "/")[1])
	if err != nil {
		log.Fatalln("getting trackSize: ", err)
	}
}
