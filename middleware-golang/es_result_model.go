package main

import "encoding/json"

func UnmarshalESResult(data []byte) (ESResult, error) {
	var r ESResult
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ESResult) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type ESResult struct {
	Took     int64  `json:"took"`
	TimedOut bool   `json:"timed_out"`
	Shards   Shards `json:"_shards"`
	Hits     Hits   `json:"hits"`
}

type Hits struct {
	Total    Total   `json:"total"`
	MaxScore float64 `json:"max_score"`
	Hits     []Hit   `json:"hits"`
}

type Hit struct {
	Index  string  `json:"_index"`
	ID     string  `json:"_id"`
	Score  float64 `json:"_score"`
	Source Source  `json:"_source"`
}

type Source struct {
	Title       string `json:"title"`
	ShortDesc   string `json:"short_desc"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Deeplink    string `json:"deeplink"`
	Icon        string `json:"icon"`
	Identifier  string `json:"identifier"`
	Price       string `json:"price"`
	SubCategory string `json:"sub-category"`
	Type        string `json:"type"`
	URL         string `json:"url"`
}

type SourceShort struct {
	Title     string `json:"title"`
	ShortDesc string `json:"short_desc"`
}

type Total struct {
	Value    int64  `json:"value"`
	Relation string `json:"relation"`
}

type Shards struct {
	Total      int64 `json:"total"`
	Successful int64 `json:"successful"`
	Skipped    int64 `json:"skipped"`
	Failed     int64 `json:"failed"`
}
