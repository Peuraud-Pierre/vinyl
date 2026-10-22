// Package lyrics trouve les paroles du morceau en cours : synchronisées (LRC) par l'API publique LRCLIB
// (sans clé), sinon en texte simple.
package lyrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Line est une ligne de paroles synchronisées : son début (ms depuis le début du morceau) et son texte.
type Line struct {
	TimeMs int64  `json:"timeMs"`
	Text   string `json:"text"`
}

// Lyrics : lignes synchronisées si connues, sinon texte simple ; tout vide si rien n'a été trouvé.
type Lyrics struct {
	Synced       []Line `json:"synced"`
	Plain        string `json:"plain"`
	Instrumental bool   `json:"instrumental"`
}

func (l Lyrics) Empty() bool {
	return len(l.Synced) == 0 && strings.TrimSpace(l.Plain) == "" && !l.Instrumental
}

var stamp = regexp.MustCompile(`\[(\d{1,3}):(\d{1,2})(?:[.:](\d{1,3}))?\]`)

// ParseLRC lit des paroles au format LRC (« [01:02.50] texte », plusieurs horodatages possibles par
// ligne) ; les balises d'en-tête ([ar:…], [ti:…]) sont ignorées. Lignes triées par début.
func ParseLRC(s string) []Line {
	lines := []Line{}
	for _, raw := range strings.Split(strings.ReplaceAll(s, "\r", ""), "\n") {
		locs := stamp.FindAllStringSubmatchIndex(raw, -1)
		if len(locs) == 0 || locs[0][0] != 0 {
			continue
		}
		end := 0
		var times []int64
		for _, m := range locs {
			if m[0] != end { // horodatages en tête de ligne seulement
				break
			}
			end = m[1]
			mins, _ := strconv.ParseInt(raw[m[2]:m[3]], 10, 64)
			sec, _ := strconv.ParseInt(raw[m[4]:m[5]], 10, 64)
			var frac int64
			if m[6] >= 0 {
				f := raw[m[6]:m[7]]
				frac, _ = strconv.ParseInt(f, 10, 64)
				for i := len(f); i < 3; i++ { // « 5 » = 500 ms, « 50 » = 500 ms
					frac *= 10
				}
			}
			times = append(times, (mins*60+sec)*1000+frac)
		}
		text := strings.TrimSpace(raw[end:])
		for _, t := range times {
			lines = append(lines, Line{TimeMs: t, Text: text})
		}
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].TimeMs < lines[j].TimeMs })
	return lines
}

// LRCLib interroge https://lrclib.net.
type LRCLib struct {
	BaseURL   string
	HTTP      *http.Client
	UserAgent string
}

func NewLRCLib(version string) *LRCLib {
	return &LRCLib{BaseURL: "https://lrclib.net", HTTP: &http.Client{Timeout: 8 * time.Second},
		UserAgent: "Vinyl/" + version + " (https://github.com/Peuraud-Pierre/vinyl)"}
}

type lrcRecord struct {
	Duration     float64 `json:"duration"`
	Instrumental bool    `json:"instrumental"`
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
}

func (r lrcRecord) lyrics() Lyrics {
	return Lyrics{Synced: ParseLRC(r.SyncedLyrics), Plain: r.PlainLyrics, Instrumental: r.Instrumental}
}

func (c *LRCLib) get(ctx context.Context, path string, q url.Values, out any) (found bool, err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+path+"?"+q.Encode(), nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("lrclib : HTTP %d", resp.StatusCode)
	}
	return true, json.NewDecoder(resp.Body).Decode(out)
}

// Find cherche les paroles d'un morceau : d'abord la fiche exacte (titre, artiste, album, durée), puis une
// recherche titre + artiste dont on garde le résultat de durée la plus proche, synchronisé de préférence.
func (c *LRCLib) Find(ctx context.Context, title, artist, album string, durationMs int64) (Lyrics, error) {
	if strings.TrimSpace(title) == "" {
		return Lyrics{}, nil
	}
	if durationMs > 0 && album != "" {
		var r lrcRecord
		q := url.Values{"track_name": {title}, "artist_name": {artist}, "album_name": {album}, "duration": {strconv.FormatInt((durationMs+500)/1000, 10)}}
		if ok, err := c.get(ctx, "/api/get", q, &r); err == nil && ok && !r.lyrics().Empty() {
			return r.lyrics(), nil
		}
	}
	var found []lrcRecord
	if _, err := c.get(ctx, "/api/search", url.Values{"track_name": {title}, "artist_name": {artist}}, &found); err != nil {
		return Lyrics{}, err
	}
	return pick(found, durationMs), nil
}

// pick : parmi les résultats à moins de 5 s de la durée (tous si elle est inconnue), un synchronisé de
// préférence, puis le plus proche en durée.
func pick(rs []lrcRecord, durationMs int64) Lyrics {
	best, bestScore := -1, 0.0
	for i, r := range rs {
		gap := 0.0
		if durationMs > 0 {
			gap = r.Duration - float64(durationMs)/1000
			if gap < 0 {
				gap = -gap
			}
			if gap > 5 {
				continue
			}
		}
		score := gap
		if r.SyncedLyrics == "" {
			score += 100
		}
		if r.lyrics().Empty() {
			continue
		}
		if best < 0 || score < bestScore {
			best, bestScore = i, score
		}
	}
	if best < 0 {
		return Lyrics{}
	}
	return rs[best].lyrics()
}
