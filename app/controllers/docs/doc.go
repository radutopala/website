package docs

import (
	"fmt"
	"html/template"
	"path"
	"path/filepath"

	"aahframework.org/aah.v0"
	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"

	"github.com/go-aah/website/app/controllers"
	"github.com/go-aah/website/app/markdown"
	"github.com/go-aah/website/app/util"
)

var (
	releases    []string
	docBasePath string
)

// Doc struct documentation application controller
type Doc struct {
	controllers.App
}

// Before method doc brfore interceptor
func (d *Doc) Before() {
	d.App.Before()

	d.AddViewArg("ShowVersionDocs", true).
		AddViewArg("ShowInsightSideNav", true).
		AddViewArg("CodeBlock", true)
}

// Index method is documentation application home page
func (d *Doc) Index() {
	d.Reply().Redirect(d.ReverseURL("docs.version_home", releases[0]))
}

// VersionHome method Displays the documentation in selected language. Default
// is English.
func (d *Doc) VersionHome() {
	version := d.Req.Params.PathValue("version")
	if !ess.IsSliceContainsString(releases, version) {
		switch ess.StripExt(version) {
		case "godoc":
			d.GoDoc()
		case "tutorials":
			d.Tutorials()
		default:
			queryStr := d.Req.Params.Query.Encode()
			targetURL := d.ReverseURL("docs.show_doc", releases[0], version)
			if !ess.IsStrEmpty(queryStr) {
				targetURL = targetURL + "?" + queryStr
			}
			d.Reply().Redirect(targetURL)
		}
		return
	}

	data := aah.Data{
		"IsVersionHome":      true,
		"ShowVersionDocs":    false,
		"ShowInsightSideNav": false,
		"CurrentVersion":     version,
	}
	d.Reply().HTMLl("docs.html", data)
}

// ShowDoc method displays requested documentation page based language and version.
func (d *Doc) ShowDoc() {
	version := d.Req.Params.PathValue("version")
	d.AddViewArg("CurrentVersion", version)

	branchName := util.GetBranchName(version)
	if branchName == "master" {
		d.AddViewArg("LatestRelease", true)
	}

	content := d.Req.Params.PathValue("content")
	switch ess.StripExt(util.TrimPrefixSlash(content)) {
	case "release-notes":
		d.ReleaseNotes()
		return
	}

	docPath := path.Clean(path.Join(version, content))
	mdPath := util.FilePath(docPath, docBasePath)
	article, found := markdown.Get(mdPath)
	if !found {
		d.NotFound(false)
		return
	}

	data := aah.Data{"Article": article}
	d.Reply().HTMLl("docs.html", data)
}

// GoDoc method display aah framework godoc links
func (d *Doc) GoDoc() {
	data := aah.Data{
		"IsGoDoc": true,
	}
	d.Reply().HTMLlf("docs.html", "godoc.html", data)
}

// Tutorials method display aah framework tutorials github links or guide.
func (d *Doc) Tutorials() {
	data := aah.Data{
		"IsTutorials": true,
	}
	d.Reply().HTMLlf("docs.html", "tutorials.html", data)
}

// ReleaseNotes method display aah framework release notes, changelogs, migration notes.
func (d *Doc) ReleaseNotes() {
	version := d.Req.Params.PathValue("version")
	changelogMdPath := util.FilePath(path.Join(version, "changelog.md"), docBasePath)
	whatsNewMdPath := util.FilePath(path.Join(version, "whats-new.md"), docBasePath)
	migrationGuideMdPath := util.FilePath(path.Join(version, "migration-guide.md"), docBasePath)

	changelog, _ := markdown.Get(changelogMdPath)
	whatsNew, _ := markdown.Get(whatsNewMdPath)
	migrationGuide, _ := markdown.Get(migrationGuideMdPath)

	data := aah.Data{
		"IsReleaseNotes": true,
		"Changelog":      changelog,
		"WhatsNew":       whatsNew,
		"MigrationGuide": migrationGuide,
	}
	d.Reply().HTMLlf("docs.html", "release-notes.html", data)
}

// RefreshDoc method to refresh documentation from github
func (d *Doc) RefreshDoc() {
	go util.RefreshDocContent()
	log.Info("Documentation is refresh from GitHub and Cache cleared.")
	d.Reply().Text("Docs are being refreshed")
}

// NotFound method handles not found URLs.
func (d *Doc) NotFound(isStatic bool) {
	log.Warnf("Page not found: %s", d.Req.Path)
	data := aah.Data{
		"IsNotFound": true,
	}

	d.Reply().HTMLlf("docs.html", "notfound.html", data)
}

func init() {
	aah.AddTemplateFunc(template.FuncMap{
		"docurlc": func(viewArgs map[string]interface{}, key string) template.HTML {
			params := viewArgs["RequestParams"].(*ahttp.Params)
			version := params.PathValue("version")
			if !ess.IsSliceContainsString(releases, version) {
				version = releases[0]
			}

			return template.HTML(fmt.Sprintf("/%s/%s",
				version,
				aah.AppConfig().StringDefault(key, "")))
		},
		"absrequrl": func(viewArgs map[string]interface{}) template.URL {
			return template.URL(fmt.Sprintf("%s://%s%s", viewArgs["Scheme"], viewArgs["Host"], viewArgs["RequestPath"]))
		},
	})

	aah.OnStart(func(e *aah.Event) {
		releases, _ = aah.AppConfig().StringList("docs.releases")
		docBasePath = filepath.Join(aah.AppConfig().StringDefault("docs.dir", ""), "aah-documentation")
		_ = ess.MkDirAll(docBasePath, 0755)
		util.RefreshDocContent()
	})

	aah.OnShutdown(func(e *aah.Event) {
		markdown.ClearCache()
	})
}