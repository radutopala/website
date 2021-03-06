package markdown

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/russross/blackfriday"

	"aahframe.work"
	"aahframe.work/essentials"
	"aahframe.work/log"

	"aahframework.org/website/app/models"
)

var (
	articleCache = make(map[string]*models.Article)

	markdownHTMLFlags = 0 |
		blackfriday.HTML_USE_XHTML |
		blackfriday.HTML_USE_SMARTYPANTS |
		blackfriday.HTML_SMARTYPANTS_FRACTIONS |
		blackfriday.HTML_SMARTYPANTS_DASHES |
		blackfriday.HTML_SMARTYPANTS_LATEX_DASHES

	markdownExtensions = 0 |
		blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
		blackfriday.EXTENSION_TABLES |
		blackfriday.EXTENSION_FENCED_CODE |
		blackfriday.EXTENSION_AUTOLINK |
		blackfriday.EXTENSION_STRIKETHROUGH |
		blackfriday.EXTENSION_SPACE_HEADERS |
		blackfriday.EXTENSION_HEADER_IDS |
		blackfriday.EXTENSION_AUTO_HEADER_IDS |
		blackfriday.EXTENSION_BACKSLASH_LINE_BREAK |
		blackfriday.EXTENSION_DEFINITION_LISTS

	markdownOptions = blackfriday.Options{Extensions: markdownExtensions}

	isCacheEnabled bool

	mu              = &sync.Mutex{}
	contentPreparer *strings.Replacer
)

// Parse method parsed the markdown content into html using blackfriday library
// and create Article object.
func Parse(lines []string) *models.Article {
	pos := 0
	for idx, l := range lines {
		if strings.TrimSpace(l) == "---" {
			pos = idx + 1
			break
		}
	}

	article := &models.Article{}

	for _, v := range lines[:pos] {
		if v == "---" {
			break
		}
		idx := strings.IndexByte(v, ':')
		if idx == -1 {
			continue
		}
		switch v[:idx] {
		case "Title":
			article.Title = strings.TrimSpace(v[idx+1:])
		case "Desc":
			article.Desc = strings.TrimSpace(v[idx+1:])
		case "Keywords":
			article.Keywords = strings.TrimSpace(v[idx+1:])
		}
	}

	fileContent := strings.Join(lines[pos:], "\n")
	htmlRender := blackfriday.HtmlRenderer(markdownHTMLFlags, "", "")
	content := string(blackfriday.MarkdownOptions([]byte(fileContent), htmlRender, markdownOptions))
	content = contentPreparer.Replace(content)
	article.Content = content

	return article
}

// Get method returns the parsed markdown content for given URL path.
func Get(mdPath string) (*models.Article, bool) {
	if isCacheEnabled {
		if article, found := articleCache[mdPath]; found {
			return article, true
		}
	}

	article := getArticle(mdPath)
	if article == nil {
		return nil, false
	}

	if article.IsContent() && isCacheEnabled {
		mu.Lock()
		articleCache[mdPath] = article
		mu.Unlock()
	}

	return article, article.IsContent()
}

// LoadCache methods loads the markdown into cache for given base path.
func LoadCache(docBasePath string) {
	var files []string
	excludes := ess.Excludes{".git", "LICENSE", "README.md"}
	_ = ess.Walk(docBasePath, func(srcPath string, info os.FileInfo, err error) error {
		if excludes.Match(filepath.Base(srcPath)) {
			if info.IsDir() {
				// excluding directory
				return filepath.SkipDir
			}
			// excluding file
			return nil
		}

		if !info.IsDir() {
			files = append(files, srcPath)
		}
		return nil
	})

	for _, md := range files {
		RefreshCacheByFile(md)
	}
}

// RefreshCacheByFile method refereshes the Markdown cache by file.
func RefreshCacheByFile(mdPath string) {
	article := getArticle(mdPath)
	if article != nil && article.IsContent() {
		mu.Lock()
		articleCache[mdPath] = article
		mu.Unlock()
		log.Debugf("Refreshed file: %s", mdPath)
	} else {
		log.Warnf("Referesh: File not found: %s", mdPath)
	}
}

// RemoveCacheByFile method removes the Markdown cache by file.
func RemoveCacheByFile(mdPath string) {
	if _, found := articleCache[mdPath]; found {
		mu.Lock()
		delete(articleCache, mdPath)
		mu.Unlock()
		log.Infof("Removed from cache: %s", mdPath)
	} else {
		log.Warnf("Remove: File not found: %s", mdPath)
	}
}

func getArticle(mdPath string) *models.Article {
	f, err := aah.App().VFS().Open(mdPath)
	if err != nil {
		return nil
	}

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	article := Parse(lines)
	article.File = mdPath
	return article
}

// ClearCache method clears the Markdown cache.
func ClearCache(e *aah.Event) {
	if len(articleCache) > 0 {
		log.Info("Clearing cache")
	}
	mu.Lock()
	articleCache = make(map[string]*models.Article)
	mu.Unlock()
}

// FetchMarkdownConfig gets markdown related config value
func FetchMarkdownConfig(e *aah.Event) {
	cfg := aah.App().Config()
	isCacheEnabled = cfg.BoolDefault("markdown.cache", false)

	// Dynamica URL placement
	contentPreparer = strings.NewReplacer(
		"{{aah_domain_url}}", cfg.StringDefault("markdown.aah_domain_url", "https://aahframework.org"),
		"{{aah_docs_domain_url}}", cfg.StringDefault("markdown.aah_docs_domain_url", "https://docs.aahframework.org"),
		"{{aah_github_issues_url}}", cfg.StringDefault("link.aah.github_issues", "https://aahframework.org/issues"),
		"{{aah_issues_url}}", cfg.StringDefault("link.aah.github_issues", "https://aahframework.org/issues"),
		"{{aah_cdn_host}}", cfg.StringDefault("cdn.host", "//cdn.aahframework.org"),
		"{{aah_examples_url}}", cfg.StringDefault("markdown.aah_examples_url", "https://aahframework.org/examples"),
		"<table>", `<table class="table table-bordered table-hover">`,
	)
}
