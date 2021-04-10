package tts

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"math/rand"

	gtts "cloud.google.com/go/texttospeech/apiv1"
	"github.com/jonas747/ogg"
	gtts_pb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
)

var (
	ttsClient *gtts.Client
)

// Init initializes Google TTS client
func Init() {
	var err error
	ttsClient, err = gtts.NewClient(context.TODO())
	if err != nil {
		log.Fatal("failed to create tts client: ", err.Error())
	}
}

// Close closes client
func Close() {
	if err := ttsClient.Close(); err != nil {
		log.Print("error closing tts client: ", err.Error())
	}
}

func hash(s string) int64 {
	h := fnv.New64()
	h.Write([]byte(s))
	return int64(h.Sum64() / 2)
}

func ttsReq(text, lang, voiceToken string) *gtts_pb.SynthesizeSpeechRequest {
	req := &gtts_pb.SynthesizeSpeechRequest{
		Input: &gtts_pb.SynthesisInput{
			InputSource: &gtts_pb.SynthesisInput_Text{Text: text},
		},
		Voice: &gtts_pb.VoiceSelectionParams{
			LanguageCode: lang,
		},
		AudioConfig: &gtts_pb.AudioConfig{
			AudioEncoding: gtts_pb.AudioEncoding_OGG_OPUS,
		},
	}

	gs := []gtts_pb.SsmlVoiceGender{gtts_pb.SsmlVoiceGender_NEUTRAL, gtts_pb.SsmlVoiceGender_MALE, gtts_pb.SsmlVoiceGender_FEMALE}
	rs := []float64{0.75, 1.0, 1.2, 1.4}
	ps := []float64{-5, 0, 5, 8}
	r := rand.New(rand.NewSource(hash(voiceToken)))
	req.Voice.SsmlGender = gs[r.Intn(len(gs))]
	req.AudioConfig.SpeakingRate = rs[r.Intn(len(rs))]
	req.AudioConfig.Pitch = ps[r.Intn(len(ps))]

	return req
}

// OGGGoogle call Google Cloud TTS API
func OGGGoogle(text, lang, voiceToken string) ([][]byte, error) {
	if len(text) == 0 {
		return nil, fmt.Errorf("empty text")
	}
	req := ttsReq(text, lang, voiceToken)

	resp, err := ttsClient.SynthesizeSpeech(context.TODO(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to create ogg, received error from google api: %w", err)
	}

	log.Printf("ogg data created %d bytes", len(resp.AudioContent))
	return makeOGGBuffer(resp.AudioContent)
}

func makeOGGBuffer(in []byte) (output [][]byte, err error) {
	od := ogg.NewDecoder(bytes.NewReader(in))
	pd := ogg.NewPacketDecoder(od)

	// Run through the packet decoder appending the bytes to our output [][]byte
	for {
		packet, _, err := pd.Decode()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("error decode on PacketDecoder: %w", err)
			}
			return output, nil
		}
		output = append(output, packet)
	}
}
