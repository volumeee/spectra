package protocol

type ScreenshotParams struct {
	URL      string `json:"url"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	FullPage bool   `json:"full_page,omitempty"`
	Format   string `json:"format,omitempty"`
	Quality  int    `json:"quality,omitempty"`
	Headless *bool  `json:"headless,omitempty"`
}

type ScreenshotResult struct {
	Data      string `json:"data"`
	Format    string `json:"format"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	SizeBytes int    `json:"size_bytes"`
}

type PDFParams struct {
	URL             string  `json:"url"`
	Format          string  `json:"format,omitempty"`
	Landscape       bool    `json:"landscape,omitempty"`
	PrintBackground bool    `json:"print_background,omitempty"`
	MarginTop       float64 `json:"margin_top,omitempty"`
	MarginBottom    float64 `json:"margin_bottom,omitempty"`
	MarginLeft      float64 `json:"margin_left,omitempty"`
	MarginRight     float64 `json:"margin_right,omitempty"`
	Headless        *bool   `json:"headless,omitempty"`
}

type PDFResult struct {
	Data      string `json:"data"`
	SizeBytes int    `json:"size_bytes"`
	Pages     int    `json:"pages"`
}

type ScrapeParams struct {
	URL       string            `json:"url"`
	Selectors map[string]string `json:"selectors,omitempty"`
	WaitFor   string            `json:"wait_for,omitempty"`
	ExecuteJS string            `json:"execute_js,omitempty"`
	Headless  *bool             `json:"headless,omitempty"`
}

type ScrapeResult struct {
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Text        string            `json:"text"`
	Links       []string          `json:"links,omitempty"`
	Images      []string          `json:"images,omitempty"`
	Meta        map[string]string `json:"meta,omitempty"`
	Custom      map[string]string `json:"custom,omitempty"`
}
