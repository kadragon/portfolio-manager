package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
	"github.com/kadragon/portfolio-manager/internal/web/handlers"
)

func setupGroups(t *testing.T) (*echo.Echo, *container.Container) {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	c := container.NewWithQueries(sqlDB, q)
	e := echo.New()
	handlers.NewGroupHandler(c).Register(e)
	return e, c
}

func do(e *echo.Echo, method, target string, form url.Values) *httptest.ResponseRecorder {
	var body string
	if form != nil {
		body = form.Encode()
	}
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if form != nil {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestGroupsListEmpty(t *testing.T) {
	e, _ := setupGroups(t)
	rec := do(e, http.MethodGet, "/groups", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "그룹이 없습니다. 아래에서 새 그룹을 추가하세요.") {
		t.Errorf("empty state missing:\n%s", body)
	}
	if !strings.Contains(body, `<tbody id="groups-body">`) {
		t.Errorf("groups-body table missing")
	}
	// The groups nav item is the active one on this page.
	if !strings.Contains(body, `aria-current="page"`) {
		t.Errorf("active nav marker missing")
	}
}

func TestGroupCreateRendersRow(t *testing.T) {
	e, c := setupGroups(t)
	rec := do(e, http.MethodPost, "/groups", url.Values{
		"name":              {"성장주"},
		"target_percentage": {"30"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<td>성장주</td>") {
		t.Errorf("name cell missing:\n%s", body)
	}
	if !strings.Contains(body, "30.0%") {
		t.Errorf("target percent cell missing (want 30.0%%):\n%s", body)
	}

	groups, _ := c.Groups.ListAll(context.Background())
	if len(groups) != 1 {
		t.Fatalf("expected 1 group persisted, got %d", len(groups))
	}
	id := groups[0].ID.String()
	if !strings.Contains(body, `id="group-`+id+`"`) {
		t.Errorf("row id group-%s missing:\n%s", id, body)
	}
}

func TestGroupCreateTrimsName(t *testing.T) {
	e, c := setupGroups(t)
	do(e, http.MethodPost, "/groups", url.Values{"name": {"  공백  "}, "target_percentage": {"0"}})
	groups, _ := c.Groups.ListAll(context.Background())
	if len(groups) != 1 || groups[0].Name != "공백" {
		t.Fatalf("name not trimmed: %+v", groups)
	}
}

func TestGroupCreateDefaultTarget(t *testing.T) {
	// target_percentage absent -> defaults to 0.0 (FastAPI Form(0.0)).
	e, c := setupGroups(t)
	rec := do(e, http.MethodPost, "/groups", url.Values{"name": {"x"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	groups, _ := c.Groups.ListAll(context.Background())
	if len(groups) != 1 || groups[0].TargetPercentage != 0.0 {
		t.Fatalf("default target wrong: %+v", groups)
	}
}

func TestGroupCreateInvalidTargetIs422(t *testing.T) {
	e, _ := setupGroups(t)
	rec := do(e, http.MethodPost, "/groups", url.Values{
		"name":              {"x"},
		"target_percentage": {"not-a-number"},
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

func TestGroupCreateMissingNameIs422(t *testing.T) {
	e, _ := setupGroups(t)
	rec := do(e, http.MethodPost, "/groups", url.Values{"target_percentage": {"5"}})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

func TestGroupEditFormThenUpdate(t *testing.T) {
	e, c := setupGroups(t)
	created, _ := c.Groups.Create(context.Background(), "old", 10.0)
	id := created.ID.String()

	rec := do(e, http.MethodGet, "/groups/"+id+"/edit", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("edit status = %d", rec.Code)
	}
	form := rec.Body.String()
	if !strings.Contains(form, `value="old"`) {
		t.Errorf("edit form missing name value:\n%s", form)
	}
	if !strings.Contains(form, `hx-put="/groups/`+id+`"`) {
		t.Errorf("edit form missing hx-put:\n%s", form)
	}

	rec = do(e, http.MethodPut, "/groups/"+id, url.Values{
		"name":              {"new"},
		"target_percentage": {"45.5"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d", rec.Code)
	}
	row := rec.Body.String()
	if !strings.Contains(row, "<td>new</td>") || !strings.Contains(row, "45.5%") {
		t.Errorf("updated row wrong:\n%s", row)
	}
}

func TestGroupRowAndDelete(t *testing.T) {
	e, c := setupGroups(t)
	created, _ := c.Groups.Create(context.Background(), "deldme", 5.0)
	id := created.ID.String()

	rec := do(e, http.MethodGet, "/groups/"+id, nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "<td>deldme</td>") {
		t.Fatalf("row fetch failed: %d %s", rec.Code, rec.Body.String())
	}

	rec = do(e, http.MethodDelete, "/groups/"+id, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d", rec.Code)
	}
	groups, _ := c.Groups.ListAll(context.Background())
	if len(groups) != 0 {
		t.Fatalf("expected 0 after delete, got %d", len(groups))
	}
}

func TestGroupBadUUIDIs422(t *testing.T) {
	e, _ := setupGroups(t)
	rec := do(e, http.MethodGet, "/groups/not-a-uuid/edit", nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

func TestGroupRowNotFound(t *testing.T) {
	e, _ := setupGroups(t)
	nonexistent := uuidx.New()
	rec := do(e, http.MethodGet, "/groups/"+nonexistent.String(), nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGroupDeleteBadUUID(t *testing.T) {
	e, _ := setupGroups(t)
	rec := do(e, http.MethodDelete, "/groups/bad-uuid", nil)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}
