package gui

import (
	"math/rand"
	"slices"
	"time"

	"github.com/charmbracelet/log"
	"github.com/mikeflynn/honeybearhoneypot/internal/gui/assets"
)

// Other Bear Ideas: Winking, More Looking, Concerned Looking, Blushing?, ...
var bears = Bears{
	{Name: "Happy", File: "bear_happy.jpg", Category: "standard", SubCategory: "idle"},
	{Name: "Look Left", File: "bear_look_left.jpg", Category: "standard", SubCategory: "idle"},
	{Name: "Look Right", File: "bear_look_right.jpg", Category: "standard", SubCategory: "idle"},
	{Name: "Surprised", File: "bear_surprised.jpg", Category: "standard", SubCategory: "react"},

	{Name: "Talking - Bored", File: "bear_talk_bored.jpg", Category: "standard", SubCategory: "talk", Wait: getDurationSeconds(4)},
	{Name: "Talking - QR", File: "bear_talk_qr.jpg", Category: "standard", SubCategory: "talk", Wait: getDurationSeconds(6)},
	{Name: "Talking - Rickroll", File: "bear_talk_rickroll.jpg", Category: "standard", SubCategory: "talk", Wait: getDurationSeconds(6)},
	{Name: "Talking - Hydrox", File: "bear_talk_hydrox.jpg", Category: "standard", SubCategory: "talk", Wait: getDurationSeconds(6)},
	{Name: "Talking - SSH", File: "bear_talk_ssh.jpg", Category: "standard", SubCategory: "talk", Wait: getDurationSeconds(6)},
	{Name: "Talking - Pot", File: "bear_talk_honeypot.jpg", Category: "standard", SubCategory: "talk", Wait: getDurationSeconds(4)},
	{Name: "Talking - Voices", File: "bear_talk_voices.jpg", Category: "standard", SubCategory: "talk", Wait: getDurationSeconds(4)},
	{Name: "Talking - Game", File: "bear_talk_game.jpg", Category: "standard", SubCategory: "talk", Wait: getDurationSeconds(4)},
	{Name: "Talking - Hack", File: "bear_talk_hack.jpg", Category: "standard", SubCategory: "talk", Wait: getDurationSeconds(6)},

	{Name: "Angry", File: "bear_angry.jpg", Category: "emote", SubCategory: "angry"},
	{Name: "Cool", File: "bear_cool.jpg", Category: "emote", SubCategory: "happy"},
	{Name: "Laughing", File: "bear_laughing.jpg", Category: "emote", SubCategory: "happy"},
	{Name: "Sad", File: "bear_sad.jpg", Category: "emote", SubCategory: "sad"},

	{Name: "Sleeping", File: "bear_sleeping.jpg", Category: "boot"},
	{Name: "Terminator", File: "bear_terminator.jpg", Category: "special"},

	// Glitch Bears
	{Name: "001", File: "bear_glitch_001.jpg", Category: "glitch"},
	{Name: "002", File: "bear_glitch_002.jpg", Category: "glitch"},
	{Name: "003", File: "bear_glitch_003.jpg", Category: "glitch"},
	{Name: "004", File: "bear_glitch_004.jpg", Category: "glitch"},
}

type Bear struct {
	Name        string
	File        string
	Category    string
	SubCategory string
	Wait        *time.Duration
}

func (b Bear) FileData() ([]byte, error) {
	return assets.Images.ReadFile(b.File)
}

type Bears []Bear

func (b Bears) GetBear(name string) *Bear {
	for _, bear := range b {
		if bear.Name == name {
			return &bear
		}
	}
	return nil
}

func (b Bears) GetBearByCategory(category string, subCategory ...string) *Bear {
	log.Debug("GetBearByCategory", "category", category, "subCategory", subCategory)

	var bears []Bear
	if category == "" {
		bears = b
	} else {
		for _, bear := range b {
			if bear.Category == category {
				if len(subCategory) > 0 && slices.Contains(subCategory, bear.SubCategory) {
					bears = append(bears, bear)
				} else if len(subCategory) == 0 || subCategory[0] == "" {
					bears = append(bears, bear)
				}
			}
		}
	}

	// Return a single random bear
	if len(bears) > 0 {
		randomIndex := rand.Intn(len(bears))
		return &bears[randomIndex]
	}

	return nil
}

func getDurationSeconds(sec int) *time.Duration {
	if sec <= 0 {
		return nil
	}
	d := time.Duration(sec) * time.Second
	return &d
}
