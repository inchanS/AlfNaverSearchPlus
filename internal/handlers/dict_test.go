package handlers

import (
	"encoding/json"
	"testing"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
)

func TestDictionaryDefinitionsAndCopy(t *testing.T) {
	tests := []struct {
		name, prefix, word, reading, meaning, title string
		run                                         func(*alfred.Feedback, string) error
	}{
		{
			"English definition", "en1", "love", "", "an intense feeling of deep affection",
			"love     an intense feeling of deep affection", enendic,
		},
		{
			"French punctuation", "frko", "오늘", "", `aujourd'hui & "demain" <test>`,
			`오늘     aujourd'hui & "demain" <test>`,
			func(fb *alfred.Feedback, word string) error { return dictMulti(fb, "frko", word) },
		},
		{
			"Hanja punctuation", "hanja", "愛", `"애"`, `사랑 & '애정'`,
			`愛["애"] 사랑 & '애정'`, hanja,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("alfred_workflow_cache", t.TempDir())
			// The live API stores definitions in the fourth field, with empty
			// pronunciation/metadata fields preceding them in the English feed.
			body, err := json.Marshal(map[string]any{"items": []any{[]any{[]any{
				[]string{tt.word}, []string{tt.reading}, []string{""}, []string{tt.meaning},
			}}}})
			if err != nil {
				t.Fatal(err)
			}
			if err := cache.Write(cache.Key(tt.prefix, tt.word), body); err != nil {
				t.Fatal(err)
			}
			fb := alfred.New()
			if err := tt.run(fb, tt.word); err != nil {
				t.Fatal(err)
			}
			// Check what Alfred receives after JSON decoding, including text.copy.
			encoded, err := json.Marshal(fb)
			if err != nil {
				t.Fatal(err)
			}
			var got alfred.Feedback
			if err := json.Unmarshal(encoded, &got); err != nil {
				t.Fatal(err)
			}
			if len(got.Items) != 2 {
				t.Fatalf("got %d items, want fallback and one result", len(got.Items))
			}
			item := got.Items[1]
			if item.Title != tt.title || item.Text == nil || item.Text.Copy != tt.meaning {
				t.Errorf("title/copy lost definition or punctuation: %+v / %+v", item, item.Text)
			}
			if item.Arg != tt.word || !item.Valid || item.QuicklookURL == "" {
				t.Errorf("search action changed: %+v", item)
			}
		})
	}
}
