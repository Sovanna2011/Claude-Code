package api

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/kss/sugarplan/internal/domain"
	"github.com/kss/sugarplan/internal/service"
	"github.com/kss/sugarplan/internal/store"
	"github.com/kss/sugarplan/internal/tabular"
)

// The HTTP surface of the controlled import: the mapping templates, the upload
// that stages a file, the preview, the row-level error download and the commit.

// MaxUploadBytes bounds a single upload. A season of daily rows is a few tens of
// kilobytes; anything at this size is a mistake worth refusing before it is read
// into memory.
const MaxUploadBytes = 8 << 20 // 8 MiB

func (s *Server) handleListImportMappings(w http.ResponseWriter, r *http.Request) {
	mappings, err := s.imports.ListMappings(r.Context(), r.URL.Query().Get("kind"))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.ImportMapping]{Value: mappings, Count: len(mappings)})
}

func (s *Server) handleSaveImportMapping(w http.ResponseWriter, r *http.Request) {
	var m domain.ImportMapping
	if err := decodeJSON(w, r, &m); err != nil {
		writeProblem(w, r, err)
		return
	}
	saved, err := s.imports.SaveMapping(r.Context(), m)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	w.Header().Set("ETag", etagFor(saved.RowVersion))
	writeJSON(w, saved)
}

// handleImportFields serves the catalogue the mapping screen builds itself from,
// so a field added to an import needs no change in the browser.
func (s *Server) handleImportFields(w http.ResponseWriter, r *http.Request) {
	fields, err := s.imports.Fields(r.Context())
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	kinds := make([]string, 0, len(fields))
	for k := range fields {
		kinds = append(kinds, string(k))
	}
	sort.Strings(kinds)

	out := make([]map[string]any, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, map[string]any{
			"kind":   k,
			"fields": fields[domain.ImportKind(k)],
			"keys":   domain.KeyFields(domain.ImportKind(k)),
		})
	}
	writeJSON(w, map[string]any{"value": out, "count": len(out)})
}

// handleStageImport takes the upload.
//
// The body is the file itself rather than a multipart form: the mapping, the
// version and the series are small enough to be query parameters, and a raw body
// is what a script with curl and a nightly export can post without building a
// form. A browser sends the same thing.
func (s *Server) handleStageImport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	req := service.StageRequest{
		MappingCode: q.Get("mapping"),
		MappingID:   q.Get("mappingId"),
		VersionID:   q.Get("versionId"),
		FileName:    q.Get("fileName"),
		Series:      domain.Series(strings.ToUpper(q.Get("series"))),
	}
	if req.FileName == "" {
		req.FileName = fileNameFromDisposition(r.Header.Get("Content-Disposition"))
	}

	content, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxUploadBytes))
	if err != nil {
		writeProblem(w, r, wrapValidation(fmt.Sprintf(
			"the upload could not be read; the limit is %d MiB", MaxUploadBytes>>20)))
		return
	}
	req.Content = content

	// An upload is a posting like any other: a client that retried because the
	// connection dropped must not stage the same file twice.
	s.postOnce(w, r, func() (any, error) {
		return s.imports.Stage(r.Context(), req)
	})
}

// fileNameFromDisposition reads the name a browser attaches to an upload, so the
// job records what the file was called even when the caller did not say.
func fileNameFromDisposition(header string) string {
	for _, part := range strings.Split(header, ";") {
		part = strings.TrimSpace(part)
		if rest, ok := strings.CutPrefix(part, "filename="); ok {
			return strings.Trim(rest, `"`)
		}
	}
	return ""
}

func (s *Server) handleListImports(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := store.ImportFilter{
		Kind:      strings.ToUpper(q.Get("kind")),
		Status:    strings.ToUpper(q.Get("status")),
		VersionID: q.Get("versionId"),
		Skip:      atoiOr(q.Get("$skip"), 0),
		Top:       atoiOr(q.Get("$top"), 0),
	}
	page, err := s.imports.ListJobs(r.Context(), f)
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, pageResponse[domain.ImportJob]{
		Value: page.Items, Count: page.Count, Skip: f.Skip, Top: f.Top,
	})
}

func (s *Server) handleGetImport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	preview, err := s.imports.GetJob(r.Context(), r.PathValue("id"),
		q.Get("errorsOnly") == "true", atoiOr(q.Get("$skip"), 0), atoiOr(q.Get("$top"), 0))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, preview)
}

// handleImportErrors downloads the rows that failed, as a CSV somebody opens
// next to the file they uploaded.
//
// The row number is the first column for exactly that reason: it is the line in
// their own spreadsheet, and fixing an import means going to those lines.
func (s *Server) handleImportErrors(w http.ResponseWriter, r *http.Request) {
	preview, err := s.imports.GetJob(r.Context(), r.PathValue("id"), true, 0, tabular.MaxRows)
	if err != nil {
		writeProblem(w, r, err)
		return
	}

	var body strings.Builder
	cw := csv.NewWriter(&body)
	_ = cw.Write([]string{"Row", "Field", "Problem", "Value"})
	for _, row := range preview.Rows {
		for _, e := range row.Errors {
			_ = cw.Write([]string{
				fmt.Sprint(row.RowNo), e.Field, e.Message, row.Values[e.Field],
			})
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		writeProblem(w, r, err)
		return
	}

	name := "import-errors-" + preview.Job.ID + ".csv"
	if preview.Job.FileName != "" {
		name = "errors-" + preview.Job.FileName + ".csv"
	}
	sendFile(w, name, "text/csv; charset=utf-8", []byte(body.String()))
}

func (s *Server) handleCommitImport(w http.ResponseWriter, r *http.Request) {
	var req service.CommitRequest
	if r.ContentLength > 0 {
		if err := decodeJSON(w, r, &req); err != nil {
			writeProblem(w, r, err)
			return
		}
	}
	s.postOnce(w, r, func() (any, error) {
		return s.imports.Commit(r.Context(), r.PathValue("id"), req)
	})
}

func (s *Server) handleCancelImport(w http.ResponseWriter, r *http.Request) {
	job, err := s.imports.Cancel(r.Context(), r.PathValue("id"))
	if err != nil {
		writeProblem(w, r, err)
		return
	}
	writeJSON(w, job)
}
