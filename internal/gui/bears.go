package gui

import (
	"math/rand"

	"github.com/mikeflynn/hardhat-honeybear/internal/gui/assets"
)

// Other Bear Ideas: Winking, More Looking, Concerned Looking, Blushing?, ...
var bears = Bears{
	{Name: "Sleeping", File: "bear_sleeping.jpg", Category: "boot"},
	{Name: "Angry", File: "bear_angry.jpg", Category: "emote", SubCategory: "angry"},
	{Name: "Cool", File: "bear_cool.jpg", Category: "emote", SubCategory: "happy"},
	{Name: "Happy", File: "bear_happy.jpg", Category: "standard"},
	{Name: "Laughing", File: "bear_laughing.jpg", Category: "standard"},
	{Name: "Look Left", File: "bear_look_left.jpg", Category: "standard"},
	{Name: "Look Right", File: "bear_look_right.jpg", Category: "standard"},
	{Name: "Sad", File: "bear_sad.jpg", Category: "emote", SubCategory: "sad"},
	{Name: "Surprised", File: "bear_surprised.jpg", Category: "standard"},
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

func (b Bears) GetBearByCategory(category string, subCategory *string) *Bear {
	var bears []Bear
	if category == "" {
		bears = b
	} else {
		for _, bear := range b {
			if bear.Category == category {
				if subCategory != nil && bear.SubCategory == *subCategory {
					bears = append(bears, bear)
				} else if subCategory == nil {
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
