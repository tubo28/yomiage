package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"

	tts "cloud.google.com/go/texttospeech/apiv1"
	"github.com/jonas747/ogg"
	tts_pb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
)

var (
	client *tts.Client
)

func init() {
	var err error
	client, err = tts.NewClient(context.TODO())
	if err != nil {
		log.Fatal("failed to create tts client: ", err.Error())
	}
}

// lang: https://cloud.google.com/text-to-speech/docs/voices
// todo: randomize voice
func ttsOGGGoogle(s, lang string) ([][]byte, error) {
	req := tts_pb.SynthesizeSpeechRequest{
		Input: &tts_pb.SynthesisInput{
			InputSource: &tts_pb.SynthesisInput_Text{Text: s},
		},
		Voice: &tts_pb.VoiceSelectionParams{
			LanguageCode: lang,
			SsmlGender:   tts_pb.SsmlVoiceGender_NEUTRAL,
		},
		AudioConfig: &tts_pb.AudioConfig{
			AudioEncoding: tts_pb.AudioEncoding_OGG_OPUS,
		},
	}

	resp, err := client.SynthesizeSpeech(context.TODO(), &req)
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
			if err != io.EOF {
				return nil, fmt.Errorf("error decode on PacketDecoder: %w", err)
			}
			return output, nil
		}
		output = append(output, packet)
	}
}
