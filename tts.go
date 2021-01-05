package main

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"math/rand"
	"regexp"
	"strings"

	tts "cloud.google.com/go/texttospeech/apiv1"
	"github.com/jonas747/ogg"
	tts_pb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
	"mvdan.cc/xurls/v2"
)

var (
	ttsClient *tts.Client
	urlReg    = xurls.Relaxed()
	ignoreReg = regexp.MustCompile("^[(（)].*[）)]$")
	kusaReg   = regexp.MustCompile("[wWｗＷ]+$")
)

func hash(s string) int64 {
	h := fnv.New64()
	h.Write([]byte(s))
	x := h.Sum64() / 2
	if x < 0 {
		x = -x
	}
	return int64(x)
}

func ttsReq(text, lang, voiceToken string) *tts_pb.SynthesizeSpeechRequest {
	req := &tts_pb.SynthesizeSpeechRequest{
		Input: &tts_pb.SynthesisInput{
			InputSource: &tts_pb.SynthesisInput_Text{Text: text},
		},
		Voice: &tts_pb.VoiceSelectionParams{
			LanguageCode: lang,
		},
		AudioConfig: &tts_pb.AudioConfig{
			AudioEncoding: tts_pb.AudioEncoding_OGG_OPUS,
		},
	}

	gs := []tts_pb.SsmlVoiceGender{tts_pb.SsmlVoiceGender_NEUTRAL, tts_pb.SsmlVoiceGender_MALE, tts_pb.SsmlVoiceGender_FEMALE}
	rs := []float64{0.75, 1.0, 1.3, 1.7}
	ps := []float64{-15, -8, 0, 8, 15}
	r := rand.New(rand.NewSource(hash(voiceToken)))
	req.Voice.SsmlGender = gs[r.Intn(len(gs))]
	req.AudioConfig.SpeakingRate = rs[r.Intn(len(rs))]
	req.AudioConfig.Pitch = ps[r.Intn(len(ps))]

	return req
}

// lang: https://cloud.google.com/text-to-speech/docs/voices
func ttsOGGGoogle(text, lang, voiceToken string) ([][]byte, error) {
	text = Sanitize(text, lang)
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
			if err != io.EOF {
				return nil, fmt.Errorf("error decode on PacketDecoder: %w", err)
			}
			return output, nil
		}
		output = append(output, packet)
	}
}

// Sanitize modifies s easier to read in following steps:
// 1. trim spaces
// 2. replace continuous 'w's to kusa
// 3. replace URL to "URL"
// 4. replace continuous whitespaces to single one
func Sanitize(s, lang string) string {
	s = strings.TrimSpace(s)
	b := sanitizeBytes([]byte(s), lang)
	s = string(b)
	return strings.Join(strings.Fields(s), " ")
}

func sanitizeBytes(s []byte, lang string) []byte {
	b := []byte(s)
	if ignoreReg.Match(b) {
		return nil
	}
	if (strings.HasPrefix(lang, "ja-") || lang == "ja") && kusaReg.Match(b) {
		b = kusaReg.ReplaceAll(b, []byte(" くさ"))
	}
	return urlReg.ReplaceAll(b, []byte(" URL "))
}
