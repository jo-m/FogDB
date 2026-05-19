package api

import (
	"testing"
	"time"
)

func TestFindLatestAssetsPicksHighestRun(t *testing.T) {
	mk := func(href string) stacAsset {
		return stacAsset{Type: "text/csv", Href: href}
	}
	feats := []stacFeature{
		{
			ID: "20260519-ch",
			Assets: map[string]stacAsset{
				"vnut12.lssw.202605190200.tre200h0.csv": mk("https://h/a"),
				"vnut12.lssw.202605190400.tre200h0.csv": mk("https://h/b"),
				"vnut12.lssw.202605190300.tre200h0.csv": mk("https://h/c"),
				"vnut12.lssw.202605190100.rre150h0.csv": mk("https://h/d"),
			},
		},
		{
			ID: "20260520-ch",
			Assets: map[string]stacAsset{
				"vnut12.lssw.202605190500.rre150h0.csv": mk("https://h/e"),
				"vnut12.lssw.202605190200.jww003i0.csv": mk("https://h/f"),
			},
		},
	}
	got, err := findLatestAssets(feats, []string{"tre200h0", "rre150h0", "jww003i0"})
	if err != nil {
		t.Fatalf("findLatestAssets: %v", err)
	}
	want := map[string]struct {
		run  string
		href string
	}{
		"tre200h0": {"202605190400", "https://h/b"},
		"rre150h0": {"202605190500", "https://h/e"},
		"jww003i0": {"202605190200", "https://h/f"},
	}
	for p, w := range want {
		la, ok := got[p]
		if !ok {
			t.Errorf("missing parameter %q", p)
			continue
		}
		wantTime, _ := time.ParseInLocation("200601021504", w.run, time.UTC)
		if !la.RunTime.Equal(wantTime) {
			t.Errorf("%s: RunTime = %s, want %s", p, la.RunTime, wantTime)
		}
		if la.Href != w.href {
			t.Errorf("%s: Href = %s, want %s", p, la.Href, w.href)
		}
	}
}

func TestFindLatestAssetsMissingParam(t *testing.T) {
	feats := []stacFeature{
		{
			ID: "x",
			Assets: map[string]stacAsset{
				"vnut12.lssw.202605190200.tre200h0.csv": {Href: "h"},
			},
		},
	}
	_, err := findLatestAssets(feats, []string{"tre200h0", "rre150h0"})
	if err == nil {
		t.Fatalf("expected error for missing parameter")
	}
}
