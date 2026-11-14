package handlers

import (
	"encoding/json"
	"testing"

	"github.com/inchanS/AlfNaverSearchPlus/internal/alfred"
	"github.com/inchanS/AlfNaverSearchPlus/internal/cache"
)

func TestFinanceResultURLs(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())
	t.Setenv("cache_age", "3600")

	tests := []struct {
		name  string
		code  string
		route any
		want  string
	}{
		{"에스앤에스텍", "101490", "/domestic/stock/101490/total", "https://stock.naver.com/domestic/stock/101490/price"},
		{"엔비디아", "NVDA", "/worldstock/stock/NVDA.O/total", "https://stock.naver.com/worldstock/stock/NVDA.O/price"},
		{"애플랙", "AFL", "/worldstock/stock/AFL/total", "https://stock.naver.com/worldstock/stock/AFL/price"},
		{"토요타자동차", "7203", "/worldstock/stock/7203.T/total", "https://stock.naver.com/worldstock/stock/7203.T/price"},
		{"텐센트", "00700", "/worldstock/stock/0700.HK/total", "https://stock.naver.com/worldstock/stock/0700.HK/price"},
		{"코스피", "KOSPI", "/domestic/index/KOSPI", "https://stock.naver.com/domestic/index/KOSPI"},
		{"나스닥 종합", "IXIC", "/worldstock/index/.IXIC", "https://stock.naver.com/worldstock/index/.IXIC"},
		{"나스닥 100 선물", "NQcv1", "/worldstock/futures/NQcv1", "https://stock.naver.com/worldstock/futures/NQcv1"},
		{"existing price tab", "NVDA", "/worldstock/stock/NVDA.O/price", "https://stock.naver.com/worldstock/stock/NVDA.O/price"},
		{"query and fragment", "NVDA", "/worldstock/stock/NVDA.O/total?foo=total#total", "https://stock.naver.com/worldstock/stock/NVDA.O/price?foo=total#total"},
		{"non-stock total suffix", "example", "/marketindex/example/total", "https://stock.naver.com/marketindex/example/total"},
		{"missing URL", "NVDA", nil, ""},
		{"empty URL", "NVDA", "", ""},
		{"non-string URL", "NVDA", 123, ""},
		{"malformed escape", "NVDA", "/worldstock/stock/%zz/total", ""},
		{"relative URL", "NVDA", "worldstock/stock/NVDA.O/total", ""},
		{"absolute URL", "NVDA", "https://example.com/worldstock/stock/NVDA.O/total", ""},
		{"protocol-relative URL", "NVDA", "//example.com/worldstock/stock/NVDA.O/total", ""},
		{"root URL", "NVDA", "/", ""},
		{"missing stock identifier", "NVDA", "/worldstock/stock//total", ""},
		{"incomplete stock route", "NVDA", "/worldstock/stock/NVDA.O", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Cache the API response so this exercises parsing and Alfred feedback
			// without network requests or changes to the user's Alfred cache.
			item := map[string]any{"name": tt.name, "code": tt.code}
			if tt.route != nil {
				item["url"] = tt.route
			}
			body, err := json.Marshal(map[string]any{"items": []any{item}})
			if err != nil {
				t.Fatal(err)
			}
			if err := cache.Write(cache.Key("navfinance", tt.name), body); err != nil {
				t.Fatal(err)
			}

			fb := alfred.New()
			if err := testRequest(t, nil).finance(fb, tt.name); err != nil {
				t.Fatal(err)
			}
			if len(fb.Items) != 1 {
				t.Fatalf("got %d items, want 1", len(fb.Items))
			}
			got := fb.Items[0]
			if got.Arg != tt.want || got.QuicklookURL != tt.want {
				t.Errorf("arg = %q, quicklook = %q; want both %q", got.Arg, got.QuicklookURL, tt.want)
			}
			if got.Valid != (tt.want != "") {
				t.Errorf("valid = %v, want %v", got.Valid, tt.want != "")
			}
			if got.Autocomplete != tt.name || got.Title != "Search Naver Finance for '"+tt.name+"'" {
				t.Errorf("result label or autocomplete changed: %+v", got)
			}
		})
	}
}
